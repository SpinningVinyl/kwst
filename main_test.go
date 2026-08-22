package main

import (
	"bytes"
	"errors"
	"io"
	"math"
	"os"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/godbus/dbus/v5/introspect"
)

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) {
	return 0, io.ErrClosedPipe
}

type scriptCommand interface {
	Run(*ScriptPackage) error
}

func TestCommandRunMethods(t *testing.T) {
	customScript := "custom script body"
	customScriptFile, err := os.CreateTemp(t.TempDir(), "custom-script-*.js")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		customScriptFile.Close()
	})
	if _, err := customScriptFile.WriteString(customScript); err != nil {
		t.Fatal(err)
	}

	const initialTemplate = "initial template\n"
	tests := []struct {
		name         string
		command      scriptCommand
		wantTemplate string
		wantParams   ScriptParams
		wantCustom   bool
	}{
		{
			name:         "list",
			command:      &ListCmd{IncludeSpecialWindows: true, ShowCaptions: true, ShowPids: true},
			wantTemplate: initialTemplate + JS_WINDOW_LIST_HELPERS + JS_LIST,
			wantParams: ScriptParams{
				IncludeSpecialWindows: true,
				ShowCaptions:          true,
				ShowPids:              true,
			},
		},
		{
			name:         "find",
			command:      &FindCmd{SearchField: "caption", SearchTerm: "terminal"},
			wantTemplate: initialTemplate + JS_FIND,
			wantParams:   ScriptParams{SearchField: "caption", SearchTerm: "terminal"},
		},
		{
			name:         "get active window",
			command:      &GetActiveWindowCmd{},
			wantTemplate: initialTemplate + JS_GET_ACTIVE_WINDOW,
		},
		{
			name:         "get window geometry",
			command:      &GetWindowGeometryCmd{Uuid: "window-id"},
			wantTemplate: initialTemplate + JS_GET_WINDOW_GEOMETRY,
			wantParams:   ScriptParams{Uuid: "window-id"},
		},
		{
			name:         "get workspace",
			command:      &GetWorkspaceCmd{},
			wantTemplate: initialTemplate + JS_GET_WORKSPACE,
		},
		{
			name:         "set workspace",
			command:      &SetWorkspaceCmd{WorkspaceId: 3},
			wantTemplate: initialTemplate + JS_SET_WORKSPACE,
			wantParams:   ScriptParams{WorkspaceId: 3},
		},
		{
			name:         "activate window",
			command:      &ActivateWindowCmd{Uuid: "window-id"},
			wantTemplate: initialTemplate + JS_ACTIVATE_WINDOW,
			wantParams:   ScriptParams{Uuid: "window-id"},
		},
		{
			name:         "set window size",
			command:      &SetWindowSizeCmd{Uuid: "window-id", Width: 640, Height: 480},
			wantTemplate: initialTemplate + JS_TILE_UNASSIGNMENT_HELPER + JS_SET_WINDOW_SIZE,
			wantParams:   ScriptParams{Uuid: "window-id", Width: 640, Height: 480},
		},
		{
			name:         "set window position",
			command:      &SetWindowPosCmd{Uuid: "window-id", X: 10, Y: 20},
			wantTemplate: initialTemplate + JS_TILE_UNASSIGNMENT_HELPER + JS_SET_WINDOW_POSITION,
			wantParams:   ScriptParams{Uuid: "window-id", X: 10, Y: 20},
		},
		{
			name:         "set window geometry",
			command:      &SetWindowGeometryCmd{Uuid: "window-id", X: 10, Y: 20, Width: 640, Height: 480},
			wantTemplate: initialTemplate + JS_TILE_UNASSIGNMENT_HELPER + JS_SET_WINDOW_GEOMETRY,
			wantParams:   ScriptParams{Uuid: "window-id", X: 10, Y: 20, Width: 640, Height: 480},
		},
		{
			name: "set window geometry relative",
			command: &SetWindowGeometryRelativeCmd{
				Uuid:           "window-id",
				RelativeX:      10.5,
				RelativeY:      20.25,
				RelativeWidth:  66.7,
				RelativeHeight: 50,
			},
			wantTemplate: initialTemplate + JS_TILE_UNASSIGNMENT_HELPER + JS_SET_WINDOW_GEOMETRY_RELATIVE,
			wantParams: ScriptParams{
				Uuid:           "window-id",
				RelativeX:      10.5,
				RelativeY:      20.25,
				RelativeWidth:  66.7,
				RelativeHeight: 50,
			},
		},
		{
			name: "set window size relative",
			command: &SetWindowSizeRelativeCmd{
				Uuid:           "window-id",
				RelativeWidth:  66.7,
				RelativeHeight: 50,
			},
			wantTemplate: initialTemplate + JS_TILE_UNASSIGNMENT_HELPER + JS_SET_WINDOW_SIZE_RELATIVE,
			wantParams: ScriptParams{
				Uuid:           "window-id",
				RelativeWidth:  66.7,
				RelativeHeight: 50,
			},
		},
		{
			name: "set window position relative",
			command: &SetWindowPositionRelativeCmd{
				Uuid:      "window-id",
				RelativeX: 10.5,
				RelativeY: 20.25,
			},
			wantTemplate: initialTemplate + JS_TILE_UNASSIGNMENT_HELPER + JS_SET_WINDOW_POSITION_RELATIVE,
			wantParams: ScriptParams{
				Uuid:      "window-id",
				RelativeX: 10.5,
				RelativeY: 20.25,
			},
		},
		{
			name:         "set window workspace",
			command:      &SetWindowWorkspaceCmd{Uuid: "window-id", WorkspaceId: 3},
			wantTemplate: initialTemplate + JS_SET_WINDOW_WORKSPACE,
			wantParams:   ScriptParams{Uuid: "window-id", WorkspaceId: 3},
		},
		{
			name:         "set window property",
			command:      &SetWindowPropertyCmd{Uuid: "window-id", Property: "keepAbove", Value: "toggle"},
			wantTemplate: initialTemplate + JS_SET_WINDOW_PROPERTY,
			wantParams: ScriptParams{
				Uuid:           "window-id",
				WindowProperty: "keepAbove",
				PropertyValue:  "toggle",
			},
		},
		{
			name:         "close window",
			command:      &CloseWindowCmd{Uuid: "window-id"},
			wantTemplate: initialTemplate + JS_CLOSE_WINDOW,
			wantParams:   ScriptParams{Uuid: "window-id"},
		},
		{
			name: "run custom script",
			command: &RunCustomScriptCmd{
				Parameter1: "one",
				Parameter2: "two",
				Parameter3: "three",
				Parameter4: "four",
				Parameter5: "five",
				Parameter6: "six",
				ScriptFile: customScriptFile,
			},
			wantTemplate: customScript,
			wantParams: ScriptParams{
				P1: "one",
				P2: "two",
				P3: "three",
				P4: "four",
				P5: "five",
				P6: "six",
			},
			wantCustom: true,
		},
		{
			name:         "get mouse position",
			command:      &MousePosCmd{},
			wantTemplate: initialTemplate + JS_MOUSE_POS,
		},
		{
			name:         "list outputs",
			command:      &ListOutputsCmd{},
			wantTemplate: initialTemplate + JS_LIST_OUTPUTS,
		},
		{
			name:         "list tiles",
			command:      &ListTilesCmd{OutputName: "DP-1", LeavesOnly: true},
			wantTemplate: initialTemplate + JS_TILE_HELPERS + JS_LIST_TILES,
			wantParams:   ScriptParams{OutputName: "DP-1", LeavesOnly: true},
		},
		{
			name:         "get window tile",
			command:      &GetWindowTileCmd{Uuid: "window-id"},
			wantTemplate: initialTemplate + JS_TILE_HELPERS + JS_GET_WINDOW_TILE,
			wantParams:   ScriptParams{Uuid: "window-id"},
		},
		{
			name: "set window tile",
			command: &SetWindowTileCmd{
				OutputName: "DP-1",
				Uuid:       "window-id",
				TilePath:   "1.0",
			},
			wantTemplate: initialTemplate + JS_TILE_HELPERS + JS_SET_WINDOW_TILE,
			wantParams: ScriptParams{
				OutputName: "DP-1",
				Uuid:       "window-id",
				TilePath:   "1.0",
			},
		},
		{
			name:         "unset window tile",
			command:      &UnsetWindowTileCmd{Uuid: "window-id"},
			wantTemplate: initialTemplate + JS_TILE_UNASSIGNMENT_HELPER + JS_UNSET_WINDOW_TILE,
			wantParams:   ScriptParams{Uuid: "window-id"},
		},
		{
			name: "list tile windows",
			command: &ListTileWindowsCmd{
				OutputName:   "DP-1",
				ShowCaptions: true,
				ShowPids:     true,
				TilePath:     "1.0",
			},
			wantTemplate: initialTemplate + JS_TILE_HELPERS + JS_WINDOW_LIST_HELPERS + JS_LIST_TILE_WINDOWS,
			wantParams: ScriptParams{
				OutputName:   "DP-1",
				ShowCaptions: true,
				ShowPids:     true,
				TilePath:     "1.0",
			},
		},
		{
			name: "resize tile",
			command: &ResizeTileCmd{
				OutputName: "DP-1",
				TilePath:   "1.0",
				Delta:      2.5,
				Edge:       "right",
			},
			wantTemplate: initialTemplate + JS_TILE_HELPERS + JS_RESIZE_TILE,
			wantParams: ScriptParams{
				OutputName: "DP-1",
				TilePath:   "1.0",
				Delta:      2.5,
				Edge:       "right",
			},
		},
		{
			name:         "resize active tile",
			command:      &ResizeActiveTileCmd{Delta: -2.5, Edge: "left"},
			wantTemplate: initialTemplate + JS_TILE_HELPERS + JS_RESIZE_ACTIVE_TILE,
			wantParams:   ScriptParams{Delta: -2.5, Edge: "left"},
		},
		{
			name: "set tile geometry",
			command: &SetTileGeometryCmd{
				OutputName: "DP-1",
				TilePath:   "1.0",
				X:          0.1,
				Y:          0.2,
				Width:      0.3,
				Height:     0.4,
			},
			wantTemplate: initialTemplate + JS_TILE_HELPERS + JS_SET_TILE_GEOMETRY,
			wantParams: ScriptParams{
				OutputName:     "DP-1",
				TilePath:       "1.0",
				RelativeX:      0.1,
				RelativeY:      0.2,
				RelativeWidth:  0.3,
				RelativeHeight: 0.4,
			},
		},
		{
			name: "set active tile geometry",
			command: &SetActiveTileGeometryCmd{
				X:      0.1,
				Y:      0.2,
				Width:  0.3,
				Height: 0.4,
			},
			wantTemplate: initialTemplate + JS_TILE_HELPERS + JS_SET_ACTIVE_TILE_GEOMETRY,
			wantParams: ScriptParams{
				RelativeX:      0.1,
				RelativeY:      0.2,
				RelativeWidth:  0.3,
				RelativeHeight: 0.4,
			},
		},
		{
			name:         "get active output",
			command:      &GetActiveOutputCmd{},
			wantTemplate: initialTemplate + JS_GET_ACTIVE_OUTPUT,
		},
		{
			name:         "get cursor output",
			command:      &GetCursorOutputCmd{},
			wantTemplate: initialTemplate + JS_GET_CURSOR_OUTPUT,
		},
		{
			name:         "get active window output",
			command:      &GetActiveWindowOutputCmd{},
			wantTemplate: initialTemplate + JS_GET_ACTIVE_WINDOW_OUTPUT,
		},
		{
			name:         "get output geometry",
			command:      &GetOutputGeometryCmd{ClientArea: true, OutputName: "DP-1"},
			wantTemplate: initialTemplate + JS_GET_OUTPUT_GEOMETRY,
			wantParams:   ScriptParams{ClientArea: true, OutputName: "DP-1"},
		},
		{
			name:         "get window opacity",
			command:      &GetWindowOpacityCmd{Uuid: "window-id"},
			wantTemplate: initialTemplate + JS_GET_WINDOW_OPACITY,
			wantParams:   ScriptParams{Uuid: "window-id"},
		},
		{
			name:         "set window opacity",
			command:      &SetWindowOpacityCmd{Uuid: "window-id", Opacity: 0.5},
			wantTemplate: initialTemplate + JS_SET_WINDOW_OPACITY,
			wantParams:   ScriptParams{Uuid: "window-id", Opacity: 0.5},
		},
		{
			name:         "increase window opacity",
			command:      &IncreaseWindowOpacityCmd{Uuid: "window-id"},
			wantTemplate: initialTemplate + JS_ADJUST_WINDOW_OPACITY,
			wantParams:   ScriptParams{Uuid: "window-id", Delta: 0.05},
		},
		{
			name:         "decrease window opacity",
			command:      &DecreaseWindowOpacityCmd{Uuid: "window-id"},
			wantTemplate: initialTemplate + JS_ADJUST_WINDOW_OPACITY,
			wantParams:   ScriptParams{Uuid: "window-id", Delta: -0.05},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sp := ScriptPackage{ScriptTemplate: initialTemplate}
			if err := test.command.Run(&sp); err != nil {
				t.Fatalf("Run returned an error: %v", err)
			}
			if sp.ScriptTemplate != test.wantTemplate {
				t.Errorf("ScriptTemplate does not match template for %s", test.name)
			}
			if sp.Params != test.wantParams {
				t.Errorf("Params = %+v, want %+v", sp.Params, test.wantParams)
			}
			if sp.Custom != test.wantCustom {
				t.Errorf("Custom = %t, want %t", sp.Custom, test.wantCustom)
			}
			if !sp.Custom {
				if occurrences := strings.Count(sp.ScriptTemplate, "const execute = () => {"); occurrences != 1 {
					t.Errorf("ScriptTemplate contains execute() declaration %d times, want once", occurrences)
				}
			}
		})
	}
}

