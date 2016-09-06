/*
Released under MIT License <http://www.opensource.org/licenses/mit-license.php
Copyright (C) ITsysCOM GmbH. All Rights Reserved.

Provides Asterisk ARI connector from Go programming language.
*/

package aringo

import (
	"net/http"
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

func NewARInGO(appName, httpUrl, wsUrl, wsOrigin string) *ARInGO {
	return &ARInGO{appName: appName, httpUrl: httpUrl, httpClient: new(http.Client),
		wsUrl: wsUrl, wsOrigin: wsOrigin, wsMux: new(sync.RWMutex)}
}

// ARInGO represents one ARI connection/application
type ARInGO struct {
	appName    string
	httpUrl    string
	httpClient *http.Client
	wsUrl      string
	wsOrigin   string
	ws         *websocket.Conn
	wsMux      *sync.RWMutex
}

func (ari *ARInGO) connectWS() (err error) {
	ari.wsMux.Lock()
	defer ari.wsMux.Unlock()
	ari.ws, err = websocket.Dial(ari.wsUrl, "", ari.wsOrigin)
	if err != nil {
		return err
	}
	return nil
}
