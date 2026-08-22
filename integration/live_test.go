//go:build integration

package integration

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	integrationEnvironment    = "KWST_INTEGRATION"
	commandTimeout            = 10 * time.Second
	stateTimeout              = 10 * time.Second
	pollInterval              = 100 * time.Millisecond
	customScriptMarker        = "// your code goes here"
	previousWindowPackageID   = "net.prsv.kwst.previouswindow"
	previousWindowShortcut    = "[KWST] Switch to previously active window"
	globalAccelService        = "org.kde.kglobalaccel"
	globalAccelComponentPath  = dbus.ObjectPath("/component/kwin")
	globalAccelComponentIFace = "org.kde.kglobalaccel.Component"
)

const createWorkspaceScript = `const desktopName = {{jsString .P1}};
const existingDesktopIds = workspace.desktops.map((desktop) => desktop.id);
workspace.createDesktop(workspace.desktops.length, desktopName);
const createdDesktop = workspace.desktops.find(
    (desktop) => desktop.name == desktopName && !existingDesktopIds.includes(desktop.id)
);
if (createdDesktop) {
    returnResult(createdDesktop.x11DesktopNumber);
} else {
    returnError("Failed to create temporary workspace: " + desktopName);
}`

const removeWorkspaceScript = `const desktopName = {{jsString .P1}};
const temporaryDesktop = workspace.desktops.find((desktop) => desktop.name == desktopName);
if (!temporaryDesktop) {
    returnResult("Temporary workspace already removed");
} else if (workspace.currentDesktop == temporaryDesktop) {
    returnError("Cannot remove the active temporary workspace: " + desktopName);
} else {
    workspace.removeDesktop(temporaryDesktop);
    const stillExists = workspace.desktops.some((desktop) => desktop.name == desktopName);
    if (stillExists) {
        returnError("Failed to remove temporary workspace: " + desktopName);
    } else {
        returnResult("Temporary workspace removed");
    }
}`

const activeWindowOutputScript = `const activeWindow = workspace.activeWindow;
if (activeWindow) {
    returnResult(activeWindow.output.name);
} else {
    returnError("No active window");
}`

type commandResult struct {
	stdout   string
	stderr   string
	exitCode int
}

func (r commandResult) String() string {
	return fmt.Sprintf("exit code: %d\nstdout: %q\nstderr: %q", r.exitCode, r.stdout, r.stderr)
}

type geometry struct {
	x      float64
	y      float64
	width  float64
	height float64
}

type tileRow struct {
	output           string
	path             string
	tileType         string
	relativeGeometry geometry
	windowCount      int
}

type fixtureWindow struct {
	title string
	uuid  string
	cmd   *exec.Cmd
	done  chan error
}

func TestKWinWorkflow(t *testing.T) {
	kdialog := requireIntegrationEnvironment(t)

	kwst := buildKWST(t)
	t.Run("find rejects invalid regular expression", func(t *testing.T) {
		result := runKWST(t, kwst, "find", "[")
		if result.exitCode != 1 {
			t.Fatalf("find with an invalid regular expression returned exit code %d, want 1:\n%s", result.exitCode, result.String())
		}
		if !strings.Contains(result.stderr, "Invalid regular expression") {
			t.Fatalf("find with an invalid regular expression did not report the error:\n%s", result.String())
		}
	})
	t.Run("custom script reports unhandled exception", func(t *testing.T) {
		const failureMessage = "intentional custom script failure"
		script := writeCustomScript(t, `throw new Error("`+failureMessage+`");`)
		result := runKWST(t, kwst, "run-custom-script", script)

		if result.exitCode != 1 {
			t.Fatalf("custom script returned exit code %d, want 1:\n%s", result.exitCode, result.String())
		}
		if result.stdout != "" {
			t.Fatalf("custom script returned unexpected stdout %q", result.stdout)
		}
		for _, expected := range []string{
			"KWin script returned an error: Error executing KWin script:",
			failureMessage,
		} {
			if !strings.Contains(result.stderr, expected) {
				t.Fatalf("custom script did not report %q:\n%s", expected, result.String())
			}
		}
	})
	t.Run("UUID commands reject missing windows", func(t *testing.T) {
		missingUUID := fmt.Sprintf("kwst-missing-window-%d-%d", os.Getpid(), time.Now().UnixNano())
		tests := []struct {
			name      string
			arguments []string
		}{
			{name: "get-window-geometry", arguments: []string{"get-window-geometry", missingUUID}},
			{name: "activate-window", arguments: []string{"activate-window", missingUUID}},
			{name: "set-window-size", arguments: []string{"set-window-size", missingUUID, "640", "480"}},
			{name: "set-window-position", arguments: []string{"set-window-position", missingUUID, "10", "20"}},
			{name: "set-window-geometry", arguments: []string{"set-window-geometry", missingUUID, "10", "20", "640", "480"}},
			{name: "set-window-geometry-relative", arguments: []string{"set-window-geometry-relative", missingUUID, "10", "20", "50", "50"}},
			{name: "set-window-size-relative", arguments: []string{"set-window-size-relative", missingUUID, "50", "50"}},
			{name: "set-window-position-relative", arguments: []string{"set-window-position-relative", missingUUID, "10", "20"}},
			{name: "set-window-workspace", arguments: []string{"set-window-workspace", missingUUID, "1"}},
			{name: "set-window-property", arguments: []string{"set-window-property", "--property=keepAbove", "--value=true", missingUUID}},
			{name: "close-window", arguments: []string{"close-window", missingUUID}},
			{name: "get-window-opacity", arguments: []string{"get-window-opacity", missingUUID}},
			{name: "set-window-opacity", arguments: []string{"set-window-opacity", missingUUID, "0.5"}},
			{name: "increase-window-opacity", arguments: []string{"increase-window-opacity", missingUUID}},
			{name: "decrease-window-opacity", arguments: []string{"decrease-window-opacity", missingUUID}},
			{name: "get-window-tile", arguments: []string{"get-window-tile", missingUUID}},
			{name: "set-window-tile", arguments: []string{"set-window-tile", missingUUID, "."}},
			{name: "unset-window-tile", arguments: []string{"unset-window-tile", missingUUID}},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				result := runKWST(t, kwst, test.arguments...)
				if result.exitCode != 1 {
					t.Fatalf("command returned exit code %d, want 1:\n%s", result.exitCode, result.String())
				}
				expectedError := "Window not found: " + missingUUID
				if !strings.Contains(result.stderr, expectedError) {
					t.Fatalf("command did not report %q:\n%s", expectedError, result.String())
				}
			})
		}
	})

	originalWorkspace := getWorkspace(t, kwst)

	runID := fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
	first := startFixtureWindow(t, kdialog, "kwst-integration-"+runID+"-one")
	second := startFixtureWindow(t, kdialog, "kwst-integration-"+runID+"-two")

	waitForFixtureWindows(t, kwst, first, second)

	activateAndVerify(t, kwst, first)
	activateAndVerify(t, kwst, second)

	outputNames := testOutputCommands(t, kwst)
	t.Run("native tile commands", func(t *testing.T) {
		testNativeTileCommands(t, kwst, first, outputNames)
	})
	t.Run("window opacity commands", func(t *testing.T) {
		testWindowOpacityCommands(t, kwst, first)
	})
	t.Run("relative geometry commands", func(t *testing.T) {
		testRelativeGeometryCommands(t, kwst, first)
	})

	resizeAndMoveFixture(t, kwst, first)

	t.Run("change active workspace", func(t *testing.T) {
		targetWorkspace := 1
		if originalWorkspace == 1 {
			targetWorkspace = 2
		}

		result := runKWST(t, kwst, "set-workspace", strconv.Itoa(targetWorkspace))
		temporaryWorkspaceName := "kwst-integration-workspace-" + runID
		removeTemporaryWorkspace := false
		t.Cleanup(func() {
			if !removeTemporaryWorkspace {
				return
			}
			if restoreResult := runKWST(t, kwst, "set-workspace", strconv.Itoa(originalWorkspace)); restoreResult.exitCode != 0 {
				t.Errorf("restore original workspace during cleanup:\n%s", restoreResult.String())
			}
			if removalResult := removeWorkspace(t, kwst, temporaryWorkspaceName); removalResult.exitCode != 0 {
				t.Errorf("remove temporary workspace during cleanup:\n%s", removalResult.String())
			}
		})

		if result.exitCode != 0 && originalWorkspace == 1 && strings.Contains(result.stderr, "Invalid workspace number") {
			removeTemporaryWorkspace = true
			targetWorkspace = createWorkspace(t, kwst, temporaryWorkspaceName)
			result = runKWST(t, kwst, "set-workspace", strconv.Itoa(targetWorkspace))
		}
		requireSuccess(t, result, "change active workspace")

		eventually(t, "the target workspace to become active", func() (bool, string) {
			workspace, result := readWorkspace(t, kwst)
			return result.exitCode == 0 && workspace == targetWorkspace, result.String()
		})

		requireSuccess(t, runKWST(t, kwst, "set-workspace", strconv.Itoa(originalWorkspace)), "restore original workspace")
		eventually(t, "the original workspace to become active again", func() (bool, string) {
			workspace, result := readWorkspace(t, kwst)
			return result.exitCode == 0 && workspace == originalWorkspace, result.String()
		})

		if removeTemporaryWorkspace {
			requireSuccess(t, removeWorkspace(t, kwst, temporaryWorkspaceName), "remove temporary workspace")
			removeTemporaryWorkspace = false
		}
	})

	closeFixtureWindows(t, kwst, first, second)
}

