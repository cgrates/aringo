/*
Released under MIT License <http://www.opensource.org/licenses/mit-license.php
Copyright (C) ITsysCOM GmbH. All Rights Reserved.

Provides Asterisk ARI connector from Go programming language.
*/

package aringo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	legacyws "golang.org/x/net/websocket"
)

func TestAringoFib(t *testing.T) {
	fib := fibDuration(time.Second, 0)
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

func TestFibDurationMaxDuration(t *testing.T) {
	maxDur := 3 * time.Second
	fib := fibDuration(time.Second, maxDur)

	fib()
	fib()
	fib()
	fib()
	f := fib()
	if f != maxDur {
		t.Errorf("\nExpected: <%+v>,\nReceived: <%+v>", maxDur, f)
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

func TestAringoNewARInGOV1NoConnAttempts(t *testing.T) {
	wsUrl := ""
	wsOrigin := ""
	username := ""
	address := ""
	password := ""
	userAgent := ""
	evChannel := make(chan map[string]any)
	errChannel := make(chan error)
	stopChan := make(<-chan struct{})
	connectAttempts := 0
	reconnects := -1

	experr := ErrZeroConnectAttempts
	received, err := NewARInGOV1(wsUrl, wsOrigin, username, password, address, userAgent, evChannel,
		errChannel, stopChan, connectAttempts, reconnects, 0, fibDuration)

	if err != experr {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", experr, err)
	} else if received != nil {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", nil, received)
	}
}

func TestAringoNewARInGOV1(t *testing.T) {
	username := ""
	password := ""
	userAgent := ""
	evChannel := make(chan map[string]interface{})
	errChannel := make(chan error)
	stopChan := make(chan struct{})
	connectAttempts := -1
	reconnects := -1

	var srv *httptest.Server
	srv = httptest.NewServer(legacyws.Handler(func(c *legacyws.Conn) {

	}))

	n := strings.LastIndexByte(srv.URL, ':')
	wsOrigin := srv.URL[:n] + "/"
	wsUrl := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"
	address := strings.TrimPrefix(srv.URL, "http://")
	expected := &ARInGOV1{
		httpClient:     http.DefaultClient,
		wsURL:          wsUrl,
		wsOrigin:       wsOrigin,
		username:       "",
		password:       "",
		address:        address,
		userAgent:      userAgent,
		reconnects:     -1,
		evChannel:      evChannel,
		errChannel:     errChannel,
		wsListenerExit: stopChan,
	}
	received, err := NewARInGOV1(wsUrl, wsOrigin, username, password, address, userAgent, evChannel,
		errChannel, stopChan, connectAttempts, reconnects, 0, fibDuration)
	expected.httpClient = received.httpClient
	expected.ws = received.ws
	received.delayFunc = nil

	if err != nil {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", nil, err)
	} else if !reflect.DeepEqual(received, expected) {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", expected, received)
	}

	close(stopChan)
	srv.Close()
}

func TestAringoV1wsEventListenerValidJSON(t *testing.T) {
	var srv *httptest.Server
	stopChan := make(chan struct{})
	srv = httptest.NewServer(legacyws.Handler(func(c *legacyws.Conn) {
		time.Sleep(10 * time.Millisecond)
		close(stopChan)
		c.Write([]byte("{\"key\":\"value\"}"))
		c.Close()
	}))

	defer srv.Close()

	n := strings.LastIndexByte(srv.URL, ':')
	wsOrigin := srv.URL[:n] + "/"
	wsUrl := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"

	ari := &ARInGOV1{
		httpClient:     new(http.Client),
		wsURL:          wsUrl,
		wsOrigin:       wsOrigin,
		username:       "",
		password:       "",
		userAgent:      "",
		reconnects:     -1,
		delayFunc:      fibDuration,
		evChannel:      make(chan map[string]any, 1),
		errChannel:     make(chan error, 1),
		wsListenerExit: stopChan,
	}

	var err error
	ari.ws, err = legacyws.Dial(ari.wsURL, "", ari.wsOrigin)
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

	exp := map[string]any{
		"key": "value",
	}
	rcv := <-ari.evChannel

	if !reflect.DeepEqual(exp, rcv) {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", exp, rcv)
	}
}

func TestAringoV1wsEventListenerClosedCh(t *testing.T) {
	var srv *httptest.Server
	stopChan := make(chan struct{})
	srv = httptest.NewServer(legacyws.Handler(func(c *legacyws.Conn) {
		time.Sleep(10 * time.Millisecond)
		close(stopChan)
		c.Write([]byte("{key:value}"))
		c.Close()
	}))

	defer srv.Close()

	n := strings.LastIndexByte(srv.URL, ':')
	wsOrigin := srv.URL[:n] + "/"
	wsUrl := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"

	ari := &ARInGOV1{
		httpClient:     new(http.Client),
		wsURL:          wsUrl,
		wsOrigin:       wsOrigin,
		username:       "",
		password:       "",
		userAgent:      "",
		reconnects:     -1,
		delayFunc:      fibDuration,
		evChannel:      make(chan map[string]interface{}, 1),
		errChannel:     make(chan error, 1),
		wsListenerExit: stopChan,
	}

	var err error
	ari.ws, err = legacyws.Dial(ari.wsURL, "", ari.wsOrigin)
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

func TestAringoV1wsEventListenerReconnect(t *testing.T) {
	var srv *httptest.Server
	stopChan := make(chan struct{})
	srv = httptest.NewServer(legacyws.Handler(func(c *legacyws.Conn) {

	}))

	defer srv.Close()

	n := strings.LastIndexByte(srv.URL, ':')
	wsOrigin := srv.URL[:n] + "/"
	wsUrl := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"

	ari := &ARInGOV1{
		httpClient:     new(http.Client),
		wsURL:          wsUrl,
		wsOrigin:       wsOrigin,
		username:       "",
		password:       "",
		userAgent:      "",
		reconnects:     -1,
		delayFunc:      fibDuration,
		evChannel:      make(chan map[string]interface{}, 1),
		errChannel:     make(chan error, 1),
		wsListenerExit: stopChan,
	}

	var err error
	ari.ws, err = legacyws.Dial(ari.wsURL, "", ari.wsOrigin)
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

func TestAringoV1wsEventListenerInvalidJSONReturn(t *testing.T) {
	stopChan := make(chan struct{})
	var srv *httptest.Server

	ari := &ARInGOV1{
		httpClient:     new(http.Client),
		username:       "",
		password:       "",
		userAgent:      "",
		reconnects:     100,
		delayFunc:      fibDuration,
		evChannel:      make(chan map[string]interface{}),
		errChannel:     make(chan error, 1),
		wsListenerExit: stopChan,
	}
	srv = httptest.NewServer(legacyws.Handler(func(c *legacyws.Conn) {
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
	ari.ws, err = legacyws.Dial(ari.wsURL, "", ari.wsOrigin)
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

func TestAringoV1wsEventListenerFailReconnect(t *testing.T) {
	stopChan := make(chan struct{})
	var srv *httptest.Server

	ari := &ARInGOV1{
		httpClient:     new(http.Client),
		username:       "",
		password:       "",
		userAgent:      "",
		reconnects:     0,
		delayFunc:      fibDuration,
		evChannel:      make(chan map[string]interface{}),
		errChannel:     make(chan error, 1),
		wsListenerExit: stopChan,
	}
	srv = httptest.NewServer(legacyws.Handler(func(c *legacyws.Conn) {
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
	ari.ws, err = legacyws.Dial(ari.wsURL, "", ari.wsOrigin)
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

func TestAringoV1CallGETSuccess(t *testing.T) {
	stopChan := make(chan struct{})
	defer close(stopChan)

	expectedBody := "OK"
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != HTTP_GET {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		if r.URL.Query().Get("test") != "string" {
			t.Errorf("Expected query param test=string, got %s", r.URL.Query().Get("test"))
		}
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(expectedBody))
	}))
	defer srv.Close()

	ari := &ARInGOV1{
		httpClient:     http.DefaultClient,
		address:        strings.TrimPrefix(srv.URL, "http://"),
		reconnects:     -1,
		delayFunc:      fibDuration,
		evChannel:      make(chan map[string]interface{}),
		errChannel:     make(chan error),
		wsListenerExit: stopChan,
	}

	resp, err := ari.Call(HTTP_GET, "", map[string]string{"test": "string"}, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	if resp.MessageBody != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, resp.MessageBody)
	}
}

func TestAringoV1CallPOSTSuccess(t *testing.T) {
	stopChan := make(chan struct{})
	defer close(stopChan)

	expectedBody := `{"status":"created"}`
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != HTTP_POST {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		vals := r.URL.Query()
		if vals.Get("channel") != "SIP/1000" {
			t.Errorf("Expected body param channel=SIP/1000, got %s", vals.Get("channel"))
		}
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(expectedBody))
	}))
	defer srv.Close()

	ari := &ARInGOV1{
		httpClient:     http.DefaultClient,
		address:        strings.TrimPrefix(srv.URL, "http://"),
		reconnects:     -1,
		delayFunc:      fibDuration,
		evChannel:      make(chan map[string]interface{}),
		errChannel:     make(chan error),
		wsListenerExit: stopChan,
	}

	resp, err := ari.Call(HTTP_POST, "", map[string]string{"channel": "SIP/1000"}, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestAringoV1CallDELETESuccess(t *testing.T) {
	stopChan := make(chan struct{})
	defer close(stopChan)

	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != HTTP_DELETE {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}
		rw.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	ari := &ARInGOV1{
		httpClient:     http.DefaultClient,
		address:        strings.TrimPrefix(srv.URL, "http://"),
		reconnects:     -1,
		delayFunc:      fibDuration,
		evChannel:      make(chan map[string]interface{}),
		errChannel:     make(chan error),
		wsListenerExit: stopChan,
	}

	resp, err := ari.Call(HTTP_DELETE, "channels/12345", nil, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", resp.StatusCode)
	}
}

func TestAringoV1Call404(t *testing.T) {
	stopChan := make(chan struct{})
	defer close(stopChan)

	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusNotFound)
		rw.Write([]byte(`{"message":"Channel not found"}`))
	}))
	defer srv.Close()

	ari := &ARInGOV1{
		httpClient:     http.DefaultClient,
		address:        strings.TrimPrefix(srv.URL, "http://"),
		reconnects:     -1,
		delayFunc:      fibDuration,
		evChannel:      make(chan map[string]interface{}),
		errChannel:     make(chan error),
		wsListenerExit: stopChan,
	}

	resp, err := ari.Call(HTTP_GET, "channels/nonexistent", nil, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestAringoV1CallWithQueryAndBody(t *testing.T) {
	stopChan := make(chan struct{})
	defer close(stopChan)

	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("app") != "myapp" {
			t.Errorf("Expected query param app=myapp, got %s", r.URL.Query().Get("app"))
		}

		vals := r.URL.Query()
		if vals.Get("endpoint") != "PJSIP/1000" {
			t.Errorf("Expected body param endpoint=PJSIP/1000, got %s", vals.Get("endpoint"))
		}
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{"id":"channel123"}`))
	}))
	defer srv.Close()

	ari := &ARInGOV1{
		httpClient:     http.DefaultClient,
		address:        strings.TrimPrefix(srv.URL, "http://"),
		reconnects:     -1,
		delayFunc:      fibDuration,
		evChannel:      make(chan map[string]interface{}),
		errChannel:     make(chan error),
		wsListenerExit: stopChan,
	}

	resp, err := ari.Call(HTTP_POST, "channels",
		map[string]string{"app": "myapp", "endpoint": "PJSIP/1000"}, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestNewARInGONoConnAttempts(t *testing.T) {
	wsUrl := ""
	wsOrigin := ""
	username := ""
	password := ""
	userAgent := ""
	evChannel := make(chan map[string]any)
	errChannel := make(chan error)
	stopChan := make(<-chan struct{})
	connectAttempts := 0
	reconnects := -1

	experr := ErrZeroConnectAttempts
	received, err := NewARInGO(wsUrl, wsOrigin, username, password, userAgent, evChannel,
		errChannel, stopChan, connectAttempts, reconnects, 0, fibDuration)

	if err != experr {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", experr, err)
	} else if received != nil {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", nil, received)
	}
}

func TestARInGOCallNoConnection(t *testing.T) {
	ari := &ARInGO{
		httpClient: new(http.Client),
		delayFunc:  fibDuration,
		pending:    make(map[string]chan RESTResponse),
	}

	_, err := ari.Call(HTTP_GET, "channels", nil, nil)
	if err == nil {
		t.Fatal("Expected error for no websocket connection")
	}
	if err.Error() != "websocket not connected" {
		t.Errorf("Expected 'websocket not connected' error, got: %v", err)
	}
}

func createTestWSServer(t *testing.T, handler func(ctx context.Context, conn *websocket.Conn)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Logf("websocket accept error: %v", err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")
		handler(r.Context(), conn)
	}))
}

func TestARInGOCallSuccess(t *testing.T) {
	requestReceived := make(chan RESTRequest, 1)

	srv := createTestWSServer(t, func(ctx context.Context, conn *websocket.Conn) {
		_, data, err := conn.Read(ctx)
		if err != nil {
			t.Logf("read error: %v", err)
			return
		}
		var req RESTRequest
		if err := json.Unmarshal(data, &req); err != nil {
			t.Logf("unmarshal error: %v", err)
			return
		}
		requestReceived <- req

		resp := RESTResponse{
			Type:        "RESTResponse",
			RequestID:   req.RequestID,
			StatusCode:  200,
			MessageBody: `{"channels":[]}`,
		}
		respData, _ := json.Marshal(resp)
		conn.Write(ctx, websocket.MessageText, respData)
	})
	defer srv.Close()

	wsUrl := "ws" + strings.TrimPrefix(srv.URL, "http")
	stopChan := make(chan struct{})
	defer close(stopChan)

	ari := &ARInGO{
		httpClient:     new(http.Client),
		wsURL:          wsUrl,
		wsOrigin:       srv.URL,
		username:       "user",
		password:       "pass",
		userAgent:      "test-agent",
		reconnects:     -1,
		delayFunc:      fibDuration,
		evChannel:      make(chan map[string]any, 10),
		errChannel:     make(chan error, 1),
		wsListenerExit: stopChan,
		pending:        make(map[string]chan RESTResponse),
	}

	err := ari.connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ari.disconnect()

	resp, err := ari.Call(HTTP_GET, "channels", map[string]string{"app": "myapp"}, nil)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	select {
	case req := <-requestReceived:
		if req.Type != "RESTRequest" {
			t.Errorf("Expected type RESTRequest, got %s", req.Type)
		}
		if req.Method != HTTP_GET {
			t.Errorf("Expected method GET, got %s", req.Method)
		}
		if req.URI != "channels" {
			t.Errorf("Expected URI channels, got %s", req.URI)
		}
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for request")
	}
}

func TestARInGOCallWithQueryStrings(t *testing.T) {
	var receivedReq RESTRequest

	srv := createTestWSServer(t, func(ctx context.Context, conn *websocket.Conn) {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		json.Unmarshal(data, &receivedReq)

		resp := RESTResponse{
			Type:       "RESTResponse",
			RequestID:  receivedReq.RequestID,
			StatusCode: 200,
		}
		respData, _ := json.Marshal(resp)
		conn.Write(ctx, websocket.MessageText, respData)
	})
	defer srv.Close()

	wsUrl := "ws" + strings.TrimPrefix(srv.URL, "http")
	stopChan := make(chan struct{})
	defer close(stopChan)

	ari := &ARInGO{
		httpClient:     new(http.Client),
		wsURL:          wsUrl,
		reconnects:     -1,
		delayFunc:      fibDuration,
		evChannel:      make(chan map[string]any, 10),
		errChannel:     make(chan error, 1),
		wsListenerExit: stopChan,
		pending:        make(map[string]chan RESTResponse),
	}

	err := ari.connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ari.disconnect()

	_, err = ari.Call(HTTP_POST, "channels",
		map[string]string{"app": "myapp", "channelId": "chan123", "endpoint": "PJSIP/1000", "timeout": "30"},
		nil)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	expectedParams := map[string]bool{
		"app":       false,
		"channelId": false,
		"endpoint":  false,
		"timeout":   false,
	}
	for _, qs := range receivedReq.QueryStrings {
		if _, ok := expectedParams[qs.Name]; ok {
			expectedParams[qs.Name] = true
		}
	}
	for param, found := range expectedParams {
		if !found {
			t.Errorf("Expected param %s not found in request", param)
		}
	}
}

func TestARInGOEventReceiving(t *testing.T) {
	srv := createTestWSServer(t, func(ctx context.Context, conn *websocket.Conn) {
		event := map[string]interface{}{
			"type":        "StasisStart",
			"application": "myapp",
			"channel": map[string]interface{}{
				"id":    "chan123",
				"state": "Ring",
			},
		}
		eventData, _ := json.Marshal(event)
		conn.Write(ctx, websocket.MessageText, eventData)
		time.Sleep(100 * time.Millisecond)
	})
	defer srv.Close()

	wsUrl := "ws" + strings.TrimPrefix(srv.URL, "http")
	stopChan := make(chan struct{})
	evChannel := make(chan map[string]any, 10)

	ari := &ARInGO{
		httpClient:     new(http.Client),
		wsURL:          wsUrl,
		reconnects:     0,
		delayFunc:      fibDuration,
		evChannel:      evChannel,
		errChannel:     make(chan error, 1),
		wsListenerExit: stopChan,
		pending:        make(map[string]chan RESTResponse),
	}

	err := ari.connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer func() {
		close(stopChan)
		ari.disconnect()
	}()

	select {
	case ev := <-evChannel:
		if ev["type"] != "StasisStart" {
			t.Errorf("Expected type StasisStart, got %v", ev["type"])
		}
		if ev["application"] != "myapp" {
			t.Errorf("Expected application myapp, got %v", ev["application"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for event")
	}
}

func TestRESTRequestStruct(t *testing.T) {
	req := RESTRequest{
		Type:          "RESTRequest",
		TransactionID: "txn123",
		RequestID:     "req456",
		Method:        "POST",
		URI:           "channels",
		ContentType:   "application/json",
		QueryStrings: []QueryString{
			{Name: "app", Value: "myapp"},
			{Name: "endpoint", Value: "PJSIP/1000"},
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded RESTRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Type != req.Type {
		t.Errorf("Type mismatch: expected %s, got %s", req.Type, decoded.Type)
	}
	if decoded.TransactionID != req.TransactionID {
		t.Errorf("TransactionID mismatch: expected %s, got %s", req.TransactionID, decoded.TransactionID)
	}
	if decoded.RequestID != req.RequestID {
		t.Errorf("RequestID mismatch: expected %s, got %s", req.RequestID, decoded.RequestID)
	}
	if len(decoded.QueryStrings) != 2 {
		t.Errorf("QueryStrings count mismatch: expected 2, got %d", len(decoded.QueryStrings))
	}
}

func TestRESTResponseStruct(t *testing.T) {
	resp := RESTResponse{
		Type:         "RESTResponse",
		RequestID:    "req456",
		StatusCode:   200,
		ReasonPhrase: "OK",
		ContentType:  "application/json",
		MessageBody:  `{"channels":[]}`,
		Timestamp:    "2024-01-01T12:00:00Z",
		AsteriskID:   "asterisk-id",
		Application:  "myapp",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded RESTResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.StatusCode != 200 {
		t.Errorf("StatusCode mismatch: expected 200, got %d", decoded.StatusCode)
	}
	if decoded.MessageBody != resp.MessageBody {
		t.Errorf("MessageBody mismatch: expected %s, got %s", resp.MessageBody, decoded.MessageBody)
	}
}

func TestQueryStringStruct(t *testing.T) {
	qs := QueryString{
		Name:  "endpoint",
		Value: "PJSIP/1000",
	}

	data, err := json.Marshal(qs)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	expected := `{"name":"endpoint","value":"PJSIP/1000"}`
	if string(data) != expected {
		t.Errorf("JSON mismatch: expected %s, got %s", expected, string(data))
	}
}

func TestARInGODisconnect(t *testing.T) {
	connClosed := make(chan struct{})
	srv := createTestWSServer(t, func(ctx context.Context, conn *websocket.Conn) {
		select {
		case <-ctx.Done():
		case <-connClosed:
		}
	})
	defer srv.Close()

	wsUrl := "ws" + strings.TrimPrefix(srv.URL, "http")
	stopChan := make(chan struct{})

	ari := &ARInGO{
		httpClient:     new(http.Client),
		wsURL:          wsUrl,
		reconnects:     0,
		delayFunc:      fibDuration,
		evChannel:      make(chan map[string]any, 10),
		errChannel:     make(chan error, 1),
		wsListenerExit: stopChan,
		pending:        make(map[string]chan RESTResponse),
	}

	err := ari.connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	close(connClosed)
	close(stopChan)

	time.Sleep(50 * time.Millisecond)
}

func TestARInGODisconnectNilConnection(t *testing.T) {
	ari := &ARInGO{}

	err := ari.disconnect()
	if err != nil {
		t.Errorf("Disconnect with nil ws returned error: %v", err)
	}
}

func TestARInGOPendingMapInitialization(t *testing.T) {
	srv := createTestWSServer(t, func(ctx context.Context, conn *websocket.Conn) {
		<-ctx.Done()
	})
	defer srv.Close()

	wsUrl := "ws" + strings.TrimPrefix(srv.URL, "http")
	stopChan := make(chan struct{})
	defer close(stopChan)

	ari := &ARInGO{
		httpClient:     new(http.Client),
		wsURL:          wsUrl,
		reconnects:     0,
		delayFunc:      fibDuration,
		evChannel:      make(chan map[string]any, 10),
		errChannel:     make(chan error, 1),
		wsListenerExit: stopChan,
	}

	err := ari.connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ari.disconnect()

	if ari.pending == nil {
		t.Error("pending map should be initialized after connect")
	}
}

func fibDuration(durationUnit, maxDuration time.Duration) func() time.Duration {
	a, b := 0, 1
	return func() time.Duration {
		a, b = b, a+b
		fibNrAsDuration := time.Duration(a) * durationUnit
		if maxDuration > 0 && maxDuration < fibNrAsDuration {
			return maxDuration
		}
		return fibNrAsDuration
	}
}
