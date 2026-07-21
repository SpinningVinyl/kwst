package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/alecthomas/kong"
	"github.com/godbus/dbus/v5"
)

var debug = false

var Version = "v0.0.0"
var BuildTime = "Thu 01 Jan 1970 00:00:00 UTC"

// define the CLI structure for Kong to parse. See also commands.go
type Globals struct {
	Debug bool `help:"Enable debug mode." short:"d"`
}

type CLI struct {
	Globals

	List               ListCmd               `cmd:"" help:"List all windows. The data is returned as tab-separated rows containing the window's UUID, resourceClass and resourceName. Each window is represented by a separate row."`
	Find               FindCmd               `cmd:"" help:"Search for windows using a case-insensitive regular expression."`
	GetActiveWindow    GetActiveWindowCmd    `cmd:"" help:"Get the UUID of the active window."`
	GetWindowGeometry  GetWindowGeometryCmd  `cmd:"" help:"Get the geometry (size and position) of the window with the specified UUID. The data is returned in the format required for the set-window-geometry command (x y width height)."`
	GetWorkspace       GetWorkspaceCmd       `cmd:"" help:"Get the ID of the active workspace."`
	SetWorkspace       SetWorkspaceCmd       `cmd:"" help:"Switch to the workspace with the specified ID."`
	ActivateWindow     ActivateWindowCmd     `cmd:"" help:"Activate the window with the provided UUID, if such a window exists."`
	SetWindowSize      SetWindowSizeCmd      `cmd:"" help:"Set the size of the window with the provided UUID."`
	SetWindowPosition  SetWindowPosCmd       `cmd:"" help:"Set the position of the window with the provided UUID."`
	SetWindowGeometry  SetWindowGeometryCmd  `cmd:"" help:"Change geometry of the window with the provided UUID."`
	SetWindowWorkspace SetWindowWorkspaceCmd `cmd:"" help:"Send the window with the specified UUID to the workspace with the specified number."`
	SetWindowProperty  SetWindowPropertyCmd  `cmd:"" help:"Change the value of a property on a window with the specified UUID."`
	CloseWindow        CloseWindowCmd        `cmd:"" help:"Close the window with the provided UUID."`
	RunCustomScript    RunCustomScriptCmd    `cmd:"" help:"Run a custom script. Supports up to six optional parameters."`
	GetMousePosition   MousePosCmd           `cmd:"" help:"Return the absolute position of the mouse cursor."`
}

// parameters that are passed to the script template
type ScriptParams struct {
	DbusAddr              string
	Debug                 bool
	ScriptName            string
	SearchTerm            string
	SearchField           string
	IncludeSpecialWindows bool
	ShowCaptions          bool
	ShowPids              bool
	Uuid                  string
	WorkspaceId           int
	X                     int
	Y                     int
	Width                 int
	Height                int
	WindowProperty        string
	PropertyValue         string
	P1                    string
	P2                    string
	P3                    string
	P4                    string
	P5                    string
	P6                    string
}

type ScriptPackage struct {
	ScriptTemplate string
	Params         ScriptParams
}

// jsString returns value as a quoted JavaScript string literal. JSON string
// literals are also valid JavaScript string literals and safely escape values
// that would otherwise alter the generated script.
func jsString(value string) (string, error) {
	quoted, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(quoted), nil
}

func prepareScript(w io.Writer, sp ScriptPackage) error {
	sp.ScriptTemplate += JS_FOOTER
	tmpl, err := template.New("kwin_script").Funcs(template.FuncMap{
		"jsString": jsString,
	}).Parse(sp.ScriptTemplate)
	if err != nil {
		return fmt.Errorf("Error parsing script template: %w", err)
	}
	if err := tmpl.Execute(w, sp.Params); err != nil {
		return fmt.Errorf("Error executing script template: %w", err)
	}
	return nil
}

// define the DBus object for exporting
type Server struct {
	done   chan int
	failed atomic.Bool
	once   sync.Once
	stdout io.Writer
	stderr io.Writer
}

func newServer(stdout, stderr io.Writer) *Server {
	return &Server{
		done:   make(chan int, 1),
		stdout: stdout,
		stderr: stderr,
	}
}

func (s *Server) Msg(msgType, message string) *dbus.Error {
	if msgType == "result" {
		fmt.Fprintln(s.stdout, message)
	} else if msgType == "error" {
		s.failed.Store(true)
		fmt.Fprintln(s.stderr, "KWin script returned an error:", message)
	}
	return nil
}

