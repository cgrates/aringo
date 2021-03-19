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
	"net/url"
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

func TestAringowsEventListenerInvalidJSONReturn(t *testing.T) {
	stopChan := make(chan struct{})
	var srv *httptest.Server

	ari := &ARInGO{
		httpClient:     new(http.Client),
		username:       "",
		password:       "",
		userAgent:      "",
		reconnects:     100,
		evChannel:      make(chan map[string]interface{}),
		errChannel:     make(chan error, 1),
		wsListenerExit: stopChan,
	}
	srv = httptest.NewServer(websocket.Handler(func(c *websocket.Conn) {
		urll := ari.wsURL
		ari.wsURL = "invalidURL"
		c.Write([]byte("invalid"))
		time.Sleep(20 * time.Millisecond)
		ari.wsURL = urll
		c.Close()

	}))
	defer srv.Close()

	n := strings.LastIndexByte(srv.URL, ':')
	ari.wsOrigin = srv.URL[:n] + "/"
	ari.wsURL = "ws" + strings.TrimPrefix(srv.URL, "http") + "/"

	var err error
	ari.ws, err = websocket.Dial(ari.wsURL, "", ari.wsOrigin)
	if err != nil {
		t.Fatal(err)
	}

	ari.wsEventListener()
	close(stopChan)
}

func TestAringowsEventListenerFailReconnect(t *testing.T) {
	stopChan := make(chan struct{})
	var srv *httptest.Server

	ari := &ARInGO{
		httpClient:     new(http.Client),
		username:       "",
		password:       "",
		userAgent:      "",
		reconnects:     0,
		evChannel:      make(chan map[string]interface{}),
		errChannel:     make(chan error, 1),
		wsListenerExit: stopChan,
	}
	srv = httptest.NewServer(websocket.Handler(func(c *websocket.Conn) {
		urll := ari.wsURL
		ari.wsURL = "invalidURL"
		c.Write([]byte("invalid"))
		time.Sleep(20 * time.Millisecond)
		ari.wsURL = urll
		c.Close()

	}))
	defer srv.Close()

	n := strings.LastIndexByte(srv.URL, ':')
	ari.wsOrigin = srv.URL[:n] + "/"
	ari.wsURL = "ws" + strings.TrimPrefix(srv.URL, "http") + "/"

	var err error
	ari.ws, err = websocket.Dial(ari.wsURL, "", ari.wsOrigin)
	if err != nil {
		t.Fatal(err)
	}
	ari.wsEventListener()
	exp := "invalid character 'i' looking for beginning of value"
	rcv := <-ari.errChannel

	if exp != rcv.Error() {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", exp, rcv)
	}
	close(stopChan)
}

func TestAringoCallUnrecognizedMethod(t *testing.T) {
	stopChan := make(chan struct{})
	var srv *httptest.Server
	ari := &ARInGO{
		httpClient:     new(http.Client),
		username:       "",
		password:       "",
		userAgent:      "",
		reconnects:     -1,
		evChannel:      make(chan map[string]interface{}),
		errChannel:     make(chan error),
		wsListenerExit: stopChan,
	}
	srv = httptest.NewServer(websocket.Handler(func(c *websocket.Conn) {

	}))
	defer srv.Close()

	n := strings.LastIndexByte(srv.URL, ':')
	ari.wsOrigin = srv.URL[:n] + "/"
	ari.wsURL = "ws" + strings.TrimPrefix(srv.URL, "http") + "/"

	data := url.Values{}
	data.Set("test", "string")
	data.Add("key", "value1")
	data.Add("key", "value2")
	data.Add("key", "value3")
	experr := "Unrecognized method: invalid"
	rply, err := ari.Call("invalid", ari.wsURL, data)

	if err == nil || err.Error() != experr {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", experr, err)
	} else if rply != nil {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", nil, rply)
	}
}

