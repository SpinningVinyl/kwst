package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
	"golang.org/x/term"
)

const (
	dbusObjectPath = "/net/prsv/kwst"
	dbusInterface  = "net.prsv.kwst"
)

type debugListener struct {
	stdout io.Writer
	mu     sync.Mutex
}

func newDebugListener(stdout io.Writer) *debugListener {
	return &debugListener{stdout: stdout}
}

func (l *debugListener) Msg(msgType, message string) *dbus.Error {
	l.printf("Msg() was called, type: %s, message: %s\n", msgType, message)
	return nil
}

func (l *debugListener) Close() *dbus.Error {
	l.printf("Close() was called\n")
	return nil
}

func (l *debugListener) CloseWithStatus(status int32) *dbus.Error {
	l.printf("CloseWithStatus() was called, reported status: %d\n", status)
	return nil
}

func (l *debugListener) printf(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(l.stdout, "[%s] ", time.Now().Format(time.TimeOnly))
	fmt.Fprintf(l.stdout, format, args...)
}

func waitForQuit(input io.Reader) error {
	var key [1]byte
	for {
		_, err := input.Read(key[:])
		if err != nil {
			return err
		}
		if key[0] == 'q' {
			return nil
		}
	}
}

func makeInputImmediate(input *os.File) (func(), error) {
	fd := int(input.Fd())
	if !term.IsTerminal(fd) {
		return func() {}, nil
	}

	state, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}

	return func() {
		_ = term.Restore(fd, state)
	}, nil
}

func run(input *os.File, stdout, stderr io.Writer) int {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		fmt.Fprintln(stderr, "Failed to connect to session bus:", err)
		return 1
	}
	defer conn.Close()

	listener := newDebugListener(stdout)
	if err := conn.Export(listener, dbus.ObjectPath(dbusObjectPath), dbusInterface); err != nil {
		fmt.Fprintln(stderr, "Failed to export D-Bus server:", err)
		return 1
	}

	names := conn.Names()
	if len(names) == 0 {
		fmt.Fprintln(stderr, "D-Bus connection has no unique address")
		return 1
	}
	fmt.Fprintf(stdout, "*** KWST debug listener ***\n")
	fmt.Fprintf(stdout, "Listening at address %s, press q to quit.\n", names[0])

	restoreInput, err := makeInputImmediate(input)
	if err != nil {
		fmt.Fprintln(stderr, "Failed to configure terminal input:", err)
		return 1
	}
	defer restoreInput()

	if err := waitForQuit(input); err != nil && !errors.Is(err, io.EOF) {
		fmt.Fprintln(stderr, "Failed to read keyboard input:", err)
		return 1
	}

	return 0
}

func main() {
	os.Exit(run(os.Stdin, os.Stdout, os.Stderr))
}