func TestCommandHelperBundles(t *testing.T) {
	tests := []struct {
		name     string
		command  scriptCommand
		includes []string
		excludes []string
	}{
		{
			name:     "workspace command uses only common header",
			command:  &GetWorkspaceCmd{},
			excludes: []string{"const formatWindowRow = (", "const enumerateTiles = ("},
		},
		{
			name:     "window list uses row formatter",
			command:  &ListCmd{},
			includes: []string{"const formatWindowRow = ("},
			excludes: []string{"const enumerateTiles = ("},
		},
		{
			name:     "tile list uses tile helpers",
			command:  &ListTilesCmd{},
			includes: []string{"const enumerateTiles = (", "const formatRelativeGeometry = ("},
			excludes: []string{"const formatWindowRow = ("},
		},
		{
			name:    "tile window list uses both helper bundles",
			command: &ListTileWindowsCmd{TilePath: "."},
			includes: []string{
				"const formatWindowRow = (",
				"const tileForPath = (",
			},
		},
		{
			name:     "tile resize uses tile helpers",
			command:  &ResizeTileCmd{TilePath: ".", Delta: 1, Edge: "right"},
			includes: []string{"const resizeTileByPercent = ("},
			excludes: []string{"const formatWindowRow = ("},
		},
		{
			name:     "active tile resize uses tile helpers",
			command:  &ResizeActiveTileCmd{Delta: 1, Edge: "right"},
			includes: []string{"const resizeTileByPercent = ("},
			excludes: []string{"const formatWindowRow = ("},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sp := ScriptPackage{ScriptTemplate: JS_HEADER}
			if err := test.command.Run(&sp); err != nil {
				t.Fatalf("Run returned an error: %v", err)
			}

			var script strings.Builder
			if err := prepareScript(&script, sp); err != nil {
				t.Fatalf("prepareScript returned an error: %v", err)
			}
			generated := script.String()

			for _, expected := range test.includes {
				if occurrences := strings.Count(generated, expected); occurrences != 1 {
					t.Errorf("generated script contains %q %d times, want once", expected, occurrences)
				}
			}
			for _, unexpected := range test.excludes {
				if strings.Contains(generated, unexpected) {
					t.Errorf("generated script unexpectedly contains %q", unexpected)
				}
			}
		})
	}
}

