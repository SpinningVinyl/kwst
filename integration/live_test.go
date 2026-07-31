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
	x      int
	y      int
	width  int
	height int
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
			{name: "set-window-workspace", arguments: []string{"set-window-workspace", missingUUID, "1"}},
			{name: "set-window-property", arguments: []string{"set-window-property", "--property=keepAbove", "--value=true", missingUUID}},
			{name: "close-window", arguments: []string{"close-window", missingUUID}},
			{name: "get-window-opacity", arguments: []string{"get-window-opacity", missingUUID}},
			{name: "set-window-opacity", arguments: []string{"set-window-opacity", missingUUID, "0.5"}},
			{name: "increase-window-opacity", arguments: []string{"increase-window-opacity", missingUUID}},
			{name: "decrease-window-opacity", arguments: []string{"decrease-window-opacity", missingUUID}},
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

	testOutputCommands(t, kwst)
	t.Run("window opacity commands", func(t *testing.T) {
		testWindowOpacityCommands(t, kwst, first)
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

func testOutputCommands(t *testing.T, kwst string) {
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

func resizeAndMoveFixture(t *testing.T, kwst string, fixture *fixtureWindow) {
	t.Helper()

	original := getGeometry(t, kwst, fixture.uuid)
	target := geometry{
		x:      original.x + 20,
		y:      original.y + 20,
		width:  adjustedSize(original.width, 60, 320),
		height: adjustedSize(original.height, 40, 240),
	}

	requireSuccess(t, runKWST(t, kwst, "set-window-size", fixture.uuid, strconv.Itoa(target.width), strconv.Itoa(target.height)), "resize fixture")
	requireSuccess(t, runKWST(t, kwst, "set-window-position", fixture.uuid, strconv.Itoa(target.x), strconv.Itoa(target.y)), "move fixture")

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

func adjustedSize(current, delta, threshold int) int {
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
	parts := make([]int, 4)
	for index, field := range fields {
		part, err := strconv.Atoi(field)
		if err != nil {
			return geometry{}, fmt.Errorf("invalid geometry %q: %w", value, err)
		}
		parts[index] = part
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
