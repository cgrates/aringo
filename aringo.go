/*
Released under MIT License <http://www.opensource.org/licenses/mit-license.php
Copyright (C) ITsysCOM GmbH. All Rights Reserved.

Provides Asterisk ARI connector from Go programming language.
*/

package aringo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

const (
	HTTP_POST = "POST"
	HTTP_GET  = "GET"
)

var (
	ErrZeroConnectAttempts = errors.New("ZERO_CONNECT_ATTEMPTS")
)

// successive Fibonacci numbers.
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
	evChannel chan map[string]interface{}, errChannel chan error, connectAttempts, reconnects int) (ari *ARInGO, err error) {
	if connectAttempts == 0 {
		return nil, ErrZeroConnectAttempts
	}
	ari = &ARInGO{httpClient: new(http.Client), wsUrl: wsUrl, wsOrigin: wsOrigin,
		username: username, password: password, userAgent: userAgent, reconnects: reconnects,
		evChannel: evChannel, errChannel: errChannel,
		wsMux: new(sync.RWMutex), wsListenerMux: new(sync.Mutex)}
	delay := Fib()
	i := 0
	for {
		if err = ari.connect(); err == nil {
			break
		}
		i++
		if connectAttempts != -1 && i >= connectAttempts { // -1 for infinite attempts
			break
		}
		time.Sleep(delay()) // Increased delay to randomize network load
	}
	return
}

// ARInGO represents one ARI connection/application
type ARInGO struct {
	httpClient     *http.Client
	wsUrl          string
	wsOrigin       string
	username       string
	password       string
	userAgent      string
	ws             *websocket.Conn
	reconnects     int
	wsMux          *sync.RWMutex
	evChannel      chan map[string]interface{} // Events coming from Asterisk are posted here
	errChannel     chan error                  // Errors are posted here
	wsListenerExit chan struct{}               // Signal dispatcher to stop listening
	wsListenerMux  *sync.Mutex                 // Use it to get access to wsListenerExit recreation
}

// wsDispatcher listens for JSON rawMessages and stores them into the evChannel
func (ari *ARInGO) wsEventListener(chanExit chan struct{}) {
	for {
		select {
		case <-chanExit:
			break
		default:
			var ev map[string]interface{}
			if err := websocket.JSON.Receive(ari.ws, &ev); err != nil { // ToDo: Add reconnects here
				ari.disconnect()
				ari.errChannel <- err
				return
			}
			ari.evChannel <- ev
		}
	}
}

// connect connects to Asterisk Websocket and starts listener
func (ari *ARInGO) connect() (err error) {
	ari.wsMux.Lock()
	defer ari.wsMux.Unlock()
	ari.ws, err = websocket.Dial(ari.wsUrl, "", ari.wsOrigin)
	if err != nil {
		return
	}
	// Connected, start listener
	ari.wsListenerMux.Lock()
	if ari.wsListenerExit != nil {
		close(ari.wsListenerExit) // Order previous listener to stop before proceeding
	}
	ari.wsListenerExit = make(chan struct{})
	go ari.wsEventListener(ari.wsListenerExit)
	ari.wsListenerMux.Unlock()
	return nil
}

func (ari *ARInGO) disconnect() error {
	ari.wsListenerMux.Lock()
	close(ari.wsListenerExit) // Order previous listener to stop
	ari.wsListenerMux.Unlock()
	return ari.ws.Close()
}

// Call represents one REST call to Asterisk using httpClient call
// If there is a reply from Asterisk it should be in form map[string]interface{}
func (ari *ARInGO) Call(method, reqUrl string, data url.Values) (reply map[string]interface{}, err error) {
	var reqBody io.Reader
	switch method {
	case HTTP_GET: // Add data inside url
		u, _ := url.ParseRequestURI(reqUrl)
		u.RawQuery = data.Encode()
		reqUrl = u.String()
	case HTTP_POST:
		reqBody = bytes.NewBufferString(data.Encode())
	default:
		return nil, fmt.Errorf("Unrecognized method: %s", method)
	}
	req, err := http.NewRequest(method, reqUrl, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", ari.userAgent)
	req.SetBasicAuth(ari.username, ari.password)
	resp, err := ari.httpClient.Do(req)
	if err != nil {
		return nil, err
	} else if resp.StatusCode == 204 { // No content status code
		return nil, nil
	} else if resp.StatusCode != 200 {
		return nil, NewErrUnexpectedReplyCode(resp.StatusCode)
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if method != HTTP_GET {
		return nil, nil
	}
	if err := json.Unmarshal(respBody, reply); err != nil {
		return nil, err
	}
	return
}