func TestPreviousWindowScript(t *testing.T) {
	kdialog := requireIntegrationEnvironment(t)

	if !kwinPackageInstalled(t, previousWindowPackageID) {
		t.Skipf("optional KWin package %q is not installed", previousWindowPackageID)
	}

	connection, err := dbus.ConnectSessionBus()
	if err != nil {
		t.Fatalf("connect to session D-Bus: %v", err)
	}
	defer connection.Close()

	component := connection.Object(globalAccelService, globalAccelComponentPath)
	if !shortcutRegistered(t, component, previousWindowShortcut) {
		t.Skipf("optional KWin shortcut %q is not registered", previousWindowShortcut)
	}

	kwst := buildKWST(t)
	runID := fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
	first := startFixtureWindow(t, kdialog, "kwst-previous-window-"+runID+"-one")
	second := startFixtureWindow(t, kdialog, "kwst-previous-window-"+runID+"-two")

	waitForFixtureWindows(t, kwst, first, second)
	activateAndVerify(t, kwst, first)
	activateAndVerify(t, kwst, second)

	if call := component.Call(globalAccelComponentIFace+".invokeShortcut", 0, previousWindowShortcut); call.Err != nil {
		t.Fatalf("invoke optional previous-window shortcut: %v", call.Err)
	}
	eventually(t, "the first fixture to become active through the previous-window shortcut", func() (bool, string) {
		result := runKWST(t, kwst, "get-active-window")
		return result.exitCode == 0 && result.stdout == first.uuid, result.String()
	})

	closeFixtureWindows(t, kwst, first, second)
}

func requireIntegrationEnvironment(t *testing.T) string {
	t.Helper()

	if os.Getenv(integrationEnvironment) != "1" {
		t.Skipf("set %s=1 to run tests against the current KWin session", integrationEnvironment)
	}
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		t.Fatal("DBUS_SESSION_BUS_ADDRESS is not set")
	}
	if os.Getenv("WAYLAND_DISPLAY") == "" && os.Getenv("DISPLAY") == "" {
		t.Fatal("neither WAYLAND_DISPLAY nor DISPLAY is set")
	}

	kdialog, err := exec.LookPath("kdialog")
	if err != nil {
		t.Fatal("KDialog is not installed or is not available in PATH")
	}
	return kdialog
}

func kwinPackageInstalled(t *testing.T, packageID string) bool {
	t.Helper()

	kpackageTool, err := exec.LookPath("kpackagetool6")
	if err != nil {
		t.Log("cannot verify optional KWin package: kpackagetool6 is not available in PATH")
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, kpackageTool, "--type=KWin/Script", "--show", packageID)
	if output, err := cmd.CombinedOutput(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			t.Fatalf("checking optional KWin package exceeded %s", commandTimeout)
		}
		t.Logf("optional KWin package check failed: %v: %s", err, strings.TrimSpace(string(output)))
		return false
	}
	return true
}

func shortcutRegistered(t *testing.T, component dbus.BusObject, shortcutName string) bool {
	t.Helper()

	var shortcutNames []string
	if call := component.Call(globalAccelComponentIFace+".shortcutNames", 0); call.Err != nil {
		t.Fatalf("list KGlobalAccel shortcuts: %v", call.Err)
	} else if err := call.Store(&shortcutNames); err != nil {
		t.Fatalf("decode KGlobalAccel shortcut names: %v", err)
	}
	for _, registeredName := range shortcutNames {
		if registeredName == shortcutName {
			return true
		}
	}
	return false
}

