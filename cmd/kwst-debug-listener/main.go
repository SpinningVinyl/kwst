package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/godbus/dbus/v5"
	"golang.org/x/sys/unix"
	"kwst/internal/buildinfo"
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

func (l *debugListener) Complete(status int32, stdout, stderr string) *dbus.Error {
	l.printf("Complete() was called, reported status: %d\nstdout:\n%s\nstderr:\n%s\n", status, stdout, stderr)
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

func waitForQuitOrSignal(input io.Reader, signals <-chan os.Signal) (os.Signal, error) {
	inputDone := make(chan error, 1)
	go func() {
		inputDone <- waitForQuit(input)
	}()

	select {
	case err := <-inputDone:
		return nil, err
	case receivedSignal := <-signals:
		return receivedSignal, nil
	}
}

func signalExitCode(receivedSignal os.Signal) int {
	if unixSignal, ok := receivedSignal.(syscall.Signal); ok {
		return 128 + int(unixSignal)
	}
	return 1
}

func makeInputImmediate(input *os.File) (func(), error) {
	fd := int(input.Fd())
	state, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if errors.Is(err, unix.ENOTTY) {
		return func() {}, nil
	}
	if err != nil {
		return nil, err
	}

	immediate := *state
	immediate.Lflag &^= unix.ICANON | unix.ECHO
	immediate.Cc[unix.VMIN] = 1
	immediate.Cc[unix.VTIME] = 0
	if err := unix.IoctlSetTermios(fd, unix.TCSETS, &immediate); err != nil {
		return nil, err
	}

	return func() {
		_ = unix.IoctlSetTermios(fd, unix.TCSETS, state)
	}, nil
}

func run(arguments []string, input *os.File, stdout, stderr io.Writer) int {
	if len(arguments) > 0 && arguments[0] == "--version" {
		fmt.Fprintln(stdout, buildinfo.Version)
		fmt.Fprintln(stdout, buildinfo.BuildTime)
		return 0
	}

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
	fmt.Fprintf(stdout, "*** KWST debug listener ***\n")
	fmt.Fprintf(stdout, "Listening at address %s, press q to quit.\n", names[0])

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	restoreInput, err := makeInputImmediate(input)
	if err != nil {
		fmt.Fprintln(stderr, "Failed to configure terminal input:", err)
		return 1
	}
	defer restoreInput()

	receivedSignal, err := waitForQuitOrSignal(input, signals)
	if err != nil && !errors.Is(err, io.EOF) {
		fmt.Fprintln(stderr, "Failed to read keyboard input:", err)
		return 1
	}
	if receivedSignal != nil {
		return signalExitCode(receivedSignal)
	}

	return 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
