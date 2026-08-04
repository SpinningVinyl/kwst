package main

import (
	"bytes"
	"errors"
	"io"
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
			wantTemplate: initialTemplate + JS_INCREASE_WINDOW_OPACITY,
			wantParams:   ScriptParams{Uuid: "window-id"},
		},
		{
			name:         "decrease window opacity",
			command:      &DecreaseWindowOpacityCmd{Uuid: "window-id"},
			wantTemplate: initialTemplate + JS_DECREASE_WINDOW_OPACITY,
			wantParams:   ScriptParams{Uuid: "window-id"},
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
		ScriptTemplate: `const uuid = {{jsString .Uuid}};
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

func TestFindHandlesInvalidRegularExpression(t *testing.T) {
	params := ScriptParams{
		SearchTerm:  "[invalid",
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

	generated := script.String()
	quotedSearchTerm, err := jsString(params.SearchTerm)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"let regExp;",
		"try {",
		"regExp = new RegExp(" + quotedSearchTerm + ", 'i');",
		"catch (error)",
		`returnError("Invalid regular expression: " + error.message);`,
		"if (regExp)",
	} {
		if !strings.Contains(generated, expected) {
			t.Errorf("generated script does not contain %q:\n%s", expected, generated)
		}
	}

	errorHandler := strings.Index(generated, `returnError("Invalid regular expression: " + error.message);`)
	searchGuard := strings.Index(generated, "if (regExp)")
	windowSearch := strings.Index(generated, ".search(regExp)")
	if errorHandler < 0 || searchGuard < errorHandler || windowSearch < searchGuard {
		t.Errorf("window search is not protected by the regular-expression guard:\n%s", generated)
	}
}

func TestGetActiveWindowRejectsSpecialWindow(t *testing.T) {
	for _, expected := range []string{
		"const activeWindow = workspace.activeWindow;",
		"if (activeWindow.specialWindow)",
		`returnError("No active regular window");`,
		"returnResult(activeWindow.internalId);",
	} {
		if !strings.Contains(JS_GET_ACTIVE_WINDOW, expected) {
			t.Errorf("JS_GET_ACTIVE_WINDOW does not contain %q", expected)
		}
	}
}

func TestUUIDCommandsGuardMissingWindow(t *testing.T) {
	for _, expected := range []string{
		"const findWindow = (uuid) => {",
		"const window = workspace.windowList().find(",
		"candidate => candidate.internalId == uuid",
		"return window;",
	} {
		if !strings.Contains(JS_HEADER, expected) {
			t.Errorf("JS_HEADER does not contain %q", expected)
		}
	}

	params := ScriptParams{
		Uuid:           `missing\"; malicious(); //`,
		WorkspaceId:    3,
		X:              10,
		Y:              20,
		Width:          640,
		Height:         480,
		WindowProperty: "keepAbove",
		PropertyValue:  "toggle",
	}
	quotedUUID, err := jsString(params.Uuid)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		scriptTemplate string
		action         string
		usesWindowID   bool
	}{
		{name: "get geometry", scriptTemplate: JS_GET_WINDOW_GEOMETRY, action: "returnResult(result);"},
		{name: "activate", scriptTemplate: JS_ACTIVATE_WINDOW, action: "workspace.activeWindow = targetWindow;"},
		{name: "set size", scriptTemplate: JS_SET_WINDOW_SIZE, action: "updateWindowGeometry(targetWindow, {", usesWindowID: true},
		{name: "set position", scriptTemplate: JS_SET_WINDOW_POSITION, action: "updateWindowGeometry(targetWindow, {", usesWindowID: true},
		{name: "set geometry", scriptTemplate: JS_SET_WINDOW_GEOMETRY, action: "updateWindowGeometry(targetWindow, {", usesWindowID: true},
		{name: "set workspace", scriptTemplate: JS_SET_WINDOW_WORKSPACE, action: "targetWindow.desktops = [targetWorkspace];"},
		{name: "set property", scriptTemplate: JS_SET_WINDOW_PROPERTY, action: "targetWindow.keepAbove = !targetWindow.keepAbove;"},
		{name: "close", scriptTemplate: JS_CLOSE_WINDOW, action: "targetWindow.closeWindow();"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tmpl, err := template.New("test").Funcs(template.FuncMap{
				"jsString": jsString,
			}).Parse(test.scriptTemplate)
			if err != nil {
				t.Fatal(err)
			}

			var script strings.Builder
			if err := tmpl.Execute(&script, params); err != nil {
				t.Fatal(err)
			}
			generated := script.String()
			expectedParts := []string{"if (!targetWindow)", test.action}
			if test.usesWindowID {
				expectedParts = append(expectedParts,
					"const windowId = "+quotedUUID+";",
					"const targetWindow = findWindow(windowId);",
					"returnError(ERROR_WINDOW_NOT_FOUND + windowId);",
				)
			} else {
				expectedParts = append(expectedParts,
					"const targetWindow = findWindow("+quotedUUID+");",
					`returnError(ERROR_WINDOW_NOT_FOUND + `+quotedUUID+`);`,
				)
			}
			for _, expected := range expectedParts {
				if !strings.Contains(generated, expected) {
					t.Errorf("generated script does not contain %q:\n%s", expected, generated)
				}
			}

			guard := strings.Index(generated, "if (!targetWindow)")
			action := strings.Index(generated, test.action)
			if guard < 0 || action < guard {
				t.Errorf("window action is not protected by the missing-window guard:\n%s", generated)
			}
		})
	}
}

func TestServerCloseWithStatus(t *testing.T) {
	server := newServer(io.Discard, io.Discard)
	if err := server.CloseWithStatus(1); err != nil {
		t.Fatalf("CloseWithStatus returned a D-Bus error: %v", err)
	}

	if exitCode := <-server.done; exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}
}

func TestCloseWithStatusDBusSignature(t *testing.T) {
	for _, method := range introspect.Methods(newServer(io.Discard, io.Discard)) {
		if method.Name != "CloseWithStatus" {
			continue
		}
		if len(method.Args) != 1 || method.Args[0].Type != "i" || method.Args[0].Direction != "in" {
			t.Fatalf("CloseWithStatus D-Bus arguments = %#v, want one INT32 input", method.Args)
		}
		return
	}
	t.Fatal("CloseWithStatus is not exported over D-Bus")
}

func TestLegacyCloseReportsPriorScriptError(t *testing.T) {
	server := newServer(io.Discard, io.Discard)
	if err := server.Msg("error", "invalid workspace"); err != nil {
		t.Fatalf("Msg returned a D-Bus error: %v", err)
	}
	if err := server.Close(); err != nil {
		t.Fatalf("Close returned a D-Bus error: %v", err)
	}

	if exitCode := <-server.done; exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
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

func TestGeneratedScriptReportsExitStatus(t *testing.T) {
	for _, expected := range []string{
		"let exitCode = 0;",
		"exitCode = 1;",
		`"CloseWithStatus", exitCode`,
	} {
		if !strings.Contains(JS_HEADER, expected) {
			t.Errorf("JS_HEADER does not contain %q", expected)
		}
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
