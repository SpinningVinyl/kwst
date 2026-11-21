// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package integration

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestKWinFindSearchFields(t *testing.T) {
	kdialog := requireIntegrationEnvironment(t)
	kwst := buildKWST(t)
	fixture := startFixtureWindow(t, kdialog, fmt.Sprintf("kwst-find-%d-%d", os.Getpid(), time.Now().UnixNano()))
	eventually(t, "find fixture to appear", func() (bool, string) {
		result := runKWST(t, kwst, "list", "--show-captions")
		fixture.uuid = windowUUID(result.stdout, fixture.title)
		return result.exitCode == 0 && fixture.uuid != "", result.String()
	})
	checkFind := func(t *testing.T, term, field string, want int) {
		t.Helper()
		args := []string{"find", term}
		if field != "" {
			args = append(args, "--search-field="+field)
		}
		result := runKWST(t, kwst, args...)
		requireSuccess(t, result, "find fixture")
		count := 0
		for _, uuid := range strings.Split(result.stdout, "\n") {
			if uuid == fixture.uuid {
				count++
			}
		}
		if count != want {
			t.Fatalf("find fixture count = %d, want %d:\n%s", count, want, result.String())
		}
	}
	for _, field := range []string{"resourceClass", "resourceName", "caption"} {
		t.Run(field, func(t *testing.T) {
			value := fixture.title
			if field != "caption" {
				result := runKWST(t, kwst, "get-window-property", "--property="+field, fixture.uuid)
				requireSuccess(t, result, "read fixture "+field)
				value = result.stdout
			}
			term := "^" + regexp.QuoteMeta(strings.ToUpper(value)) + "$"
			checkFind(t, term, "", 1)
			checkFind(t, term, field, 1)
		})
	}
	t.Run("explicit field excludes caption-only match", func(t *testing.T) {
		term := "^" + regexp.QuoteMeta(fixture.title) + "$"
		checkFind(t, term, "resourceClass", 0)
		checkFind(t, term, "resourceName", 0)
	})
	t.Run("multiple matching fields emit one UUID", func(t *testing.T) {
		checkFind(t, ".*", "", 1)
	})
	t.Run("invalid regex", func(t *testing.T) {
		result := runKWST(t, kwst, "find", "[")
		if result.exitCode != 1 || result.stdout != "" || !strings.Contains(result.stderr, "Invalid regular expression") {
			t.Fatalf("invalid regex should return only an error:\n%s", result.String())
		}
	})
}

