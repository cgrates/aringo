/*
Released under MIT License <http://www.opensource.org/licenses/mit-license.php
Copyright (C) ITsysCOM GmbH. All Rights Reserved.

Provides Asterisk ARI connector from Go programming language.
*/

package aringo

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

func TestAringoFib(t *testing.T) {
	fib := Fib()
	expected := 1 * time.Second
	f := fib()
	if expected != f {
		t.Fatalf("\nExpected: <%+v>,\nReceived: <%+v>", expected, f)
	}
	expected = 1 * time.Second
	f = fib()
	if expected != f {
		t.Fatalf("\nExpected: <%+v>,\nReceived: <%+v>", expected, f)
	}
	expected = 2 * time.Second
	f = fib()
	if expected != f {
		t.Fatalf("\nExpected: <%+v>,\nReceived: <%+v>", expected, f)
	}
	expected = 3 * time.Second
	f = fib()
	if expected != f {
		t.Fatalf("\nExpected: <%+v>,\nReceived: <%+v>", expected, f)
	}
	expected = 5 * time.Second
	f = fib()
	if expected != f {
		t.Fatalf("\nExpected: <%+v>,\nReceived: <%+v>", expected, f)
	}
}

func TestNewErrUnexpectedReplyCode(t *testing.T) {
	statusCode := 111
	expected := fmt.Sprintf("UNEXPECTED_REPLY_CODE: %d", statusCode)
	received := NewErrUnexpectedReplyCode(statusCode)
	if expected != received.Error() {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", expected, received)
	}
}

func TestAringoNewARInGONoConnAttempts(t *testing.T) {
	wsUrl := ""
	wsOrigin := ""
	username := ""
	password := ""
	userAgent := ""
	evChannel := make(chan map[string]interface{})
	errChannel := make(chan error)
	stopChan := make(<-chan struct{})
	connectAttempts := 0
	reconnects := -1

	experr := ErrZeroConnectAttempts
	received, err := NewARInGO(wsUrl, wsOrigin, username, password, userAgent, evChannel, errChannel, stopChan, connectAttempts, reconnects, 0)

	if err != experr {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", experr, err)
	} else if received != nil {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", nil, received)
	}
}

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
	received, err := NewARInGO(wsUrl, wsOrigin, username, password, userAgent, evChannel, errChannel, stopChan, connectAttempts, reconnects, 0)
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

	received, err = NewARInGO(wsUrl, wsOrigin, username, password, userAgent, evChannel, errChannel, stopChan, connectAttempts, reconnects, 0)

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
	if len(ari.errChannel) != 0 {
		t.Fatalf("\nExpected: <%+v>, \nReceived: <%+v>", 0, len(ari.errChannel))
	}

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
	if len(ari.errChannel) != 0 {
		t.Fatalf("\nExpected: <%+v>, \nReceived: <%+v>", 0, len(ari.errChannel))
	}

	if len(ari.evChannel) != 0 {
		t.Fatalf("\nExpected: <%+v>, \nReceived: <%+v>", 0, len(ari.evChannel))
	}
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
	if len(ari.errChannel) != 0 {
		t.Fatalf("\nExpected: <%+v>, \nReceived: <%+v>", 0, len(ari.errChannel))
	}

	if len(ari.evChannel) != 0 {
		t.Fatalf("\nExpected: <%+v>, \nReceived: <%+v>", 0, len(ari.evChannel))
	}
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
	if len(ari.errChannel) != 0 {
		t.Fatalf("\nExpected: <%+v>, \nReceived: <%+v>", 0, len(ari.errChannel))
	}

	if len(ari.evChannel) != 0 {
		t.Fatalf("\nExpected: <%+v>, \nReceived: <%+v>", 0, len(ari.evChannel))
	}

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

	if len(ari.errChannel) != 1 {
		t.Fatalf("\nExpected: <%+v>, \nReceived: <%+v>", 1, len(ari.errChannel))
	}

	if len(ari.evChannel) != 0 {
		t.Fatalf("\nExpected: <%+v>, \nReceived: <%+v>", 0, len(ari.evChannel))
	}

	exp := "invalid character 'i' looking for beginning of value"
	rcv := <-ari.errChannel

	if exp != rcv.Error() {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", exp, rcv)
	}

	close(stopChan)
}

func TestAringoCallUnrecognizedMethod(t *testing.T) {
	ari := &ARInGO{}
	var data url.Values

	experr := "Unrecognized method: invalid"
	rply, err := ari.Call("invalid", ari.wsURL, data)

	if err == nil || err.Error() != experr {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", experr, err)
	} else if rply != nil {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", nil, rply)
	}
}