func TestAringoCallInvalidURL(t *testing.T) {
	stopChan := make(chan struct{})
	var srv *httptest.Server
	ari := &ARInGO{
		httpClient:     new(http.Client),
		username:       "",
		password:       "",
		userAgent:      "",
		reconnects:     -1,
		evChannel:      make(chan map[string]interface{}),
		errChannel:     make(chan error),
		wsListenerExit: stopChan,
	}
	srv = httptest.NewServer(websocket.Handler(func(c *websocket.Conn) {

	}))
	defer srv.Close()

	n := strings.LastIndexByte(srv.URL, ':')
	ari.wsOrigin = srv.URL[:n] + "/"
	ari.wsURL = "ws" + strings.TrimPrefix(srv.URL, "http") + "/"

	data := url.Values{}
	data.Set("test", "string")
	data.Add("key", "value1")
	data.Add("key", "value2")
	data.Add("key", "value3")
	experr := "parse \":foo\": missing protocol scheme"
	rply, err := ari.Call(HTTP_POST, ":foo", data)

	if err == nil || err.Error() != experr {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", experr, err)
	} else if rply != nil {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", nil, rply)
	}
}

func TestAringoCallUnexpectedReply(t *testing.T) {
	stopChan := make(chan struct{})
	var srv *httptest.Server
	ari := &ARInGO{
		httpClient:     new(http.Client),
		username:       "",
		password:       "",
		userAgent:      "",
		reconnects:     -1,
		evChannel:      make(chan map[string]interface{}),
		errChannel:     make(chan error),
		wsListenerExit: stopChan,
	}
	srv = httptest.NewServer(websocket.Handler(func(c *websocket.Conn) {

	}))
	defer srv.Close()

	n := strings.LastIndexByte(srv.URL, ':')
	ari.wsOrigin = srv.URL[:n] + "/"
	ari.wsURL = "ws" + strings.TrimPrefix(srv.URL, "http") + "/"

	data := url.Values{}
	data.Set("test", "string")
	data.Add("key", "value1")
	data.Add("key", "value2")
	data.Add("key", "value3")
	experr := "UNEXPECTED_REPLY_CODE: 405"
	rply, err := ari.Call(HTTP_DELETE, ari.wsOrigin, data)
	if err == nil || err.Error() != experr {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", experr, err)
	} else if rply != nil {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", nil, rply)
	}
}

func TestAringoCall1(t *testing.T) {
	stopChan := make(chan struct{})
	var srv *httptest.Server
	ari := &ARInGO{
		httpClient:     new(http.Client),
		username:       "",
		password:       "",
		userAgent:      "",
		reconnects:     -1,
		evChannel:      make(chan map[string]interface{}),
		errChannel:     make(chan error),
		wsListenerExit: stopChan,
	}
	srv = httptest.NewServer(websocket.Handler(func(c *websocket.Conn) {

	}))
	defer srv.Close()

	n := strings.LastIndexByte(srv.URL, ':')
	ari.wsOrigin = srv.URL[:n] + "/"
	ari.wsURL = "ws" + strings.TrimPrefix(srv.URL, "http") + "/"

	data := url.Values{}
	data.Set("test", "string")
	data.Add("key", "value1")
	data.Add("key", "value2")
	data.Add("key", "value3")

	rply, err := ari.Call(HTTP_POST, ari.wsOrigin, data)
	if err != nil {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", nil, err)
	} else if rply != nil {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", nil, rply)
	}
}

// func TestAringoCall2(t *testing.T) {
// 	stopChan := make(chan struct{})
// 	var srv *httptest.Server
// 	ari := &ARInGO{
// 		httpClient:     new(http.Client),
// 		username:       "",
// 		password:       "",
// 		userAgent:      "",
// 		reconnects:     -1,
// 		evChannel:      make(chan map[string]interface{}),
// 		errChannel:     make(chan error),
// 		wsListenerExit: stopChan,
// 	}
// 	srv = httptest.NewServer(websocket.Handler(func(c *websocket.Conn) {

// 	}))
// 	defer srv.Close()

// 	n := strings.LastIndexByte(srv.URL, ':')
// 	ari.wsOrigin = srv.URL[:n] + "/"
// 	ari.wsURL = "ws" + strings.TrimPrefix(srv.URL, "http") + "/"

// 	data := url.Values{}
// 	data.Set("test", "string")
// 	data.Add("key", "value1")
// 	data.Add("key", "value2")
// 	data.Add("key", "value3")
// 	expected := make([]byte, 0)
// 	rply, err = ari.Call(HTTP_GET, ari.wsOrigin, data)
// 	// if err != nil {
// 	// 	t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", nil, err)
// 	// } else if rply != expected {
// 	// 	t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", nil, rply)
// 	// }
// }
