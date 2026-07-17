package main

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/godbus/dbus/v5/introspect"
)

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
	if occurrences := strings.Count(script.String(), quotedUUID); occurrences != 2 {
		t.Fatalf("quoted UUID occurs %d times in generated script, want 2:\n%s", occurrences, script.String())
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

func TestPreviousWindowGuardsShortWindowStack(t *testing.T) {
	for _, expected := range []string{
		"if (windowStack.length < 2)",
		`returnError("No previous window available");`,
		"workspace.activeWindow = windowStack[windowStack.length - 2];",
	} {
		if !strings.Contains(JS_PREVIOUS_WINDOW, expected) {
			t.Errorf("JS_PREVIOUS_WINDOW does not contain %q", expected)
		}
	}

	guard := strings.Index(JS_PREVIOUS_WINDOW, "if (windowStack.length < 2)")
	activation := strings.Index(JS_PREVIOUS_WINDOW, "workspace.activeWindow =")
	if guard < 0 || activation < guard {
		t.Error("previous-window activation is not protected by the short-stack guard")
	}
}

func TestSetWindowWorkspaceGuardsMissingWindow(t *testing.T) {
	params := ScriptParams{
		Uuid:        `missing\"; malicious(); //`,
		WorkspaceId: 3,
	}
	tmpl, err := template.New("test").Funcs(template.FuncMap{
		"jsString": jsString,
	}).Parse(JS_SET_WINDOW_WORKSPACE)
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
	generated := script.String()
	for _, expected := range []string{
		"const targetWindow = workspace.windowList().find(",
		"window.internalId == " + quotedUUID,
		"if (!targetWindow)",
		`returnError("Window not found: " + ` + quotedUUID + `);`,
		"targetWindow.desktops = [targetWorkspace];",
	} {
		if !strings.Contains(generated, expected) {
			t.Errorf("generated script does not contain %q:\n%s", expected, generated)
		}
	}
	if strings.Contains(generated, "var w = allWindows[i]") {
		t.Errorf("generated script still uses the unsafe fall-through window lookup:\n%s", generated)
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