func waitForFixtureWindows(t *testing.T, kwst string, first, second *fixtureWindow) {
	t.Helper()

	eventually(t, "both KDialog fixtures to appear in kwst list", func() (bool, string) {
		result := runKWST(t, kwst, "list", "--show-captions")
		if result.exitCode != 0 {
			return false, result.String()
		}
		first.uuid = windowUUID(result.stdout, first.title)
		second.uuid = windowUUID(result.stdout, second.title)
		return first.uuid != "" && second.uuid != "", result.stdout
	})
}

func closeFixtureWindows(t *testing.T, kwst string, first, second *fixtureWindow) {
	t.Helper()

	requireSuccess(t, runKWST(t, kwst, "close-window", first.uuid), "close first fixture")
	requireSuccess(t, runKWST(t, kwst, "close-window", second.uuid), "close second fixture")
	eventually(t, "both fixtures to disappear from kwst list", func() (bool, string) {
		result := runKWST(t, kwst, "list", "--show-captions")
		if result.exitCode != 0 {
			return false, result.String()
		}
		closed := windowUUID(result.stdout, first.title) == "" && windowUUID(result.stdout, second.title) == ""
		return closed, result.stdout
	})
}

func buildKWST(t *testing.T) string {
	t.Helper()

	binary := filepath.Join(t.TempDir(), "kwst")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "build", "-o", binary, ".")
	cmd.Dir = repositoryRoot(t)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build kwst: %v\n%s", err, output)
	}
	return binary
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("determine integration test source path")
	}
	return filepath.Dir(filepath.Dir(currentFile))
}

func createWorkspace(t *testing.T, kwst, name string) int {
	t.Helper()
	script := writeCustomScript(t, createWorkspaceScript)
	result := runKWST(t, kwst, "run-custom-script", "--parameter-1="+name, script)
	requireSuccess(t, result, "create temporary workspace")
	workspace, err := strconv.Atoi(result.stdout)
	if err != nil {
		t.Fatalf("parse temporary workspace number %q: %v", result.stdout, err)
	}
	return workspace
}

func removeWorkspace(t *testing.T, kwst, name string) commandResult {
	t.Helper()
	script := writeCustomScript(t, removeWorkspaceScript)
	return runKWST(t, kwst, "run-custom-script", "--parameter-1="+name, script)
}

func writeCustomScript(t *testing.T, body string) string {
	t.Helper()
	templatePath := filepath.Join(repositoryRoot(t), "custom-script-template.js")
	scriptTemplate, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatalf("read custom script template: %v", err)
	}
	if strings.Count(string(scriptTemplate), customScriptMarker) != 1 {
		t.Fatalf("custom script template must contain exactly one %q marker", customScriptMarker)
	}
	script := strings.Replace(string(scriptTemplate), customScriptMarker, body, 1)
	scriptPath := filepath.Join(t.TempDir(), "workspace.js")
	if err := os.WriteFile(scriptPath, []byte(script), 0o600); err != nil {
		t.Fatalf("write custom workspace script: %v", err)
	}
	return scriptPath
}

func startFixtureWindow(t *testing.T, kdialog, title string) *fixtureWindow {
	t.Helper()

	textFile := filepath.Join(t.TempDir(), "fixture.txt")
	if err := os.WriteFile(textFile, []byte("kwst integration test fixture\n"), 0o600); err != nil {
		t.Fatalf("create KDialog text file: %v", err)
	}

	fixture := &fixtureWindow{
		title: title,
		done:  make(chan error, 1),
	}
	fixture.cmd = exec.Command(kdialog, "--title", title, "--textbox", textFile, "480", "320")
	var stderr bytes.Buffer
	fixture.cmd.Stderr = &stderr
	if err := fixture.cmd.Start(); err != nil {
		t.Fatalf("start KDialog fixture %q: %v", title, err)
	}
	go func() {
		fixture.done <- fixture.cmd.Wait()
	}()

	t.Cleanup(func() {
		if err := fixture.cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			t.Logf("kill KDialog fixture %q: %v", title, err)
		}
		select {
		case err := <-fixture.done:
			if err != nil {
				var exitError *exec.ExitError
				if !errors.As(err, &exitError) {
					t.Logf("wait for KDialog fixture %q: %v; stderr: %s", title, err, stderr.String())
				}
			}
		case <-time.After(2 * time.Second):
			t.Logf("timed out waiting for KDialog fixture %q to exit", title)
		}
	})

	return fixture
}

func activateAndVerify(t *testing.T, kwst string, fixture *fixtureWindow) {
	t.Helper()
	requireSuccess(t, runKWST(t, kwst, "activate-window", fixture.uuid), "activate fixture "+fixture.title)
	eventually(t, fixture.title+" to become active", func() (bool, string) {
		result := runKWST(t, kwst, "get-active-window")
		return result.exitCode == 0 && result.stdout == fixture.uuid, result.String()
	})
}

func testOutputCommands(t *testing.T, kwst string) []string {
	t.Helper()

	var outputNames []string
	if !t.Run("list outputs", func(t *testing.T) {
		result := runKWST(t, kwst, "list-outputs")
		requireSuccess(t, result, "list outputs")

		var err error
		outputNames, err = parseOutputNames(result.stdout)
		if err != nil {
			t.Fatal(err)
		}
	}) {
		t.FailNow()
	}

	knownOutputs := make(map[string]struct{}, len(outputNames))
	for _, name := range outputNames {
		knownOutputs[name] = struct{}{}
	}

	for _, command := range []string{"get-active-output", "get-cursor-output"} {
		t.Run(command+" returns one output", func(t *testing.T) {
			result := runKWST(t, kwst, command)
			name := requireSingleOutputName(t, result, command)
			if _, exists := knownOutputs[name]; !exists {
				t.Fatalf("%s returned output %q, which was not reported by list-outputs", command, name)
			}
		})
	}

	t.Run("get active window output", func(t *testing.T) {
		result := runKWST(t, kwst, "get-active-window-output")
		actual := requireSingleOutputName(t, result, "get active window output")

		script := writeCustomScript(t, activeWindowOutputScript)
		expectedResult := runKWST(t, kwst, "run-custom-script", script)
		expected := requireSingleOutputName(t, expectedResult, "get active window output through KWin")

		if actual != expected {
			t.Fatalf("get-active-window-output returned %q, want %q", actual, expected)
		}
	})

	for _, test := range []struct {
		name      string
		arguments []string
	}{
		{
			name:      "get full output geometry",
			arguments: []string{"get-output-geometry", outputNames[0]},
		},
		{
			name:      "get client output geometry",
			arguments: []string{"get-output-geometry", "--client-area", outputNames[0]},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := runKWST(t, kwst, test.arguments...)
			requireSuccess(t, result, test.name)
			value, err := parseGeometry(result.stdout)
			if err != nil {
				t.Fatalf("%s returned invalid geometry: %v", test.name, err)
			}
			if value.width <= 0 || value.height <= 0 {
				t.Fatalf("%s returned non-positive dimensions: %+v", test.name, value)
			}
		})
	}

	return outputNames
}

