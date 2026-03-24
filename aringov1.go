/*
Released under MIT License <http://www.opensource.org/licenses/mit-license.php
Copyright (C) ITsysCOM GmbH. All Rights Reserved.

Provides Asterisk ARI connector from Go programming language.
*/

package aringo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/websocket"
)

const (
	HTTP_POST   = "POST"
	HTTP_GET    = "GET"
	HTTP_DELETE = "DELETE"
	HTTP_PUT    = "PUT"
)

func NewARInGOV1(wsUrl, wsOrigin, username, password, address, userAgent string, evChannel chan map[string]interface{},
	errChannel chan error, stopChan <-chan struct{}, connectAttempts, reconnects int,
	maxReconnectInterval time.Duration, delayFunc func(time.Duration, time.Duration) func() time.Duration) (ari *ARInGOV1, err error) {
	if connectAttempts == 0 {
		return nil, ErrZeroConnectAttempts
	}
	ari = &ARInGOV1{
		httpClient:           new(http.Client),
		wsURL:                wsUrl,
		wsOrigin:             wsOrigin,
		username:             username,
		password:             password,
		address:              address,
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

// ARInGOV1 represents one ARI connection/application
type ARInGOV1 struct {
	httpClient           *http.Client
	wsURL                string
	wsOrigin             string
	username             string
	password             string
	address              string
	userAgent            string
	ws                   *websocket.Conn
	reconnects           int
	maxReconnectInterval time.Duration
	delayFunc            func(time.Duration, time.Duration) func() time.Duration // used to create/reset the delay function
	evChannel            chan map[string]any                                     // Events coming from Asterisk are posted here
	errChannel           chan error                                              // Errors are posted here
	wsListenerExit       <-chan struct{}                                         // Signal dispatcher to stop listening
}

// wsDispatcher listens for JSON rawMessages and stores them into the evChannel
func (ari *ARInGOV1) wsEventListener() {
	for {
		select {
		case <-ari.wsListenerExit:
			ari.disconnect()
			return
		default:
		}
		var ev map[string]any
		if err := websocket.JSON.Receive(ari.ws, &ev); err != nil {
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
		ari.evChannel <- ev
	}
}

// connect connects to Asterisk Websocket and starts listener
func (ari *ARInGOV1) connect() (err error) {
	if ari.ws, err = websocket.Dial(ari.wsURL, "", ari.wsOrigin); err != nil {
		return
	}
	// Connected, start listener
	go ari.wsEventListener()
	return
}

func (ari *ARInGOV1) disconnect() error {
	return ari.ws.Close()
}

// Call represents one REST call to Asterisk using httpClient call
func (ari *ARInGOV1) Call(method, uri string, queryStr map[string]string, body []byte) (reply RESTResponse, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	fullURL := fmt.Sprintf("http://%s/ari/%s", ari.address, uri)
	if len(queryStr) > 0 {
		u, _ := url.Parse(fullURL)
		q := u.Query()
		for k, v := range queryStr {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
		fullURL = u.String()
	}
	var reqBody io.Reader
	if len(body) > 0 {
		reqBody = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return RESTResponse{}, err
	}
	req.Header.Set("User-Agent", ari.userAgent)
	req.SetBasicAuth(ari.username, ari.password)
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := ari.httpClient.Do(req)
	if err != nil {
		return RESTResponse{}, err
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	return RESTResponse{
		StatusCode:  resp.StatusCode,
		MessageBody: string(bodyBytes),
		Type:        "RESTResponse",
	}, nil
}