func TestResizeCommandValidation(t *testing.T) {
	commands := []struct {
		name     string
		validate func(float64) error
	}{
		{
			name: "resize tile",
			validate: func(delta float64) error {
				return (ResizeTileCmd{Delta: delta}).Validate()
			},
		},
		{
			name: "resize active tile",
			validate: func(delta float64) error {
				return (ResizeActiveTileCmd{Delta: delta}).Validate()
			},
		},
	}
	values := []struct {
		name    string
		delta   float64
		wantErr bool
	}{
		{name: "minimum", delta: -100},
		{name: "negative fraction", delta: -0.5},
		{name: "positive fraction", delta: 0.5},
		{name: "maximum", delta: 100},
		{name: "below minimum", delta: -100.1, wantErr: true},
		{name: "zero", delta: 0, wantErr: true},
		{name: "above maximum", delta: 100.1, wantErr: true},
		{name: "NaN", delta: math.NaN(), wantErr: true},
		{name: "infinity", delta: math.Inf(1), wantErr: true},
	}

	for _, command := range commands {
		for _, value := range values {
			t.Run(command.name+"/"+value.name, func(t *testing.T) {
				err := command.validate(value.delta)
				if (err != nil) != value.wantErr {
					t.Errorf("Validate() error = %v, wantErr %t", err, value.wantErr)
				}
			})
		}
	}
}