func testNativeTileCommands(t *testing.T, kwst string, fixture *fixtureWindow, outputNames []string) {
	t.Helper()

	for _, outputName := range outputNames {
		t.Run("list tiles on "+outputName, func(t *testing.T) {
			result := runKWST(t, kwst, "list-tiles", "--output="+outputName)
			requireSuccess(t, result, "list tiles on output "+outputName)
			rows, err := parseTileRows(result.stdout)
			if err != nil {
				t.Fatal(err)
			}

			for _, row := range rows {
				if row.output == outputName && row.path == "." {
					return
				}
			}
			t.Fatalf("list-tiles did not return the root tile for output %q:\n%s", outputName, result.String())
		})
	}

	missingOutput := fmt.Sprintf("kwst-missing-output-%d-%d", os.Getpid(), time.Now().UnixNano())
	for _, outputName := range outputNames {
		if outputName == missingOutput {
			missingOutput += "-missing"
		}
	}
	for _, test := range []struct {
		name      string
		arguments []string
	}{
		{
			name:      "list-tiles rejects missing output",
			arguments: []string{"list-tiles", "--output=" + missingOutput},
		},
		{
			name:      "set-window-tile rejects missing output",
			arguments: []string{"set-window-tile", "--output=" + missingOutput, fixture.uuid, "."},
		},
		{
			name:      "list-tile-windows rejects missing output",
			arguments: []string{"list-tile-windows", "--output=" + missingOutput, "."},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := runKWST(t, kwst, test.arguments...)
			if result.exitCode != 1 {
				t.Fatalf("command returned exit code %d, want 1:\n%s", result.exitCode, result.String())
			}
			expectedError := "Output not found: " + missingOutput
			if !strings.Contains(result.stderr, expectedError) {
				t.Fatalf("command did not report %q:\n%s", expectedError, result.String())
			}
		})
	}

	activateAndVerify(t, kwst, fixture)
	requireSuccess(t, runKWST(t, kwst, "unset-window-tile", fixture.uuid), "ensure fixture is initially untiled")
	t.Cleanup(func() {
		if result := runKWST(t, kwst, "unset-window-tile", fixture.uuid); result.exitCode != 0 {
			t.Errorf("untile fixture during cleanup:\n%s", result.String())
		}
	})

	initialResult := runKWST(t, kwst, "list-tiles")
	requireSuccess(t, initialResult, "list tiles on active output")
	initialRows, err := parseTileRows(initialResult.stdout)
	if err != nil {
		t.Fatal(err)
	}
	var targetTile tileRow
	resizeEdge := ""
	leavesFound := 0
	for _, row := range initialRows {
		if row.tileType == "Leaf" {
			leavesFound++
			if edge, ok := resizableTileEdge(row.relativeGeometry); ok && resizeEdge == "" {
				targetTile = row
				resizeEdge = edge
			}
		}
	}
	if leavesFound == 0 {
		t.Fatal("list-tiles returned no leaf tiles")
	}
	if resizeEdge == "" {
		t.Fatal("list-tiles returned no leaf tile with an interior edge")
	}

	t.Run("resize tile", func(t *testing.T) {
		verifyTileGeometryChange(t, kwst, targetTile,
			[]string{"resize-tile", "--output=" + targetTile.output, targetTile.path, "1", resizeEdge},
			[]string{"resize-tile", "--output=" + targetTile.output, "--", targetTile.path, "-1", resizeEdge},
		)
	})

	originalGeometryArguments := formatTileGeometryArguments(targetTile.relativeGeometry)
	targetGeometryArguments := formatTileGeometryArguments(moveTileInteriorEdge(targetTile.relativeGeometry, resizeEdge))
	t.Run("set tile geometry", func(t *testing.T) {
		verifyTileGeometryChange(t, kwst, targetTile,
			append([]string{"set-tile-geometry", "--output=" + targetTile.output, targetTile.path}, targetGeometryArguments...),
			append([]string{"set-tile-geometry", "--output=" + targetTile.output, targetTile.path}, originalGeometryArguments...),
		)
	})

	requireSuccess(t, runKWST(t, kwst, "set-window-tile", fixture.uuid, targetTile.path), "assign fixture to tile")
	eventually(t, "the assigned tile's window count to increase", func() (bool, string) {
		result := runKWST(t, kwst, "list-tiles")
		if result.exitCode != 0 {
			return false, result.String()
		}
		rows, err := parseTileRows(result.stdout)
		if err != nil {
			return false, err.Error()
		}
		for _, row := range rows {
			if row.output == targetTile.output && row.path == targetTile.path {
				return row.windowCount == targetTile.windowCount+1,
					fmt.Sprintf("tile %s window count is %d, want %d", row.path, row.windowCount, targetTile.windowCount+1)
			}
		}
		return false, fmt.Sprintf("tile %s on output %s was not returned", targetTile.path, targetTile.output)
	})

	tileResult := runKWST(t, kwst, "get-window-tile", fixture.uuid)
	requireSuccess(t, tileResult, "get fixture tile")
	actualOutput, actualPath, err := parseWindowTile(tileResult.stdout)
	if err != nil {
		t.Fatal(err)
	}
	if actualOutput != targetTile.output || actualPath != targetTile.path {
		t.Fatalf("get-window-tile returned output %q and path %q, want output %q and path %q",
			actualOutput, actualPath, targetTile.output, targetTile.path)
	}

	windowsResult := runKWST(t, kwst, "list-tile-windows", "--output="+targetTile.output, targetTile.path)
	requireSuccess(t, windowsResult, "list windows assigned to fixture tile")
	if !containsWindowUUID(windowsResult.stdout, fixture.uuid) {
		t.Fatalf("list-tile-windows did not return fixture %q:\n%s", fixture.uuid, windowsResult.String())
	}

	activateAndVerify(t, kwst, fixture)
	t.Run("resize active tile", func(t *testing.T) {
		verifyTileGeometryChange(t, kwst, targetTile,
			[]string{"resize-active-tile", "1", resizeEdge},
			[]string{"resize-tile", "--output=" + targetTile.output, "--", targetTile.path, "-1", resizeEdge},
		)
	})
	t.Run("set active tile geometry", func(t *testing.T) {
		verifyTileGeometryChange(t, kwst, targetTile,
			append([]string{"set-active-tile-geometry"}, targetGeometryArguments...),
			append([]string{"set-active-tile-geometry"}, originalGeometryArguments...),
		)
	})

	requireSuccess(t, runKWST(t, kwst, "unset-window-tile", fixture.uuid), "remove fixture from tile")
	untiledResult := runKWST(t, kwst, "get-window-tile", fixture.uuid)
	if untiledResult.exitCode != 1 {
		t.Fatalf("get-window-tile returned exit code %d for an untiled fixture, want 1:\n%s",
			untiledResult.exitCode, untiledResult.String())
	}
	expectedMessage := "Window does not appear to be tiled: " + fixture.uuid
	if !strings.Contains(untiledResult.stderr, expectedMessage) {
		t.Fatalf("get-window-tile did not report %q after unsetting the tile:\n%s", expectedMessage, untiledResult.String())
	}
}

