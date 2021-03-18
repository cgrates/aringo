// +build integration

/*
Released under MIT License <http://www.opensource.org/licenses/mit-license.php
Copyright (C) ITsysCOM GmbH. All Rights Reserved.

Provides Asterisk ARI connector from Go programming language.
*/

package aringo

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

func TestAringoNewARInGO(t *testing.T) {
	username := ""
	password := ""
	userAgent := ""
	evChannel := make(chan map[string]interface{})
	errChannel := make(chan error)
	stopChan := make(chan struct{})
	connectAttempts := -1
	reconnects := -1

	var srv *httptest.Server
	srv = httptest.NewServer(websocket.Handler(func(c *websocket.Conn) {

	}))

	n := strings.LastIndexByte(srv.URL, ':')
	wsOrigin := srv.URL[:n] + "/"
	wsUrl := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"

	expected := &ARInGO{
		httpClient:     http.DefaultClient,
		wsURL:          wsUrl,
		wsOrigin:       wsOrigin,
		username:       "",
		password:       "",
		userAgent:      userAgent,
		reconnects:     -1,
		evChannel:      evChannel,
		errChannel:     errChannel,
		wsListenerExit: stopChan,
	}
	received, err := NewARInGO(wsUrl, wsOrigin, username, password, userAgent, evChannel, errChannel, stopChan, connectAttempts, reconnects)
	expected.httpClient = received.httpClient
	expected.ws = received.ws

	if err != nil {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", nil, err)
	} else if !reflect.DeepEqual(received, expected) {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", expected, received)
	}

	close(stopChan)
	srv.Close()

	stopChan = make(chan struct{})
	expected.wsListenerExit = stopChan

	srv2 := &http.Server{
		Addr: strings.TrimPrefix(srv.URL, "http://"),
		Handler: websocket.Handler(func(c *websocket.Conn) {

		}),
	}
	defer srv2.Close()

	go func() {
		time.Sleep(10 * time.Millisecond)
		srv2.ListenAndServe()

	}()

	received, err = NewARInGO(wsUrl, wsOrigin, username, password, userAgent, evChannel, errChannel, stopChan, connectAttempts, reconnects)

	expected.httpClient = received.httpClient
	expected.ws = received.ws

	if err != nil {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", nil, err)
	} else if !reflect.DeepEqual(received, expected) {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", expected, received)
	}
	close(stopChan)
}

func TestAringowsEventListenerValidJSON(t *testing.T) {
	var srv *httptest.Server
	stopChan := make(chan struct{})
	srv = httptest.NewServer(websocket.Handler(func(c *websocket.Conn) {

		time.Sleep(10 * time.Millisecond)
		close(stopChan)
		c.Write([]byte("{\"key\":\"value\"}"))
		c.Close()
	}))

	defer srv.Close()

	n := strings.LastIndexByte(srv.URL, ':')
	wsOrigin := srv.URL[:n] + "/"
	wsUrl := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"

	ari := &ARInGO{
		httpClient:     new(http.Client),
		wsURL:          wsUrl,
		wsOrigin:       wsOrigin,
		username:       "",
		password:       "",
		userAgent:      "",
		reconnects:     -1,
		evChannel:      make(chan map[string]interface{}, 1),
		errChannel:     make(chan error, 1),
		wsListenerExit: stopChan,
	}

	var err error
	ari.ws, err = websocket.Dial(ari.wsURL, "", ari.wsOrigin)
	if err != nil {
		t.Fatal(err)
	}

	ari.wsEventListener()

	if len(ari.evChannel) != 1 {
		t.Fatalf("\nExpected: <%+v>, \nReceived: <%+v>", 1, len(ari.evChannel))
	}

	exp := map[string]interface{}{
		"key": "value",
	}
	rcv := <-ari.evChannel

	if !reflect.DeepEqual(exp, rcv) {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", exp, rcv)
	}
}

func TestAringowsEventListenerClosedCh(t *testing.T) {
	var srv *httptest.Server
	stopChan := make(chan struct{})
	srv = httptest.NewServer(websocket.Handler(func(c *websocket.Conn) {
		time.Sleep(10 * time.Millisecond)
		close(stopChan)
		c.Write([]byte("{key:value}"))
		c.Close()

	}))

	defer srv.Close()

	n := strings.LastIndexByte(srv.URL, ':')
	wsOrigin := srv.URL[:n] + "/"
	wsUrl := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"

	ari := &ARInGO{
		httpClient:     new(http.Client),
		wsURL:          wsUrl,
		wsOrigin:       wsOrigin,
		username:       "",
		password:       "",
		userAgent:      "",
		reconnects:     -1,
		evChannel:      make(chan map[string]interface{}, 1),
		errChannel:     make(chan error, 1),
		wsListenerExit: stopChan,
	}

	var err error
	ari.ws, err = websocket.Dial(ari.wsURL, "", ari.wsOrigin)
	if err != nil {
		t.Fatal(err)
	}

	ari.wsEventListener()
	// if len(ari.evChannel) != 0 {
	// 	t.Fatalf("\nExpected: <%+v>, \nReceived: <%+v>", 0, len(ari.evChannel))
	// }
}

func TestAringowsEventListenerReconnect(t *testing.T) {
	var srv *httptest.Server
	stopChan := make(chan struct{})
	srv = httptest.NewServer(websocket.Handler(func(c *websocket.Conn) {

	}))

	defer srv.Close()

	n := strings.LastIndexByte(srv.URL, ':')
	wsOrigin := srv.URL[:n] + "/"
	wsUrl := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"

	ari := &ARInGO{
		httpClient:     new(http.Client),
		wsURL:          wsUrl,
		wsOrigin:       wsOrigin,
		username:       "",
		password:       "",
		userAgent:      "",
		reconnects:     -1,
		evChannel:      make(chan map[string]interface{}, 1),
		errChannel:     make(chan error, 1),
		wsListenerExit: stopChan,
	}

	var err error
	ari.ws, err = websocket.Dial(ari.wsURL, "", ari.wsOrigin)
	if err != nil {
		t.Fatal(err)
	}

	ari.wsEventListener()
	// if len(ari.evChannel) != 0 {
	// 	t.Fatalf("\nExpected: <%+v>, \nReceived: <%+v>", 0, len(ari.evChannel))
	// }
}

// func TestAringowsEventListenerFailReconnect(t *testing.T) {
// 	var srv *httptest.Server
// 	srv = httptest.NewServer(websocket.Handler(func(c *websocket.Conn) {

// 	}))
// 	defer srv.Close()

// 	n := strings.LastIndexByte(srv.URL, ':')
// 	wsOrigin := srv.URL[:n] + "/"
// 	wsUrl := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"
// 	stopChan := make(chan struct{})

// 	ari := &ARInGO{
// 		httpClient:     new(http.Client),
// 		wsURL:          wsUrl,
// 		wsOrigin:       wsOrigin,
// 		username:       "",
// 		password:       "",
// 		userAgent:      "",
// 		reconnects:     -1,
// 		evChannel:      make(chan map[string]interface{}),
// 		errChannel:     make(chan error),
// 		wsListenerExit: stopChan,
// 	}

// 	var err error
// 	ari.ws, err = websocket.Dial(ari.wsURL, "", ari.wsOrigin)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	close(stopChan)
// 	ari.wsEventListener()

// }