func TestSetTileGeometryCommandValidation(t *testing.T) {
	commands := []struct {
		name     string
		validate func(float64, float64, float64, float64) error
	}{
		{
			name: "set tile geometry",
			validate: func(x, y, width, height float64) error {
				return (SetTileGeometryCmd{X: x, Y: y, Width: width, Height: height}).Validate()
			},
		},
		{
			name: "set active tile geometry",
			validate: func(x, y, width, height float64) error {
				return (SetActiveTileGeometryCmd{X: x, Y: y, Width: width, Height: height}).Validate()
			},
		},
	}
	values := []struct {
		name                string
		x, y, width, height float64
		wantErr             bool
	}{
		{name: "minimum", x: 0, y: 0, width: 0, height: 0},
		{name: "maximum", x: 1, y: 1, width: 1, height: 1},
		{name: "x below minimum", x: -0.1, wantErr: true},
		{name: "y above maximum", y: 1.1, wantErr: true},
		{name: "width NaN", width: math.NaN(), wantErr: true},
		{name: "height infinity", height: math.Inf(1), wantErr: true},
	}

	for _, command := range commands {
		for _, value := range values {
			t.Run(command.name+"/"+value.name, func(t *testing.T) {
				err := command.validate(value.x, value.y, value.width, value.height)
				if (err != nil) != value.wantErr {
					t.Errorf("Validate() error = %v, wantErr %t", err, value.wantErr)
				}
			})
		}
	}
}