// Close is retained for compatibility with existing custom scripts. New
// scripts should use CloseWithStatus so completion does not depend on the
// ordering of separate Msg and Close calls.
func (s *Server) Close() *dbus.Error {
	exitCode := 0
	if s.failed.Load() {
		exitCode = 1
	}
	s.finish(exitCode)
	return nil
}

func (s *Server) CloseWithStatus(exitCode int32) *dbus.Error {
	status := normalizeExitCode(int(exitCode))
	if s.failed.Load() {
		status = 1
	}
	s.finish(status)
	return nil
}

func (s *Server) finish(exitCode int) {
	s.once.Do(func() {
		s.done <- exitCode
	})
}

func normalizeExitCode(exitCode int) int {
	if exitCode < 0 || exitCode > 255 {
		return 1
	}
	return exitCode
}

func scriptTimeout(debug bool) time.Duration {
	if debug {
		return 5 * time.Minute
	}
	return 5 * time.Second
}

func waitForCompletion(done <-chan int, timeout time.Duration, stderr io.Writer) int {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case exitCode := <-done:
		return normalizeExitCode(exitCode)
	case <-timer.C:
		fmt.Fprintln(stderr, "Close() call not received from KWin scripting. Timing out...")
		return 124
	}
}

func debugPrint(a ...any) {
	if debug {
		fmt.Print("DEBUG: ")
		for _, element := range a {
			fmt.Print(element)
			fmt.Print(" ")
		}
		fmt.Println()
	}
}

func main() {
	os.Exit(run())
}

func run() int {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println(Version)
		fmt.Println(BuildTime)
		return 0
	}

	cli := CLI{}

	// parse command line parameters
	ctx := kong.Parse(&cli,
		kong.Name("kwst"),
		kong.Description("KWin scripting tool"),
		kong.UsageOnError())

	debug = cli.Globals.Debug

	// create a temporary file
	scriptFile, err := os.CreateTemp(os.TempDir(), "kwst-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating temporary file:", err)
		return 1
	}
	defer scriptFile.Close()
	debugPrint("Temp script file name:", scriptFile.Name())
	if !debug { // do not delete the script file in the debug mode
		defer os.Remove(scriptFile.Name())
	}

	// set up the DBus connection
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to connect to session bus:", err)
		return 1
	}
	defer conn.Close()
	debugPrint("DBus address:", conn.Names()[0])

	s := newServer(os.Stdout, os.Stderr)
	if err := conn.Export(s, "/net/prsv/kwst", "net.prsv.kwst"); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to export D-Bus server:", err)
		return 1
	}

	// create and populate the ScriptParams struct
	var sp ScriptPackage
	sp.Params.ScriptName = filepath.Base(scriptFile.Name())
	sp.Params.DbusAddr = conn.Names()[0]
	sp.Params.Debug = debug

	// process the template depending on the command line arguments
	sp.ScriptTemplate = JS_HEADER
	err = ctx.Run(&sp)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if err := prepareScript(scriptFile, sp); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := scriptFile.Close(); err != nil {
		fmt.Fprintln(os.Stderr, "Error closing temporary script file:", err)
		return 1
	}

	// get the KWin object and load the script
	var scriptId int
	kwinConn := conn.Object("org.kde.KWin", "/Scripting")
	err = kwinConn.Call("loadScript", 0, scriptFile.Name(), sp.Params.ScriptName).Store(&scriptId)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to load KWin script:", err)
		return 1
	}
	debugPrint("Registered script ID:", strconv.Itoa(scriptId))

	// get the script object and run the script
	scriptConn := conn.Object("org.kde.KWin", dbus.ObjectPath(fmt.Sprintf("/Scripting/Script%d", scriptId)))
	if call := scriptConn.Call("org.kde.kwin.Script.run", 0); call.Err != nil {
		fmt.Fprintln(os.Stderr, "Failed to run KWin script:", call.Err)
		if unloadCall := kwinConn.Call("unloadScript", 0, sp.Params.ScriptName); unloadCall.Err != nil {
			fmt.Fprintln(os.Stderr, "Failed to unload KWin script:", unloadCall.Err)
		}
		return 1
	}

	exitCode := waitForCompletion(s.done, scriptTimeout(debug), os.Stderr)

	// give it some time to finish receiving and processing DBus messages
	time.Sleep(5 * time.Millisecond)
	if call := kwinConn.Call("unloadScript", 0, sp.Params.ScriptName); call.Err != nil {
		fmt.Fprintln(os.Stderr, "Failed to unload KWin script:", call.Err)
		return 1
	}
	return exitCode
}
