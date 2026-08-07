package main

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"strings"
	"syscall"
	"testing"

	"github.com/godbus/dbus/v5/introspect"
	"kwst/internal/buildinfo"
)

func TestVersion(t *testing.T) {
	originalVersion := buildinfo.Version
	originalBuildTime := buildinfo.BuildTime
	buildinfo.Version = "v-test"
	buildinfo.BuildTime = "test build time"
	t.Cleanup(func() {
		buildinfo.Version = originalVersion
		buildinfo.BuildTime = originalBuildTime
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if exitCode := run([]string{"--version"}, nil, &stdout, &stderr); exitCode != 0 {
		t.Fatalf("run returned exit code %d, want 0", exitCode)
	}
	if got, want := stdout.String(), "v-test\ntest build time\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want no output", stderr.String())
	}
}

func TestDebugListenerPrintsCalls(t *testing.T) {
	var output bytes.Buffer
	listener := newDebugListener(&output)

	if err := listener.Complete(23, "window-id", "script failed"); err != nil {
		t.Fatalf("Complete returned a D-Bus error: %v", err)
	}

	want := regexp.MustCompile(
		`^\[\d{2}:\d{2}:\d{2}\] Complete\(\) was called, reported status: 23\n` +
			`stdout:\nwindow-id\nstderr:\nscript failed\n$`,
	)
	if got := output.String(); !want.MatchString(got) {
		t.Fatalf("output = %q, want timestamped listener calls", got)
	}
}

func TestDebugListenerDBusMethods(t *testing.T) {
	methods := introspect.Methods(newDebugListener(&bytes.Buffer{}))
	want := map[string][]string{
		"Complete": {"i", "s", "s"},
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

func TestWaitForQuitOrSignal(t *testing.T) {
	t.Run("keyboard input", func(t *testing.T) {
		receivedSignal, err := waitForQuitOrSignal(strings.NewReader("abq"), make(chan os.Signal))
		if err != nil {
			t.Fatalf("waitForQuitOrSignal returned an error: %v", err)
		}
		if receivedSignal != nil {
			t.Fatalf("waitForQuitOrSignal returned signal %v, want nil", receivedSignal)
		}
	})

	t.Run("signal", func(t *testing.T) {
		reader, writer := io.Pipe()
		t.Cleanup(func() {
			_ = reader.Close()
			_ = writer.Close()
		})
		signals := make(chan os.Signal, 1)
		signals <- os.Interrupt

		receivedSignal, err := waitForQuitOrSignal(reader, signals)
		if err != nil {
			t.Fatalf("waitForQuitOrSignal returned an error: %v", err)
		}
		if receivedSignal != os.Interrupt {
			t.Fatalf("waitForQuitOrSignal returned signal %v, want %v", receivedSignal, os.Interrupt)
		}
	})
}

func TestSignalExitCode(t *testing.T) {
	for _, test := range []struct {
		signal os.Signal
		want   int
	}{
		{signal: os.Interrupt, want: 130},
		{signal: syscall.SIGTERM, want: 143},
	} {
		if got := signalExitCode(test.signal); got != test.want {
			t.Errorf("signalExitCode(%v) = %d, want %d", test.signal, got, test.want)
		}
	}
}
