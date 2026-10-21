// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package integration

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func testListCommand(t *testing.T, kwst string, first, second *fixtureWindow, outputs []string) {
	t.Helper()
	originalWorkspace := getWorkspace(t, kwst)
	name := fmt.Sprintf("kwst-list-%d-%d", os.Getpid(), time.Now().UnixNano())
	targetWorkspace := createWorkspace(t, kwst, name)
	t.Cleanup(func() {
		for _, fixture := range []*fixtureWindow{first, second} {
			result := runKWST(t, kwst, "set-window-workspace", fixture.uuid, strconv.Itoa(originalWorkspace))
			if result.exitCode != 0 {
				t.Errorf("restore fixture workspace:\n%s", result.String())
			}
		}
		if result := removeWorkspace(t, kwst, name); result.exitCode != 0 {
			t.Errorf("remove list test workspace:\n%s", result.String())
		}
	})
	requireSuccess(t, runKWST(t, kwst, "set-window-workspace", first.uuid, strconv.Itoa(targetWorkspace)), "move first fixture to temporary workspace")
	requireSuccess(t, runKWST(t, kwst, "set-window-workspace", second.uuid, strconv.Itoa(originalWorkspace)), "move second fixture to original workspace")

	checkList := func(t *testing.T, wantFirst, wantSecond bool, flags ...string) {
		t.Helper()
		eventually(t, "list to return the expected fixtures", func() (bool, string) {
			result := runKWST(t, kwst, append([]string{"list", "--id-only"}, flags...)...)
			if result.exitCode != 0 {
				return false, result.String()
			}
			for _, line := range strings.Split(result.stdout, "\n") {
				if strings.ContainsAny(line, "\t ") {
					return false, "UUID-only output contains extra fields: " + result.stdout
				}
			}
			return containsWindowUUID(result.stdout, first.uuid) == wantFirst &&
				containsWindowUUID(result.stdout, second.uuid) == wantSecond, result.String()
		})
	}

	t.Run("UUIDs only", func(t *testing.T) {
		checkList(t, true, true)
		checkList(t, true, true, "--show-captions", "--show-pids")
	})
	t.Run("workspace filter", func(t *testing.T) {
		checkList(t, true, false, "--workspace="+strconv.Itoa(targetWorkspace))
		checkList(t, false, true, "--workspace="+strconv.Itoa(originalWorkspace))
	})

	outputScript := writeCustomScript(t, `const window = workspace.windowList().find(w => w.internalId == {{jsString .P1}});
returnResult(window.output.name);`)
	fixtureOutputs := make([]string, 2)
	for i, fixture := range []*fixtureWindow{first, second} {
		result := runKWST(t, kwst, "run-custom-script", "--parameter-1="+fixture.uuid, outputScript)
		fixtureOutputs[i] = requireSingleOutputName(t, result, "read fixture output")
	}
	for _, output := range outputs {
		t.Run("output filter "+output, func(t *testing.T) {
			checkList(t, fixtureOutputs[0] == output, fixtureOutputs[1] == output, "--output="+output)
			checkList(t, fixtureOutputs[0] == output, false, "--output="+output, "--workspace="+strconv.Itoa(targetWorkspace))
		})
	}

	t.Run("pinned windows", func(t *testing.T) {
		script := writeCustomScript(t, `const window = workspace.windowList().find(w => w.internalId == {{jsString .P1}});
window.desktops = [];`)
		requireSuccess(t, runKWST(t, kwst, "run-custom-script", "--parameter-1="+second.uuid, script), "pin second fixture to all workspaces")
		checkList(t, true, true, "--workspace="+strconv.Itoa(targetWorkspace))
		checkList(t, false, true, "--workspace="+strconv.Itoa(originalWorkspace))
	})

	for _, test := range []struct {
		flag string
		err  string
	}{
		{"--output=" + name, "Output not found: " + name},
		{"--workspace=-1", "Workspace not found: -1"},
	} {
		t.Run("invalid "+test.flag, func(t *testing.T) {
			result := runKWST(t, kwst, "list", test.flag)
			if result.exitCode != 1 || result.stdout != "" || !strings.Contains(result.stderr, test.err) {
				t.Fatalf("invalid list filter should return only an error:\n%s", result.String())
			}
		})
	}
}