func TestRunCustomScriptReportsReadError(t *testing.T) {
	scriptFile, err := os.CreateTemp(t.TempDir(), "missing-script-*.js")
	if err != nil {
		t.Fatal(err)
	}
	scriptPath := scriptFile.Name()
	if err := scriptFile.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(scriptPath); err != nil {
		t.Fatal(err)
	}

	command := RunCustomScriptCmd{ScriptFile: scriptFile}
	if err := command.Run(&ScriptPackage{}); err == nil {
		t.Fatal("Run returned nil for an unreadable custom script")
	}
}

func TestJSString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "plain UUID", input: "1234-abcd", want: `"1234-abcd"`},
		{name: "quotes", input: `uuid"with"quotes`, want: `"uuid\"with\"quotes"`},
		{name: "backslash", input: `uuid\path`, want: `"uuid\\path"`},
		{name: "control chars", input: "uuid\n\tvalue", want: `"uuid\n\tvalue"`},
		{name: "line separators", input: "uuid\u2028\u2029value", want: `"uuid\u2028\u2029value"`},
		{name: "template literal", input: "uuid`${malicious}`", want: "\"uuid`${malicious}`\""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := jsString(test.input)
			if err != nil {
				t.Fatalf("jsString(%q) returned an error: %v", test.input, err)
			}
			if got != test.want {
				t.Errorf("jsString(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestPrepareScript(t *testing.T) {
	sp := ScriptPackage{
		ScriptTemplate: `const execute = () => {
	const uuid = {{jsString .Uuid}};
}
`,
		Params: ScriptParams{Uuid: `uuid"with\characters`},
	}

	var script strings.Builder
	if err := prepareScript(&script, sp); err != nil {
		t.Fatalf("prepareScript returned an error: %v", err)
	}

	quotedUUID, err := jsString(sp.Params.Uuid)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(script.String(), "const uuid = "+quotedUUID+";") {
		t.Fatalf("prepared script does not contain escaped UUID:\n%s", script.String())
	}
	if !strings.HasSuffix(script.String(), JS_FOOTER) {
		t.Fatalf("prepared script does not end with JS_FOOTER:\n%s", script.String())
	}
}

func TestPrepareCustomScriptDoesNotAppendFooter(t *testing.T) {
	const customScript = `const value = {{jsString .P1}};
close();
`
	sp := ScriptPackage{
		ScriptTemplate: customScript,
		Params:         ScriptParams{P1: `value"with\characters`},
		Custom:         true,
	}

	var script strings.Builder
	if err := prepareScript(&script, sp); err != nil {
		t.Fatalf("prepareScript returned an error: %v", err)
	}

	quotedValue, err := jsString(sp.Params.P1)
	if err != nil {
		t.Fatal(err)
	}
	want := "const value = " + quotedValue + ";\nclose();\n"
	if got := script.String(); got != want {
		t.Fatalf("prepared custom script = %q, want %q", got, want)
	}
	if strings.Contains(script.String(), "execute();") {
		t.Fatalf("prepared custom script unexpectedly contains the built-in footer:\n%s", script.String())
	}
}

func TestPrepareScriptReportsTemplateErrors(t *testing.T) {
	err := prepareScript(io.Discard, ScriptPackage{ScriptTemplate: "{{"})
	if err == nil || !strings.Contains(err.Error(), "Error parsing script template:") {
		t.Fatalf("prepareScript error = %v, want template parsing error", err)
	}
}

func TestPrepareScriptReportsWriterErrors(t *testing.T) {
	err := prepareScript(errorWriter{}, ScriptPackage{ScriptTemplate: "content"})
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("prepareScript error = %v, want %v", err, io.ErrClosedPipe)
	}
}

func TestUUIDIsEscapedInGeneratedScript(t *testing.T) {
	params := ScriptParams{Uuid: `uuid\"; malicious(); //`}
	tmpl, err := template.New("test").Funcs(template.FuncMap{
		"jsString": jsString,
	}).Parse(JS_ACTIVATE_WINDOW)
	if err != nil {
		t.Fatal(err)
	}

	var script strings.Builder
	if err := tmpl.Execute(&script, params); err != nil {
		t.Fatal(err)
	}

	quotedUUID, err := jsString(params.Uuid)
	if err != nil {
		t.Fatal(err)
	}
	if occurrences := strings.Count(script.String(), quotedUUID); occurrences != 3 {
		t.Fatalf("quoted UUID occurs %d times in generated script, want 3:\n%s", occurrences, script.String())
	}
}

func TestSearchTermIsEscapedInGeneratedScript(t *testing.T) {
	params := ScriptParams{
		SearchTerm:  "term\"` ${malicious} \\\nnext line",
		SearchField: "caption",
	}
	tmpl, err := template.New("test").Funcs(template.FuncMap{
		"jsString": jsString,
	}).Parse(JS_FIND)
	if err != nil {
		t.Fatal(err)
	}

	var script strings.Builder
	if err := tmpl.Execute(&script, params); err != nil {
		t.Fatal(err)
	}

	quotedSearchTerm, err := jsString(params.SearchTerm)
	if err != nil {
		t.Fatal(err)
	}
	generated := script.String()
	expected := "regExp = new RegExp(" + quotedSearchTerm + ", 'i');"
	if !strings.Contains(generated, expected) {
		t.Fatalf("generated script does not contain %q:\n%s", expected, generated)
	}
	if strings.Contains(generated, "String.raw`") {
		t.Fatalf("generated script still uses an unsafe template literal:\n%s", generated)
	}
}

func TestServerComplete(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	server := newServer(&stdout, &stderr)

	if err := server.Complete(23, "first\nsecond", "script failed"); err != nil {
		t.Fatalf("Complete returned a D-Bus error: %v", err)
	}

	if got, want := stdout.String(), "first\nsecond\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got, want := stderr.String(), "script failed\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
	if exitCode := <-server.done; exitCode != 23 {
		t.Fatalf("exit code = %d, want 23", exitCode)
	}
}

func TestServerDBusMethods(t *testing.T) {
	methods := introspect.Methods(newServer(io.Discard, io.Discard))
	if len(methods) != 1 {
		t.Fatalf("exported method count = %d, want 1: %#v", len(methods), methods)
	}

	method := methods[0]
	if method.Name != "Complete" {
		t.Fatalf("exported method = %q, want Complete", method.Name)
	}
	wantArgs := []string{"i", "s", "s"}
	if len(method.Args) != len(wantArgs) {
		t.Fatalf("Complete argument count = %d, want %d", len(method.Args), len(wantArgs))
	}
	for i, arg := range method.Args {
		if arg.Type != wantArgs[i] || arg.Direction != "in" {
			t.Errorf("Complete argument %d = %#v, want input type %q", i, arg, wantArgs[i])
		}
	}
}

func TestWaitForCompletion(t *testing.T) {
	done := make(chan int, 1)
	done <- 1

	if exitCode := waitForCompletion(done, time.Second, io.Discard); exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}
}

func TestWaitForCompletionTimesOut(t *testing.T) {
	var stderr bytes.Buffer
	exitCode := waitForCompletion(make(chan int), time.Nanosecond, &stderr)

	if exitCode != 124 {
		t.Fatalf("exit code = %d, want 124", exitCode)
	}
	if !strings.Contains(stderr.String(), "Timing out") {
		t.Fatalf("timeout message not written to stderr: %q", stderr.String())
	}
}

func TestNormalizeExitCode(t *testing.T) {
	for _, test := range []struct {
		input int
		want  int
	}{
		{input: -1, want: 1},
		{input: 0, want: 0},
		{input: 1, want: 1},
		{input: 124, want: 124},
		{input: 255, want: 255},
		{input: 256, want: 1},
	} {
		if got := normalizeExitCode(test.input); got != test.want {
			t.Errorf("normalizeExitCode(%d) = %d, want %d", test.input, got, test.want)
		}
	}
}