func resizableTileEdge(geometry geometry) (string, bool) {
	const boundaryTolerance = 1e-6
	if geometry.x+geometry.width < 1-boundaryTolerance {
		return "right", true
	}
	if geometry.y+geometry.height < 1-boundaryTolerance {
		return "bottom", true
	}
	if geometry.x > boundaryTolerance {
		return "left", true
	}
	if geometry.y > boundaryTolerance {
		return "top", true
	}
	return "", false
}

func moveTileInteriorEdge(value geometry, edge string) geometry {
	const maximumDelta = 0.01
	switch edge {
	case "right":
		value.width += min(maximumDelta, 1-value.x-value.width)
	case "bottom":
		value.height += min(maximumDelta, 1-value.y-value.height)
	case "left":
		delta := min(maximumDelta, value.x)
		value.x -= delta
		value.width += delta
	case "top":
		delta := min(maximumDelta, value.y)
		value.y -= delta
		value.height += delta
	}
	return value
}

func formatTileGeometryArguments(value geometry) []string {
	return []string{
		strconv.FormatFloat(value.x, 'f', -1, 64),
		strconv.FormatFloat(value.y, 'f', -1, 64),
		strconv.FormatFloat(value.width, 'f', -1, 64),
		strconv.FormatFloat(value.height, 'f', -1, 64),
	}
}

func verifyTileGeometryChange(t *testing.T, kwst string, tile tileRow, changeArguments, restoreArguments []string) {
	t.Helper()

	initial, err := readTileGeometry(t, kwst, tile.output, tile.path)
	if err != nil {
		t.Fatal(err)
	}
	restoreNeeded := false
	t.Cleanup(func() {
		if restoreNeeded {
			if result := runKWST(t, kwst, restoreArguments...); result.exitCode != 0 {
				t.Errorf("restore tile geometry during cleanup:\n%s", result.String())
			}
		}
	})

	requireSuccess(t, runKWST(t, kwst, changeArguments...), strings.Join(changeArguments, " "))
	restoreNeeded = true
	eventually(t, "tile geometry to change", func() (bool, string) {
		current, err := readTileGeometry(t, kwst, tile.output, tile.path)
		if err != nil {
			return false, err.Error()
		}
		return current != initial, fmt.Sprintf("tile geometry is still %+v", current)
	})

	requireSuccess(t, runKWST(t, kwst, restoreArguments...), "restore tile geometry")
	restoreNeeded = false
	eventually(t, "tile geometry to be restored", func() (bool, string) {
		current, err := readTileGeometry(t, kwst, tile.output, tile.path)
		if err != nil {
			return false, err.Error()
		}
		return current == initial, fmt.Sprintf("tile geometry is %+v, want %+v", current, initial)
	})
}

func readTileGeometry(t *testing.T, kwst, output, path string) (geometry, error) {
	t.Helper()
	result := runKWST(t, kwst, "list-tiles", "--output="+output)
	if result.exitCode != 0 {
		return geometry{}, fmt.Errorf("list tiles on output %s: %s", output, result.String())
	}
	rows, err := parseTileRows(result.stdout)
	if err != nil {
		return geometry{}, err
	}
	for _, row := range rows {
		if row.output == output && row.path == path {
			return row.relativeGeometry, nil
		}
	}
	return geometry{}, fmt.Errorf("tile %s on output %s was not returned", path, output)
}

func testWindowOpacityCommands(t *testing.T, kwst string, fixture *fixtureWindow) {
	t.Helper()

	original := getOpacity(t, kwst, fixture.uuid)
	if math.IsNaN(original) || math.IsInf(original, 0) || original < 0.0 || original > 1.0 {
		t.Fatalf("get-window-opacity returned %g, want a value between 0.0 and 1.0", original)
	}
	if original < 0.1 {
		t.Fatalf("fixture opacity %g cannot be restored with set-window-opacity", original)
	}
	t.Cleanup(func() {
		setOpacityAndWait(t, kwst, fixture.uuid, original)
	})

	t.Run("set rejects invalid values", func(t *testing.T) {
		tests := []struct {
			name      string
			arguments []string
		}{
			{name: "missing value", arguments: []string{"set-window-opacity", fixture.uuid}},
			{name: "not a number", arguments: []string{"set-window-opacity", fixture.uuid, "not-a-number"}},
			{name: "zero", arguments: []string{"set-window-opacity", fixture.uuid, "0.0"}},
			{name: "below minimum", arguments: []string{"set-window-opacity", fixture.uuid, "0.099"}},
			{name: "negative", arguments: []string{"set-window-opacity", fixture.uuid, "-0.5"}},
			{name: "above maximum", arguments: []string{"set-window-opacity", fixture.uuid, "1.001"}},
			{name: "NaN", arguments: []string{"set-window-opacity", fixture.uuid, "NaN"}},
			{name: "positive infinity", arguments: []string{"set-window-opacity", fixture.uuid, "+Inf"}},
			{name: "negative infinity", arguments: []string{"set-window-opacity", fixture.uuid, "-Inf"}},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				result := runKWST(t, kwst, test.arguments...)
				if result.exitCode == 0 {
					t.Fatalf("set-window-opacity accepted invalid input:\n%s", result.String())
				}
				if actual := getOpacity(t, kwst, fixture.uuid); !opacityEqual(actual, original) {
					t.Fatalf("invalid input changed opacity to %g, want %g", actual, original)
				}
			})
		}
	})

	t.Run("set changes opacity", func(t *testing.T) {
		target := 0.5
		if opacityEqual(original, target) {
			target = 0.6
		}
		setOpacityAndWait(t, kwst, fixture.uuid, target)
		actual := getOpacity(t, kwst, fixture.uuid)
		if opacityEqual(original, actual) {
			t.Fatalf("set-window-opacity did not change opacity from %g", original)
		}
	})

	t.Run("increase and decrease change opacity", func(t *testing.T) {
		setOpacityAndWait(t, kwst, fixture.uuid, 0.5)

		requireSuccess(t, runKWST(t, kwst, "increase-window-opacity", fixture.uuid), "increase window opacity")
		waitForOpacity(t, kwst, fixture.uuid, 0.55)

		requireSuccess(t, runKWST(t, kwst, "decrease-window-opacity", fixture.uuid), "decrease window opacity")
		waitForOpacity(t, kwst, fixture.uuid, 0.5)
	})

	t.Run("increase and decrease respect limits", func(t *testing.T) {
		setOpacityAndWait(t, kwst, fixture.uuid, 1.0)
		requireSuccess(t, runKWST(t, kwst, "increase-window-opacity", fixture.uuid), "increase opacity at upper limit")
		waitForOpacity(t, kwst, fixture.uuid, 1.0)

		setOpacityAndWait(t, kwst, fixture.uuid, 0.1)
		requireSuccess(t, runKWST(t, kwst, "decrease-window-opacity", fixture.uuid), "decrease opacity at lower limit")
		waitForOpacity(t, kwst, fixture.uuid, 0.1)
	})
}