func TestKWinFindFilters(t *testing.T) {
	kdialog := requireIntegrationEnvironment(t)
	kwst := buildKWST(t)
	prefix := fmt.Sprintf("kwst-find-filters-%d-%d", os.Getpid(), time.Now().UnixNano())
	first := startFixtureWindow(t, kdialog, prefix+"-one")
	second := startFixtureWindow(t, kdialog, prefix+"-two")
	waitForFixtureWindows(t, kwst, first, second)
	originalWorkspace := getWorkspace(t, kwst)
	targetWorkspace := createWorkspace(t, kwst, prefix)
	t.Cleanup(func() {
		for _, fixture := range []*fixtureWindow{first, second} {
			if result := runKWST(t, kwst, "set-window-workspace", fixture.uuid, strconv.Itoa(originalWorkspace)); result.exitCode != 0 {
				t.Errorf("restore fixture workspace:\n%s", result.String())
			}
		}
		if result := removeWorkspace(t, kwst, prefix); result.exitCode != 0 {
			t.Errorf("remove find test workspace:\n%s", result.String())
		}
	})
	requireSuccess(t, runKWST(t, kwst, "set-window-workspace", first.uuid, strconv.Itoa(targetWorkspace)), "move first fixture to temporary workspace")
	requireSuccess(t, runKWST(t, kwst, "set-window-workspace", second.uuid, strconv.Itoa(originalWorkspace)), "move second fixture to original workspace")

	checkFind := func(t *testing.T, wantFirst, wantSecond bool, flags ...string) {
		t.Helper()
		eventually(t, "find to return the expected fixtures", func() (bool, string) {
			result := runKWST(t, kwst, append([]string{"find", "^" + regexp.QuoteMeta(prefix) + "-"}, flags...)...)
			wantCount := 0
			if wantFirst {
				wantCount++
			}
			if wantSecond {
				wantCount++
			}
			return result.exitCode == 0 && len(strings.Fields(result.stdout)) == wantCount &&
				containsWindowUUID(result.stdout, first.uuid) == wantFirst &&
				containsWindowUUID(result.stdout, second.uuid) == wantSecond, result.String()
		})
	}
	t.Run("workspace filter", func(t *testing.T) {
		checkFind(t, true, true)
		checkFind(t, true, false, "--workspace="+strconv.Itoa(targetWorkspace))
		checkFind(t, false, true, "--workspace="+strconv.Itoa(originalWorkspace))
	})

	outputScript := writeCustomScript(t, `const window = workspace.windowList().find(w => w.internalId == {{jsString .P1}});
returnResult(window.output.name);`)
	fixtureOutputs := make([]string, 2)
	for i, fixture := range []*fixtureWindow{first, second} {
		result := runKWST(t, kwst, "run-custom-script", "--parameter-1="+fixture.uuid, outputScript)
		fixtureOutputs[i] = requireSingleOutputName(t, result, "read fixture output")
	}
	result := runKWST(t, kwst, "list-outputs")
	requireSuccess(t, result, "list outputs")
	outputs, err := parseOutputNames(result.stdout)
	if err != nil {
		t.Fatal(err)
	}
	for _, output := range outputs {
		t.Run("output filter "+output, func(t *testing.T) {
			checkFind(t, fixtureOutputs[0] == output, fixtureOutputs[1] == output, "--output="+strings.ToLower(output))
			checkFind(t, fixtureOutputs[0] == output, false, "--output="+output, "--workspace="+strconv.Itoa(targetWorkspace))
		})
	}
	t.Run("property and combined filters", func(t *testing.T) {
		requireSuccess(t, runKWST(t, kwst, "unminimize-window", second.uuid), "restore second fixture")
		requireSuccess(t, runKWST(t, kwst, "minimize-window", first.uuid), "minimize first fixture")
		checkFind(t, true, false, "--property=minimized:true")
		checkFind(t, false, true, "--property=minimized:false")
		checkFind(t, true, false, "--workspace="+strconv.Itoa(targetWorkspace), "--output="+fixtureOutputs[0], "--property=minimized:true", "--search-field=caption")
		checkFind(t, false, false, "--workspace="+strconv.Itoa(targetWorkspace), "--output="+fixtureOutputs[0], "--property=minimized:false")
		requireSuccess(t, runKWST(t, kwst, "unminimize-window", first.uuid), "restore first fixture")
		checkFind(t, false, false, "--property=minimized:true")
		checkFind(t, true, true, "--property=minimized:false")
	})
	t.Run("pinned windows", func(t *testing.T) {
		script := writeCustomScript(t, `const window = workspace.windowList().find(w => w.internalId == {{jsString .P1}});
window.desktops = [];`)
		requireSuccess(t, runKWST(t, kwst, "run-custom-script", "--parameter-1="+second.uuid, script), "pin second fixture")
		checkFind(t, true, true, "--workspace="+strconv.Itoa(targetWorkspace))
		checkFind(t, false, true, "--workspace="+strconv.Itoa(originalWorkspace))
	})
	for _, test := range []struct{ flag, message string }{
		{"--workspace=-1", "Workspace not found: -1"},
		{"--output=" + prefix, "Output not found: " + prefix},
	} {
		t.Run("invalid "+test.flag, func(t *testing.T) {
			result := runKWST(t, kwst, "find", "^no-matching-window-"+prefix, test.flag)
			if result.exitCode != 1 || result.stdout != "" || !strings.Contains(result.stderr, test.message) {
				t.Fatalf("invalid filter should return only the standard error even with no regex matches:\n%s", result.String())
			}
		})
	}
}
