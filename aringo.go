/*
Released under MIT License <http://www.opensource.org/licenses/mit-license.php
Copyright (C) ITsysCOM GmbH. All Rights Reserved.

Provides Asterisk ARI connector from Go programming language.
*/

package aringo

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sync"
	"time"

	"golang.org/x/net/websocket"
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

func NewARInGO(wsUrl, wsOrigin string, evChannel chan *json.RawMessage, errChannel chan error, connectAttempts, reconnects int) (ari *ARInGO, err error) {
	if connectAttempts == 0 {
		return nil, ErrZeroConnectAttempts
	}
	ari = &ARInGO{httpClient: new(http.Client), wsUrl: wsUrl, wsOrigin: wsOrigin, reconnects: reconnects,
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
	ws             *websocket.Conn
	reconnects     int
	wsMux          *sync.RWMutex
	evChannel      chan *json.RawMessage // Events coming from Asterisk are posted here
	errChannel     chan error            // Errors are posted here
	wsListenerExit chan struct{}         // Signal dispatcher to stop listening
	wsListenerMux  *sync.Mutex           // Use it to get access to wsListenerExit recreation
}

// wsDispatcher listens for JSON rawMessages and stores them into the evChannel
func (ari *ARInGO) wsEventListener(chanExit chan struct{}) {
	for {
		select {
		case <-chanExit:
			break
		default:
			var data json.RawMessage
			if err := websocket.JSON.Receive(ari.ws, &data); err != nil { // ToDo: Add reconnects here
				ari.disconnect()
				ari.errChannel <- err
				return
			}
			ari.evChannel <- &data
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

// Call represents one REST call to Asterisk using httpClient
// Returns a http.Response so we can process additional data out of it
func (ari *ARInGO) Call(url string, data url.Values, resp *http.Response) error {
	if reply, err := ari.httpClient.PostForm(url, data); err != nil {
		return err
	} else {
		*resp = *reply
	}
	return nil
}