func testRelativeGeometryCommands(t *testing.T, kwst string, fixture *fixtureWindow) {
	t.Helper()

	activateAndVerify(t, kwst, fixture)
	outputResult := runKWST(t, kwst, "get-active-window-output")
	outputName := requireSingleOutputName(t, outputResult, "get fixture output")
	areaResult := runKWST(t, kwst, "get-output-geometry", "--client-area", outputName)
	requireSuccess(t, areaResult, "get fixture output client area")
	clientArea, err := parseGeometry(areaResult.stdout)
	if err != nil {
		t.Fatalf("parse fixture output client area %q: %v", areaResult.stdout, err)
	}
	if clientArea.width <= 0 || clientArea.height <= 0 {
		t.Fatalf("fixture output has non-positive client area dimensions: %+v", clientArea)
	}

	original := getGeometry(t, kwst, fixture.uuid)
	t.Cleanup(func() {
		setGeometryAndWait(t, kwst, fixture.uuid, original)
	})

	baseline := relativeGeometry(clientArea, 20, 20, 60, 60)

	t.Run("set relative position", func(t *testing.T) {
		setGeometryAndWait(t, kwst, fixture.uuid, baseline)
		expected := baseline
		expected.x = relativeCoordinate(clientArea.x, clientArea.width, 10)
		expected.y = relativeCoordinate(clientArea.y, clientArea.height, 10)

		requireSuccess(t, runKWST(t, kwst, "set-window-position-relative", fixture.uuid, "10", "10"), "set relative fixture position")
		waitForGeometry(t, kwst, fixture.uuid, expected)
	})

	t.Run("set relative size", func(t *testing.T) {
		setGeometryAndWait(t, kwst, fixture.uuid, baseline)
		expected := baseline
		expected.width = relativeLength(clientArea.width, 50)
		expected.height = relativeLength(clientArea.height, 50)

		requireSuccess(t, runKWST(t, kwst, "set-window-size-relative", fixture.uuid, "50", "50"), "set relative fixture size")
		waitForGeometry(t, kwst, fixture.uuid, expected)
	})

	t.Run("set relative geometry", func(t *testing.T) {
		setGeometryAndWait(t, kwst, fixture.uuid, baseline)
		expected := relativeGeometry(clientArea, 10, 10, 50, 50)

		requireSuccess(t, runKWST(t, kwst, "set-window-geometry-relative", fixture.uuid, "10", "10", "50", "50"), "set relative fixture geometry")
		waitForGeometry(t, kwst, fixture.uuid, expected)
	})

	positionClampTests := []struct {
		name         string
		boundaryArgs []string
		clampedArgs  []string
	}{
		{
			name:         "geometry lower position limit",
			boundaryArgs: []string{"set-window-geometry-relative", fixture.uuid, "0", "0", "50", "50"},
			clampedArgs:  []string{"set-window-geometry-relative", "--", fixture.uuid, "-10", "-20", "50", "50"},
		},
		{
			name:         "geometry upper position limit",
			boundaryArgs: []string{"set-window-geometry-relative", fixture.uuid, "100", "100", "50", "50"},
			clampedArgs:  []string{"set-window-geometry-relative", fixture.uuid, "110", "120", "50", "50"},
		},
		{
			name:         "position lower limit",
			boundaryArgs: []string{"set-window-position-relative", fixture.uuid, "0", "0"},
			clampedArgs:  []string{"set-window-position-relative", "--", fixture.uuid, "-10", "-20"},
		},
		{
			name:         "position upper limit",
			boundaryArgs: []string{"set-window-position-relative", fixture.uuid, "100", "100"},
			clampedArgs:  []string{"set-window-position-relative", fixture.uuid, "110", "120"},
		},
	}
	for _, test := range positionClampTests {
		t.Run("clamp "+test.name, func(t *testing.T) {
			verifyRelativeClamp(t, kwst, fixture.uuid, baseline, test.boundaryArgs, test.clampedArgs)
		})
	}

	dimensionClampTests := []struct {
		name         string
		boundaryArgs []string
		clampedArgs  []string
	}{
		{
			name:         "geometry lower dimension limit",
			boundaryArgs: []string{"set-window-geometry-relative", fixture.uuid, "10", "10", "10", "10"},
			clampedArgs:  []string{"set-window-geometry-relative", fixture.uuid, "10", "10", "0", "9"},
		},
		{
			name:         "geometry upper dimension limit",
			boundaryArgs: []string{"set-window-geometry-relative", fixture.uuid, "10", "10", "100", "100"},
			clampedArgs:  []string{"set-window-geometry-relative", fixture.uuid, "10", "10", "110", "120"},
		},
		{
			name:         "size lower limit",
			boundaryArgs: []string{"set-window-size-relative", fixture.uuid, "10", "10"},
			clampedArgs:  []string{"set-window-size-relative", fixture.uuid, "0", "9"},
		},
		{
			name:         "size upper limit",
			boundaryArgs: []string{"set-window-size-relative", fixture.uuid, "100", "100"},
			clampedArgs:  []string{"set-window-size-relative", fixture.uuid, "110", "120"},
		},
	}
	for _, test := range dimensionClampTests {
		t.Run("clamp "+test.name, func(t *testing.T) {
			verifyRelativeClamp(t, kwst, fixture.uuid, baseline, test.boundaryArgs, test.clampedArgs)
		})
	}
}

