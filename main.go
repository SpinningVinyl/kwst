package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"text/template"
	"time"

	"github.com/alecthomas/kong"
	"github.com/godbus/dbus/v5"
)

var quit = make(chan int)

var debug = false

var Version = "v00000"
var BuildTime = "Thu 01 Jan 1970 00:00:00 UTC"

// define the CLI structure for Kong to parse. See also commands.go
type Globals struct {
	Debug bool `help:"Enable debug mode." short:"d"`
}

type CLI struct {
	Globals

	List               ListCmd               `cmd:"" help:"List all windows. The data is returned as tab-separated rows containing the window's UUID, resourceClass, resourceName and PID of the process the window belongs to (the PID is not guaranteed to be correct for X11 windows). Each window is represented by a separate row."`
	Find               FindCmd               `cmd:"" help:"Search for windows using the specified search term."`
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
}

// parameters that are passed to the script template
type ScriptParams struct {
	DbusAddr              string
	Debug                 bool
	ScriptName            string
	SearchTerm            string
	SearchField           string
	IncludeSpecialWindows bool
	Uuid                  string
	WorkspaceId           int
	X                     int
	Y                     int
	Width                 int
	Height                int
	WindowProperty        string
	PropertyValue         string
}

// define the DBus object for exporting
type Server struct{}

func (s Server) Msg(msgType, message string) *dbus.Error {
	if msgType == "result" {
		fmt.Fprintln(os.Stdout, message)
	} else if msgType == "error" {
		fmt.Fprintln(os.Stderr, "KWin script returned an error:", message)
	}
	return nil
}

func (s Server) Close() *dbus.Error {
	quit <- 0
	return nil
}

func timerStart() {
	delay := "5s"
	if debug {
		delay = "5m"
	}
	duration, _ := time.ParseDuration(delay)
	time.Sleep(duration)
	fmt.Fprintln(os.Stderr, "Close() call not received from KWin scripting. Timing out...")
	quit <- 0
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

	if os.Args[1] == "--version" {
		fmt.Println(Version)
		fmt.Println(BuildTime)
		os.Exit(0)
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
		os.Exit(1)
	}
	debugPrint("Temp script file name:", scriptFile.Name())
	if !debug { // do not delete the script file in the debug mode
		defer os.Remove(scriptFile.Name())
	}

	// set up the DBus connection
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to connect to session bus:", err)
		os.Exit(1)
	}
	defer conn.Close()
	debugPrint("DBus address:", conn.Names()[0])

	s := Server{}
	conn.Export(s, "/net/prsv/kwst", "net.prsv.kwst")

	// create and populate the ScriptParams struct
	var params ScriptParams
	params.ScriptName = filepath.Base(scriptFile.Name())
	params.DbusAddr = conn.Names()[0]
	params.Debug = debug

	// process the template depending on the command line arguments
	scriptTemplate := JS_HEADER
	debugPrint("cmd:", ctx.Command())
	if ctx.Command() == "list" {
		params.IncludeSpecialWindows = cli.List.IncludeSpecialWindows
		scriptTemplate += JS_LIST
	}
	if ctx.Command() == "find <search-term>" {
		params.SearchTerm = cli.Find.SearchTerm
		params.SearchField = cli.Find.SearchField
		scriptTemplate += JS_FIND
	}
	if ctx.Command() == "get-active-window" {
		scriptTemplate += JS_GET_ACTIVE_WINDOW
	}
	if ctx.Command() == "get-window-geometry <uuid>" {
		params.Uuid = cli.GetWindowGeometry.Uuid
		scriptTemplate += JS_GET_WINDOW_GEOMETRY
	}
	if ctx.Command() == "get-workspace" {
		scriptTemplate += JS_GET_WORKSPACE
	}
	if ctx.Command() == "set-workspace <workspace-id>" {
		params.WorkspaceId = cli.SetWorkspace.WorkspaceId
		scriptTemplate += JS_SET_WORKSPACE
	}
	if ctx.Command() == "activate-window <uuid>" {
		params.Uuid = cli.ActivateWindow.Uuid
		scriptTemplate += JS_ACTIVATE_WINDOW
	}
	if ctx.Command() == "set-window-geometry <uuid> <x> <y> <width> <height>" {
		params.X = cli.SetWindowGeometry.X
		params.Y = cli.SetWindowGeometry.Y
		params.Width = cli.SetWindowGeometry.Width
		params.Height = cli.SetWindowGeometry.Height
		params.Uuid = cli.SetWindowGeometry.Uuid
		scriptTemplate += JS_SET_WINDOW_GEOMETRY
	}
	if ctx.Command() == "set-window-size <uuid> <width> <height>" {
		params.Width = cli.SetWindowSize.Width
		params.Height = cli.SetWindowSize.Height
		params.Uuid = cli.SetWindowSize.Uuid
		scriptTemplate += JS_SET_WINDOW_SIZE
	}
	if ctx.Command() == "set-window-position <uuid> <x> <y>" {
		params.X = cli.SetWindowPosition.X
		params.Y = cli.SetWindowPosition.Y
		params.Uuid = cli.SetWindowPosition.Uuid
		scriptTemplate += JS_SET_WINDOW_POSITION
	}
	if ctx.Command() == "set-window-workspace <uuid> <workspace-id>" {
		params.Uuid = cli.SetWindowWorkspace.Uuid
		params.WorkspaceId = cli.SetWindowWorkspace.WorkspaceId
		scriptTemplate += JS_SET_WINDOW_WORKSPACE
	}
	if ctx.Command() == "set-window-property <uuid>" {
		params.WindowProperty = cli.SetWindowProperty.Property
		params.PropertyValue = cli.SetWindowProperty.Value
		params.Uuid = cli.SetWindowProperty.Uuid
		scriptTemplate += JS_SET_WINDOW_PROPERTY
	}

	scriptTemplate += JS_FOOTER
	tmpl, err := template.New("kwin_script").Parse(scriptTemplate)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error parsing script template:", err)
		os.Exit(1)
	}
	tmpl.Execute(scriptFile, params)

	// get the KWin object and load the script
	var scriptId int
	kwinConn := conn.Object("org.kde.KWin", "/Scripting")
	err = kwinConn.Call("loadScript", 0, scriptFile.Name(), params.ScriptName).Store(&scriptId)
	debugPrint("Registered script ID:", strconv.Itoa(scriptId))

	// get the script object and run the script
	scriptConn := conn.Object("org.kde.KWin", dbus.ObjectPath(fmt.Sprintf("/Scripting/Script%d", scriptId)))
	scriptConn.Call("org.kde.kwin.Script.run", 0)

	// make sure that the program eventually exits even if we do not receive the Close() call
	go timerStart()

	select {
	case <-quit:
		// give it some time to finish receiving and processing DBus messages
		time.Sleep(5 * time.Millisecond)
		// unload the script from KWin
		kwinConn.Call("unloadScript", 0, params.ScriptName)
		return
	}
}
