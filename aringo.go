/*
Released under MIT License <http://www.opensource.org/licenses/mit-license.php
Copyright (C) ITsysCOM GmbH. All Rights Reserved.

Provides Asterisk ARI connector from Go programming language.
*/

package aringo

import (
	"bytes"
	"errors"
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
)

var (
	ErrZeroConnectAttempts = errors.New("ZERO_CONNECT_ATTEMPTS")
)

// Fib returns successive Fibonacci numbers.
func Fib() func() time.Duration {
	a, b := 0, 1
	return func() time.Duration {
		a, b = b, a+b
		return time.Duration(a) * time.Second
	}
}

func NewErrUnexpectedReplyCode(statusCode int) error {
	return fmt.Errorf("UNEXPECTED_REPLY_CODE: %d", statusCode)
}

func NewARInGO(wsUrl, wsOrigin, username, password, userAgent string,
	evChannel chan map[string]interface{}, errChannel chan error, stopChan <-chan struct{}, connectAttempts, reconnects int) (ari *ARInGO, err error) {
	if connectAttempts == 0 {
		return nil, ErrZeroConnectAttempts
	}
	ari = &ARInGO{
		httpClient:     new(http.Client),
		wsURL:          wsUrl,
		wsOrigin:       wsOrigin,
		username:       username,
		password:       password,
		userAgent:      userAgent,
		reconnects:     reconnects,
		evChannel:      evChannel,
		errChannel:     errChannel,
		wsListenerExit: stopChan,
	}
	if err = ari.connect(); err != nil {
		delay := Fib()
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
	httpClient     *http.Client
	wsURL          string
	wsOrigin       string
	username       string
	password       string
	userAgent      string
	ws             *websocket.Conn
	reconnects     int
	evChannel      chan map[string]interface{} // Events coming from Asterisk are posted here
	errChannel     chan error                  // Errors are posted here
	wsListenerExit <-chan struct{}             // Signal dispatcher to stop listening
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
		var ev map[string]interface{}
		if err := websocket.JSON.Receive(ari.ws, &ev); err != nil {
			ari.disconnect()
			select {
			case <-ari.wsListenerExit:
				return // if the chanel was closed already do not try to reconnect
			default:
			}
			if errConn := ari.connect(); errConn != nil { // give up on success since another goroutine will pick up events
				delay := Fib()
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
func (ari *ARInGO) connect() (err error) {
	if ari.ws, err = websocket.Dial(ari.wsURL, "", ari.wsOrigin); err != nil {
		return
	}
	// Connected, start listener
	go ari.wsEventListener()
	return
}

func (ari *ARInGO) disconnect() error {
	return ari.ws.Close()
}

// Call represents one REST call to Asterisk using httpClient call
// If there is a reply from Asterisk it should be in form map[string]interface{}
func (ari *ARInGO) Call(method, reqURL string, data url.Values) (reply []byte, err error) {
	var reqBody io.Reader
	switch method {
	case HTTP_GET: // Add data inside url
		u, _ := url.ParseRequestURI(reqURL)
		u.RawQuery = data.Encode()
		reqURL = u.String()
	case HTTP_POST, HTTP_DELETE:
		reqBody = bytes.NewBufferString(data.Encode())
	default:
		err = fmt.Errorf("Unrecognized method: %s", method)
		return
	}
	var req *http.Request
	if req, err = http.NewRequest(method, reqURL, reqBody); err != nil {
		return
	}
	req.Header.Set("User-Agent", ari.userAgent)
	req.SetBasicAuth(ari.username, ari.password)
	var resp *http.Response
	if resp, err = ari.httpClient.Do(req); err != nil {
		return
	}
	if resp.StatusCode == 204 { // No content status code
		return
	}
	if resp.StatusCode != 200 {
		err = NewErrUnexpectedReplyCode(resp.StatusCode)
		return
	}
	var respBody []byte
	respBody, err = io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil ||
		method != HTTP_GET {
		return
	}
	reply = respBody
	return

}