func relativeGeometry(area geometry, xPercent, yPercent, widthPercent, heightPercent float64) geometry {
	return geometry{
		x:      relativeCoordinate(area.x, area.width, xPercent),
		y:      relativeCoordinate(area.y, area.height, yPercent),
		width:  relativeLength(area.width, widthPercent),
		height: relativeLength(area.height, heightPercent),
	}
}

func relativeCoordinate(origin, length, percent float64) float64 {
	return origin + relativeLength(length, percent)
}

func relativeLength(length, percent float64) float64 {
	return math.Round(length * percent / 100)
}

func verifyRelativeClamp(t *testing.T, kwst, uuid string, baseline geometry, boundaryArgs, clampedArgs []string) {
	t.Helper()

	setGeometryAndWait(t, kwst, uuid, baseline)
	requireSuccess(t, runKWST(t, kwst, boundaryArgs...), "set relative boundary value")
	boundary := waitForGeometryChange(t, kwst, uuid, baseline)

	setGeometryAndWait(t, kwst, uuid, baseline)
	requireSuccess(t, runKWST(t, kwst, clampedArgs...), "set out-of-range relative value")
	waitForGeometry(t, kwst, uuid, boundary)
}

func resizeAndMoveFixture(t *testing.T, kwst string, fixture *fixtureWindow) {
	t.Helper()

	original := getGeometry(t, kwst, fixture.uuid)
	target := geometry{
		x:      original.x + 20,
		y:      original.y + 20,
		width:  adjustedSize(original.width, 60, 320),
		height: adjustedSize(original.height, 40, 240),
	}

	requireSuccess(t, runKWST(t, kwst, "set-window-size", fixture.uuid,
		strconv.FormatFloat(target.width, 'f', 0, 64),
		strconv.FormatFloat(target.height, 'f', 0, 64),
	), "resize fixture")
	requireSuccess(t, runKWST(t, kwst, "set-window-position", fixture.uuid,
		strconv.FormatFloat(target.x, 'f', 0, 64),
		strconv.FormatFloat(target.y, 'f', 0, 64),
	), "move fixture")

	eventually(t, "fixture geometry to change", func() (bool, string) {
		result := runKWST(t, kwst, "get-window-geometry", fixture.uuid)
		if result.exitCode != 0 {
			return false, result.String()
		}
		actual, err := parseGeometry(result.stdout)
		if err != nil {
			return false, err.Error()
		}
		return actual == target, fmt.Sprintf("got %+v, want %+v", actual, target)
	})
}

func adjustedSize(current, delta, threshold float64) float64 {
	if current > threshold {
		return current - delta
	}
	return current + delta
}

func getGeometry(t *testing.T, kwst, uuid string) geometry {
	t.Helper()
	result := runKWST(t, kwst, "get-window-geometry", uuid)
	requireSuccess(t, result, "get fixture geometry")
	value, err := parseGeometry(result.stdout)
	if err != nil {
		t.Fatalf("parse fixture geometry %q: %v", result.stdout, err)
	}
	return value
}

func setGeometryAndWait(t *testing.T, kwst, uuid string, expected geometry) {
	t.Helper()
	requireSuccess(t, runKWST(t, kwst, "set-window-geometry", uuid,
		strconv.FormatFloat(expected.x, 'f', 0, 64),
		strconv.FormatFloat(expected.y, 'f', 0, 64),
		strconv.FormatFloat(expected.width, 'f', 0, 64),
		strconv.FormatFloat(expected.height, 'f', 0, 64),
	), "set fixture geometry")
	waitForGeometry(t, kwst, uuid, expected)
}

func waitForGeometry(t *testing.T, kwst, uuid string, expected geometry) {
	t.Helper()
	eventually(t, fmt.Sprintf("window geometry to become %+v", expected), func() (bool, string) {
		result := runKWST(t, kwst, "get-window-geometry", uuid)
		if result.exitCode != 0 {
			return false, result.String()
		}
		actual, err := parseGeometry(result.stdout)
		if err != nil {
			return false, err.Error()
		}
		return actual == expected, fmt.Sprintf("got %+v, want %+v", actual, expected)
	})
}

func waitForGeometryChange(t *testing.T, kwst, uuid string, original geometry) geometry {
	t.Helper()
	var actual geometry
	eventually(t, fmt.Sprintf("window geometry to change from %+v", original), func() (bool, string) {
		result := runKWST(t, kwst, "get-window-geometry", uuid)
		if result.exitCode != 0 {
			return false, result.String()
		}
		var err error
		actual, err = parseGeometry(result.stdout)
		if err != nil {
			return false, err.Error()
		}
		return actual != original, fmt.Sprintf("geometry is still %+v", actual)
	})
	return actual
}

func getOpacity(t *testing.T, kwst, uuid string) float64 {
	t.Helper()
	opacity, result := readOpacity(t, kwst, uuid)
	requireSuccess(t, result, "get window opacity")
	return opacity
}

func readOpacity(t *testing.T, kwst, uuid string) (float64, commandResult) {
	t.Helper()
	result := runKWST(t, kwst, "get-window-opacity", uuid)
	if result.exitCode != 0 {
		return 0, result
	}
	opacity, err := strconv.ParseFloat(result.stdout, 64)
	if err != nil {
		result.exitCode = -1
		result.stderr = fmt.Sprintf("parse window opacity %q: %v", result.stdout, err)
	}
	return opacity, result
}

func setOpacityAndWait(t *testing.T, kwst, uuid string, opacity float64) {
	t.Helper()
	value := strconv.FormatFloat(opacity, 'f', -1, 64)
	requireSuccess(t, runKWST(t, kwst, "set-window-opacity", uuid, value), "set window opacity")
	waitForOpacity(t, kwst, uuid, opacity)
}

func waitForOpacity(t *testing.T, kwst, uuid string, expected float64) {
	t.Helper()
	eventually(t, fmt.Sprintf("window opacity to become %g", expected), func() (bool, string) {
		actual, result := readOpacity(t, kwst, uuid)
		if result.exitCode != 0 {
			return false, result.String()
		}
		return opacityEqual(actual, expected), fmt.Sprintf("got %g, want %g", actual, expected)
	})
}