func TestAringoCallInvalidURL(t *testing.T) {
	ari := &ARInGO{}
	var data url.Values

	experr := "parse \":foo\": missing protocol scheme"
	rply, err := ari.Call(HTTP_POST, ":foo", data)

	if err == nil || err.Error() != experr {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", experr, err)
	} else if rply != nil {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", nil, rply)
	}
}

func TestAringoCallSuccess(t *testing.T) {
	stopChan := make(chan struct{})
	var srv *httptest.Server
	ari := &ARInGO{
		httpClient:     http.DefaultClient,
		reconnects:     -1,
		evChannel:      make(chan map[string]interface{}),
		errChannel:     make(chan error),
		wsListenerExit: stopChan,
	}

	data := url.Values{}
	data.Set("test", "string")
	expected := []byte("OK")
	srv = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {

		rcv, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(data, rcv) {
			t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", data, rcv)
		}
		rw.Write(expected)
	}))
	defer srv.Close()

	received, err := ari.Call(HTTP_GET, srv.URL, data)

	if err != nil {
		t.Fatalf("\nExpected: <%+v>, \nReceived: <%+v>", nil, err)
	}

	if !reflect.DeepEqual(received, expected) {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", expected, received)
	}
}

func TestAringoCallNoGET(t *testing.T) {
	stopChan := make(chan struct{})
	var srv *httptest.Server
	ari := &ARInGO{
		httpClient:     http.DefaultClient,
		reconnects:     -1,
		evChannel:      make(chan map[string]interface{}),
		errChannel:     make(chan error),
		wsListenerExit: stopChan,
	}
	data := url.Values{
		"test": {"string"},
	}

	srv = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		expdata := []byte(data.Encode())
		rcv, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			return
		}

		if !reflect.DeepEqual(expdata, rcv) {
			t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", expdata, rcv)
		}
		rw.Write([]byte("OK"))
	}))
	defer srv.Close()

	rcv, err := ari.Call(HTTP_POST, srv.URL, data)

	if err != nil {
		t.Fatalf("\nExpected: <%+v>, \nReceived: <%+v>", nil, err)
	}

	if len(rcv) != 0 {
		t.Errorf("\nExpected empty reply, \nReceived: <%+v>", rcv)
	}
}

func TestAringoCallDoErr(t *testing.T) {
	stopChan := make(chan struct{})

	ari := &ARInGO{
		httpClient:     http.DefaultClient,
		reconnects:     -1,
		evChannel:      make(chan map[string]interface{}),
		errChannel:     make(chan error),
		wsListenerExit: stopChan,
	}

	data := url.Values{
		"test": {"string"},
	}

	experr := "Post \"\": unsupported protocol scheme \"\""
	_, err := ari.Call(HTTP_POST, "", data)

	if err == nil || err.Error() != experr {
		t.Fatalf("\nExpected: <%+v>, \nReceived: <%+v>", experr, err)
	}
}

func TestAringoCall204(t *testing.T) {
	stopChan := make(chan struct{})
	var srv *httptest.Server

	ari := &ARInGO{
		reconnects:     -1,
		evChannel:      make(chan map[string]interface{}),
		errChannel:     make(chan error),
		wsListenerExit: stopChan,
	}
	var data url.Values

	srv = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(204)
		rw.Write([]byte("OK"))
	}))
	defer srv.Close()
	ari.httpClient = http.DefaultClient

	rcv, err := ari.Call(HTTP_POST, srv.URL, data)

	if err != nil {
		t.Fatalf("\nExpected: <%+v>, \nReceived: <%+v>", nil, err)
	}

	if len(rcv) != 0 {
		t.Errorf("\nExpected empty slice, \nReceived: <%+v>", rcv)
	}
}

func TestAringoCallNot200(t *testing.T) {
	stopChan := make(chan struct{})
	var srv *httptest.Server

	ari := &ARInGO{
		reconnects:     -1,
		evChannel:      make(chan map[string]interface{}),
		errChannel:     make(chan error),
		wsListenerExit: stopChan,
	}

	srv = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(201)
	}))
	defer srv.Close()
	ari.httpClient = srv.Client()

	var data url.Values
	experr := "UNEXPECTED_REPLY_CODE: 201"
	_, err := ari.Call(HTTP_POST, srv.URL, data)

	if err == nil || err.Error() != experr {
		t.Fatalf("\nExpected: <%+v>, \nReceived: <%+v>", experr, err)
	}

}
