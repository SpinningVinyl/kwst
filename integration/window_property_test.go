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

func TestKWinWindowProperties(t *testing.T) {
	kdialog := requireIntegrationEnvironment(t)
	kwst := buildKWST(t)
	fixture := startFixtureWindow(t, kdialog, fmt.Sprintf("kwst-properties-%d-%d", os.Getpid(), time.Now().UnixNano()))
	eventually(t, "KDialog fixture to appear", func() (bool, string) {
		result := runKWST(t, kwst, "list", "--show-captions")
		fixture.uuid = windowUUID(result.stdout, fixture.title)
		return result.exitCode == 0 && fixture.uuid != "", result.String()
	})

	getProperty := func(t *testing.T, property string) string {
		t.Helper()
		result := runKWST(t, kwst, "get-window-property", "--property="+property, fixture.uuid)
		requireSuccess(t, result, "get property "+property)
		return result.stdout
	}
	getAll := func(t *testing.T) map[string]string {
		t.Helper()
		values := map[string]string{}
		for _, row := range strings.Split(getProperty(t, "all"), "\n") {
			name, value, ok := strings.Cut(row, "\t")
			if !ok || strings.Contains(value, "\t") {
				t.Fatalf("invalid property TSV row: %q", row)
			}
			if _, duplicate := values[name]; duplicate {
				t.Fatalf("duplicate property: %q", name)
			}
			values[name] = value
		}
		return values
	}

	t.Run("all supported properties", func(t *testing.T) {
		properties := strings.Fields(`active demandsAttention dialog fullScreen fullScreenable
			hidden keepAbove keepBelow layer maximizable minimizable minimized modal noBorder
			normalWindow onAllDesktops onScreenDisplay popupMenu popupWindow resizeable
			resourceClass resourceName skipPager skipSwitcher skipTaskbar specialWindow splash
			transient utility windowRole windowType`)
		values := getAll(t)
		if len(values) != len(properties) {
			t.Fatalf("got %d properties, want %d", len(values), len(properties))
		}
		for _, property := range properties {
			value, ok := values[property]
			if !ok {
				t.Fatalf("all output is missing %q", property)
			}
			switch property {
			case "layer", "windowType":
				if value == "" {
					t.Errorf("%s has an empty enum name", property)
				} else if _, err := strconv.Atoi(value); err == nil {
					t.Errorf("%s returned an integer instead of a name: %q", property, value)
				}
				if single := getProperty(t, property); single != value {
					t.Errorf("%s = %q, all returned %q", property, single, value)
				}
			case "resourceClass", "resourceName", "windowRole":
				if single := getProperty(t, property); single != value {
					t.Errorf("%s = %q, all returned %q", property, single, value)
				}
			default:
				if value != "true" && value != "false" {
					t.Errorf("%s = %q, want true or false", property, value)
				}
			}
		}
		if !(values["normalWindow"] == "true" && values["windowType"] == "Normal") &&
			!(values["dialog"] == "true" && values["windowType"] == "Dialog") {
			t.Errorf("windowType=%q does not match normalWindow=%q or dialog=%q", values["windowType"], values["normalWindow"], values["dialog"])
		}
	})

	for _, property := range []string{"keepAbove", "keepBelow", "skipPager", "skipSwitcher", "skipTaskbar"} {
		t.Run("change "+property, func(t *testing.T) {
			original := getProperty(t, property)
			if original != "true" && original != "false" {
				t.Fatalf("%s = %q, want a boolean", property, original)
			}
			setAndVerify := func(value string) {
				t.Helper()
				requireSuccess(t, runKWST(t, kwst, "set-window-property", "--property="+property, "--value="+value, fixture.uuid), "set property "+property)
				eventually(t, property+" to become "+value, func() (bool, string) {
					result := runKWST(t, kwst, "get-window-property", "--property="+property, fixture.uuid)
					return result.exitCode == 0 && result.stdout == value, result.String()
				})
				if allValue := getAll(t)[property]; allValue != value {
					t.Errorf("all returned %s=%q, want %q", property, allValue, value)
				}
			}
			t.Cleanup(func() { setAndVerify(original) })
			setAndVerify(strconv.FormatBool(original != "true"))
			if property == "keepAbove" || property == "keepBelow" {
				expectedLayer := "NormalLayer"
				if original == "false" {
					expectedLayer = "AboveLayer"
					if property == "keepBelow" {
						expectedLayer = "BelowLayer"
					}
				}
				if layer := getProperty(t, "layer"); layer != expectedLayer {
					t.Errorf("layer = %q, want %q", layer, expectedLayer)
				}
			}
		})
	}

	t.Run("missing window", func(t *testing.T) {
		const missingUUID = "kwst-missing-property-window"
		for _, property := range []string{"keepAbove", "all"} {
			result := runKWST(t, kwst, "get-window-property", "--property="+property, missingUUID)
			if result.exitCode != 1 || !strings.Contains(result.stderr, "Window not found: "+missingUUID) {
				t.Errorf("missing window with property=%s:\n%s", property, result.String())
			}
		}
	})
}
