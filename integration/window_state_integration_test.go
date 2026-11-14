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

func TestKWinWindowState(t *testing.T) {
	kdialog := requireIntegrationEnvironment(t)
	kwst := buildKWST(t)
	fixture := startFixtureWindow(t, kdialog, fmt.Sprintf("kwst-state-%d-%d", os.Getpid(), time.Now().UnixNano()))
	eventually(t, "window state fixture to appear", func() (bool, string) {
		result := runKWST(t, kwst, "list", "--show-captions")
		fixture.uuid = windowUUID(result.stdout, fixture.title)
		return result.exitCode == 0 && fixture.uuid != "", result.String()
	})

	t.Run("minimize and restore", func(t *testing.T) {
		for _, command := range []string{"minimize-window", "unminimize-window"} {
			for range 2 {
				requireSuccess(t, runKWST(t, kwst, command, fixture.uuid), command)
				eventually(t, command+" to update minimized state", func() (bool, string) {
					result := runKWST(t, kwst, "get-window-property", "--property=minimized", fixture.uuid)
					return result.exitCode == 0 && result.stdout == strconv.FormatBool(command == "minimize-window"), result.String()
				})
			}
		}
	})

	t.Run("minimized property", func(t *testing.T) {
		for _, step := range []struct{ value, want string }{
			{"true", "true"}, {"toggle", "false"}, {"toggle", "true"}, {"false", "false"},
		} {
			requireSuccess(t, runKWST(t, kwst, "set-window-property", "--property=minimized", "--value="+step.value, fixture.uuid), "set minimized property")
			eventually(t, "minimized property to become "+step.want, func() (bool, string) {
				result := runKWST(t, kwst, "get-window-property", "--property=minimized", fixture.uuid)
				return result.exitCode == 0 && result.stdout == step.want, result.String()
			})
		}
	})

	stateScript := writeCustomScript(t, `const window = workspace.windowList().find(w => w.internalId == {{jsString .P1}});
returnResult(window.maximizeMode);`)
	setupScript := writeCustomScript(t, `const window = workspace.windowList().find(w => w.internalId == {{jsString .P1}});
const mode = Number({{jsString .P2}});
window.setMaximize((mode & 1) !== 0, (mode & 2) !== 0);`)
	waitForMode := func(t *testing.T, mode int) {
		t.Helper()
		eventually(t, fmt.Sprintf("maximize mode to become %d", mode), func() (bool, string) {
			result := runKWST(t, kwst, "run-custom-script", "--parameter-1="+fixture.uuid, stateScript)
			return result.exitCode == 0 && result.stdout == strconv.Itoa(mode), result.String()
		})
	}

	for _, command := range []string{"maximize-window", "unmaximize-window"} {
		for _, direction := range []struct {
			name string
			mask int
		}{
			{"default", 3}, {"vertical", 1}, {"horizontal", 2}, {"both", 3},
		} {
			for initial := range 4 {
				t.Run(fmt.Sprintf("%s/%s/from-%d", command, direction.name, initial), func(t *testing.T) {
					requireSuccess(t, runKWST(t, kwst, "run-custom-script", "--parameter-1="+fixture.uuid, "--parameter-2="+strconv.Itoa(initial), setupScript), "set initial maximize mode")
					waitForMode(t, initial)
					args := []string{command, fixture.uuid}
					if direction.name != "default" {
						args = append(args, "--direction="+direction.name)
					}
					want := initial | direction.mask
					if command == "unmaximize-window" {
						want = initial &^ direction.mask
					}
					requireSuccess(t, runKWST(t, kwst, args...), command)
					waitForMode(t, want)
				})
			}
		}
	}

	t.Run("missing windows", func(t *testing.T) {
		missingUUID := fixture.title + "-missing"
		for _, command := range []string{"minimize-window", "unminimize-window", "maximize-window", "unmaximize-window"} {
			t.Run(command, func(t *testing.T) {
				result := runKWST(t, kwst, command, missingUUID)
				if result.exitCode != 1 || result.stdout != "" || !strings.Contains(result.stderr, "Window not found: "+missingUUID) {
					t.Fatalf("missing window should return only the standard error:\n%s", result.String())
				}
			})
		}
	})
}