func opacityEqual(left, right float64) bool {
	return math.Abs(left-right) < 1e-9
}

func parseGeometry(value string) (geometry, error) {
	fields := strings.Fields(value)
	if len(fields) != 4 {
		return geometry{}, fmt.Errorf("invalid geometry %q", value)
	}
	parts := make([]float64, 4)
	for index, field := range fields {
		part, err := strconv.ParseFloat(field, 64)
		if err != nil {
			return geometry{}, fmt.Errorf("invalid geometry %q: %w", value, err)
		}
		parts[index] = math.Round(part)
	}
	return geometry{x: parts[0], y: parts[1], width: parts[2], height: parts[3]}, nil
}

func parseOutputNames(value string) ([]string, error) {
	if value == "" {
		return nil, errors.New("list-outputs returned no output names")
	}

	lines := strings.Split(value, "\n")
	names := make([]string, 0, len(lines))
	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name == "" {
			return nil, fmt.Errorf("list-outputs returned an empty output name in %q", value)
		}
		names = append(names, name)
	}
	return names, nil
}

func parseTileRows(value string) ([]tileRow, error) {
	lines := strings.Split(value, "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("list-tiles returned no tile rows: %q", value)
	}
	const expectedHeader = "OUTPUT\tPATH\tTYPE\tRELATIVE\tABSOLUTE\tWINDOWS"
	if lines[0] != expectedHeader {
		return nil, fmt.Errorf("list-tiles returned header %q, want %q", lines[0], expectedHeader)
	}

	rows := make([]tileRow, 0, len(lines)-1)
	for _, line := range lines[1:] {
		fields := strings.Split(line, "\t")
		if len(fields) != 6 {
			return nil, fmt.Errorf("list-tiles returned invalid row %q", line)
		}
		windowCount, err := strconv.Atoi(fields[5])
		if err != nil {
			return nil, fmt.Errorf("list-tiles returned invalid window count in row %q: %w", line, err)
		}
		relativeGeometry, err := parseTileGeometry(fields[3])
		if err != nil {
			return nil, fmt.Errorf("list-tiles returned invalid relative geometry in row %q: %w", line, err)
		}
		rows = append(rows, tileRow{
			output:           fields[0],
			path:             fields[1],
			tileType:         fields[2],
			relativeGeometry: relativeGeometry,
			windowCount:      windowCount,
		})
	}
	return rows, nil
}

func parseTileGeometry(value string) (geometry, error) {
	fields := strings.Fields(value)
	if len(fields) != 4 {
		return geometry{}, fmt.Errorf("invalid geometry %q", value)
	}
	parts := make([]float64, 4)
	for index, field := range fields {
		part, err := strconv.ParseFloat(field, 64)
		if err != nil {
			return geometry{}, fmt.Errorf("invalid geometry %q: %w", value, err)
		}
		parts[index] = part
	}
	return geometry{x: parts[0], y: parts[1], width: parts[2], height: parts[3]}, nil
}

func parseWindowTile(value string) (string, string, error) {
	lines := strings.Split(value, "\n")
	if len(lines) != 2 {
		return "", "", fmt.Errorf("get-window-tile returned %d lines, want 2: %q", len(lines), value)
	}
	const expectedHeader = "OUTPUT\tDESKTOP\tPATH\tRELATIVE\tABSOLUTE"
	if lines[0] != expectedHeader {
		return "", "", fmt.Errorf("get-window-tile returned header %q, want %q", lines[0], expectedHeader)
	}
	fields := strings.Split(lines[1], "\t")
	if len(fields) != 5 {
		return "", "", fmt.Errorf("get-window-tile returned invalid row %q", lines[1])
	}
	return fields[0], fields[2], nil
}

func containsWindowUUID(value, uuid string) bool {
	for _, line := range strings.Split(value, "\n") {
		fields := strings.SplitN(line, "\t", 2)
		if len(fields) > 0 && fields[0] == uuid {
			return true
		}
	}
	return false
}

func requireSingleOutputName(t *testing.T, result commandResult, action string) string {
	t.Helper()
	requireSuccess(t, result, action)

	names, err := parseOutputNames(result.stdout)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 {
		t.Fatalf("%s returned %d output names, want 1: %q", action, len(names), result.stdout)
	}
	return names[0]
}

func getWorkspace(t *testing.T, kwst string) int {
	t.Helper()
	workspace, result := readWorkspace(t, kwst)
	requireSuccess(t, result, "get active workspace")
	return workspace
}

func readWorkspace(t *testing.T, kwst string) (int, commandResult) {
	t.Helper()
	result := runKWST(t, kwst, "get-workspace")
	if result.exitCode != 0 {
		return 0, result
	}
	workspace, err := strconv.Atoi(result.stdout)
	if err != nil {
		result.exitCode = -1
		result.stderr = fmt.Sprintf("parse workspace %q: %v", result.stdout, err)
	}
	return workspace, result
}

func windowUUID(listOutput, title string) string {
	for _, line := range strings.Split(listOutput, "\n") {
		fields := strings.SplitN(line, "\t", 4)
		if len(fields) == 4 && strings.Contains(fields[3], title) {
			return fields[0]
		}
	}
	return ""
}

func runKWST(t *testing.T, kwst string, arguments ...string) commandResult {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, kwst, arguments...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := commandResult{
		stdout: strings.TrimSpace(stdout.String()),
		stderr: strings.TrimSpace(stderr.String()),
	}
	if err == nil {
		return result
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatalf("kwst %s exceeded %s", strings.Join(arguments, " "), commandTimeout)
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		result.exitCode = exitError.ExitCode()
		return result
	}
	t.Fatalf("run kwst %s: %v", strings.Join(arguments, " "), err)
	return commandResult{}
}

func requireSuccess(t *testing.T, result commandResult, action string) {
	t.Helper()
	if result.exitCode != 0 {
		t.Fatalf("%s failed:\n%s", action, result.String())
	}
}

func eventually(t *testing.T, description string, check func() (bool, string)) {
	t.Helper()

	deadline := time.Now().Add(stateTimeout)
	lastState := ""
	for time.Now().Before(deadline) {
		if ok, state := check(); ok {
			return
		} else {
			lastState = state
		}
		time.Sleep(pollInterval)
	}
	t.Fatalf("timed out waiting for %s; last state:\n%s", description, lastState)
}
