/*
Released under MIT License <http://www.opensource.org/licenses/mit-license.php
Copyright (C) ITsysCOM GmbH. All Rights Reserved.

Provides Asterisk ARI connector from Go programming language.
*/

package aringo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"

	"github.com/coder/websocket"
)

var (
	ErrZeroConnectAttempts = errors.New("ZERO_CONNECT_ATTEMPTS")
)

func NewErrUnexpectedReplyCode(statusCode int) error {
	return fmt.Errorf("UNEXPECTED_REPLY_CODE: %d", statusCode)
}

func NewARInGO(wsUrl, wsOrigin, username, password, userAgent string, evChannel chan map[string]interface{},
	errChannel chan error, stopChan <-chan struct{}, connectAttempts, reconnects int,
	maxReconnectInterval time.Duration, delayFunc func(time.Duration, time.Duration) func() time.Duration) (ari *ARInGO, err error) {
	if connectAttempts == 0 {
		return nil, ErrZeroConnectAttempts
	}
	ari = &ARInGO{
		httpClient:           new(http.Client),
		wsURL:                wsUrl,
		wsOrigin:             wsOrigin,
		username:             username,
		password:             password,
		userAgent:            userAgent,
		reconnects:           reconnects,
		maxReconnectInterval: maxReconnectInterval,
		delayFunc:            delayFunc,
		evChannel:            evChannel,
		errChannel:           errChannel,
		wsListenerExit:       stopChan,
	}
	if err = ari.connect(); err != nil {
		delay := ari.delayFunc(time.Second, 0)
		for i := 0; connectAttempts == -1 || i < connectAttempts-1; i++ { // -1 for infinite attempts
			time.Sleep(delay()) // Increased delay to randomize network load
			if err = ari.connect(); err == nil {
				return
			}
		}
	}
	return
}

// ARInGO represents one ARI connection/application
type ARInGO struct {
	httpClient           *http.Client
	wsURL                string
	wsOrigin             string
	username             string
	password             string
	userAgent            string
	ws                   *websocket.Conn
	reconnects           int
	maxReconnectInterval time.Duration
	delayFunc            func(time.Duration, time.Duration) func() time.Duration // used to create/reset the delay function
	evChannel            chan map[string]any                                     // Events coming from Asterisk are posted here
	errChannel           chan error                                              // Errors are posted here
	wsListenerExit       <-chan struct{}                                         // Signal dispatcher to stop listening
	pendingMu            sync.Mutex                                              // Protects pending map
	pending              map[string]chan RESTResponse                            // request_id -> typed RESTResponse
}

// RESTRequest represents an ARI REST-over-WebSocket request envelope.
type RESTRequest struct {
	Type          string        `json:"type"` // must be "RESTRequest"
	TransactionID string        `json:"transaction_id,omitempty"`
	RequestID     string        `json:"request_id"`
	Method        string        `json:"method"`
	URI           string        `json:"uri"`
	ContentType   string        `json:"content_type,omitempty"`
	QueryStrings  []QueryString `json:"query_strings,omitempty"`
	MessageBody   string        `json:"message_body,omitempty"`
}

type QueryString struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// RESTResponse represents an ARI REST-over-WebSocket response envelope.
type RESTResponse struct {
	Type          string `json:"type"`
	TransactionID string `json:"transaction_id,omitempty"`
	RequestID     string `json:"request_id"`
	StatusCode    int    `json:"status_code"`
	ReasonPhrase  string `json:"reason_phrase,omitempty"`
	ContentType   string `json:"content_type,omitempty"`
	Uri           string `json:"uri,omitempty"`
	MessageBody   string `json:"message_body,omitempty"`
	Timestamp     string `json:"timestamp,omitempty"`
	AsteriskID    string `json:"asterisk_id,omitempty"`
	Application   string `json:"application,omitempty"`
}

// wsDispatcher listens for JSON rawMessages and stores them into the evChannel
func (ari *ARInGO) wsEventListener() {
	for {
		select {
		case <-ari.wsListenerExit:
			ari.disconnect()
			return
		default:
		}
		var ev map[string]any
		_, data, err := ari.ws.Read(context.Background())
		if err != nil {
			ari.disconnect()
			select {
			case <-ari.wsListenerExit:
				return // if the chanel was closed already do not try to reconnect
			default:
			}
			if errConn := ari.connect(); errConn != nil { // give up on success since another goroutine will pick up events
				delay := ari.delayFunc(time.Second, ari.maxReconnectInterval)
				for i := 0; i < ari.reconnects-1; i++ { // attempt reconnect
					time.Sleep(delay())
					if errConn := ari.connect(); errConn == nil { // give up on success since another goroutine will pick up events
						return
					}
				}
				// reconnect did not succeed, pass the original error and give up
				ari.errChannel <- err
			}
			return
		}
		if err := json.Unmarshal(data, &ev); err != nil {
			continue
		}
		// If this is a RESTResponse, try to correlate and resolve waiter
		if t, ok := ev["type"].(string); ok && t == "RESTResponse" {
			var rr RESTResponse
			if err := json.Unmarshal(data, &rr); err != nil {
				continue
			}
			ari.pendingMu.Lock()
			ch, exists := ari.pending[rr.RequestID]
			if exists {
				delete(ari.pending, rr.RequestID)
			}
			ari.pendingMu.Unlock()
			if exists {
				ch <- rr
				close(ch)
				continue
			}

		}
		ari.evChannel <- ev
	}
}

// connect connects to Asterisk Websocket and starts listener
func (ari *ARInGO) connect() (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var c *websocket.Conn
	if c, _, err = websocket.Dial(ctx, ari.wsURL, nil); err != nil {
		return
	}
	ari.ws = c
	if ari.pending == nil {
		ari.pending = make(map[string]chan RESTResponse)
	}
	// Connected, start listener
	go ari.wsEventListener()
	return
}

func (ari *ARInGO) disconnect() error {
	if ari.ws == nil {
		return nil
	}
	return ari.ws.Close(websocket.StatusNormalClosure, "")
}

// Call sends a REST request over WebSocket with optional query params and JSON body.
func (ari *ARInGO) Call(method, uri string, queryStr map[string]string, body []byte) (RESTResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if ari.ws == nil {
		return RESTResponse{}, errors.New("websocket not connected")
	}
	requestID := fmt.Sprintf("%d", time.Now().UnixNano())
	transactionID := uuid.New().String()
	var qs []QueryString
	for k, val := range queryStr {
		qs = append(qs, QueryString{Name: k, Value: val})
	}
	rr := RESTRequest{
		Type:          "RESTRequest",
		TransactionID: transactionID,
		RequestID:     requestID,
		Method:        method,
		URI:           uri,
		QueryStrings:  qs,
	}
	if len(body) > 0 {
		rr.ContentType = "application/json"
		rr.MessageBody = string(body)
	}
	respCh := make(chan RESTResponse, 1)
	ari.pendingMu.Lock()
	ari.pending[requestID] = respCh
	ari.pendingMu.Unlock()
	if err := wsjson.Write(ctx, ari.ws, rr); err != nil {
		ari.pendingMu.Lock()
		delete(ari.pending, requestID)
		ari.pendingMu.Unlock()
		return RESTResponse{}, err
	}

	// Wait for response or context cancellation
	select {
	case resp := <-respCh:
		return resp, nil
	case <-ctx.Done():
		ari.pendingMu.Lock()
		delete(ari.pending, requestID)
		ari.pendingMu.Unlock()
		return RESTResponse{}, ctx.Err()
	}
}
