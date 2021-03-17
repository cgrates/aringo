/*
Released under MIT License <http://www.opensource.org/licenses/mit-license.php
Copyright (C) ITsysCOM GmbH. All Rights Reserved.

Provides Asterisk ARI connector from Go programming language.
*/

package aringo

import (
	"fmt"
	"testing"
	"time"
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
	received, err := NewARInGO(wsUrl, wsOrigin, username, password, userAgent, evChannel, errChannel, stopChan, connectAttempts, reconnects)

	if err != experr {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", experr, err)
	} else if received != nil {
		t.Errorf("\nExpected: <%+v>, \nReceived: <%+v>", nil, received)
	}
}
