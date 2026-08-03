package main

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/godbus/dbus/v5/introspect"
)

func TestDebugListenerPrintsCalls(t *testing.T) {
	var output bytes.Buffer
	listener := newDebugListener(&output)

	if err := listener.Msg("result", "window-id"); err != nil {
		t.Fatalf("Msg returned a D-Bus error: %v", err)
	}
	if err := listener.Close(); err != nil {
		t.Fatalf("Close returned a D-Bus error: %v", err)
	}
	if err := listener.CloseWithStatus(23); err != nil {
		t.Fatalf("CloseWithStatus returned a D-Bus error: %v", err)
	}

	want := regexp.MustCompile(
		`^\[\d{2}:\d{2}:\d{2}\] Msg\(\) was called, type: result, message: window-id\n` +
			`\[\d{2}:\d{2}:\d{2}\] Close\(\) was called\n` +
			`\[\d{2}:\d{2}:\d{2}\] CloseWithStatus\(\) was called, reported status: 23\n$`,
	)
	if got := output.String(); !want.MatchString(got) {
		t.Fatalf("output = %q, want timestamped listener calls", got)
	}
}

func TestDebugListenerDBusMethods(t *testing.T) {
	methods := introspect.Methods(newDebugListener(&bytes.Buffer{}))
	want := map[string][]string{
		"Close":           {},
		"CloseWithStatus": {"i"},
		"Msg":             {"s", "s"},
	}

	if len(methods) != len(want) {
		t.Fatalf("exported method count = %d, want %d: %#v", len(methods), len(want), methods)
	}

	for _, method := range methods {
		wantArgs, ok := want[method.Name]
		if !ok {
			t.Errorf("unexpected exported method %q", method.Name)
			continue
		}
		if len(method.Args) != len(wantArgs) {
			t.Errorf("%s argument count = %d, want %d", method.Name, len(method.Args), len(wantArgs))
			continue
		}
		for i, arg := range method.Args {
			if arg.Type != wantArgs[i] || arg.Direction != "in" {
				t.Errorf("%s argument %d = %#v, want input type %q", method.Name, i, arg, wantArgs[i])
			}
		}
	}
}

func TestWaitForQuit(t *testing.T) {
	if err := waitForQuit(strings.NewReader("abq")); err != nil {
		t.Fatalf("waitForQuit returned an error: %v", err)
	}
}
