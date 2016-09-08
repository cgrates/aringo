/*
Released under MIT License <http://www.opensource.org/licenses/mit-license.php
Copyright (C) ITsysCOM GmbH. All Rights Reserved.

Provides Asterisk ARI connector from Go programming language.
*/

package aringo

import (
	"net/http"
	"net/url"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

// successive Fibonacci numbers.
func Fib() func() time.Duration {
	a, b := 0, 1
	return func() time.Duration {
		a, b = b, a+b
		return time.Duration(a) * time.Second
	}
}

func NewARInGO(wsUrl string, reconnects int) (*ARInGO, error) {
	ari := &ARInGO{httpClient: new(http.Client), wsUrl: wsUrl, reconnects: reconnects, wsMux: new(sync.RWMutex)}
	if err := ari.Connect(); err != nil {
		return nil, err
	}
	return ari, nil
}

// ARInGO represents one ARI connection/application
type ARInGO struct {
	httpClient *http.Client
	wsUrl      string
	ws         *websocket.Conn
	reconnects int
	wsMux      *sync.RWMutex
}

func (ari *ARInGO) connectWS() (err error) {
	ari.wsMux.Lock()
	defer ari.wsMux.Unlock()
	ari.ws, err = websocket.Dial(ari.wsUrl, "", "http://localhost")
	if err != nil {
		return
	}
	return nil
}

func (ari *ARInGO) Connect() error {
	if err := ari.connectWS(); err != nil {
		return err
	}
	return nil
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
