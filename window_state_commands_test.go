// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
)

func TestWindowStateCommands(t *testing.T) {
	const uuid = `window"with\characters`
	for _, command := range []string{"minimize-window", "unminimize-window", "maximize-window", "unmaximize-window"} {
		directions := []string{""}
		maximize := command == "maximize-window" || command == "unmaximize-window"
		if maximize {
			directions = append(directions, "vertical", "horizontal", "both")
		}
		for _, direction := range directions {
			t.Run(command+"/"+direction, func(t *testing.T) {
				var cli CLI
				parser, err := kong.New(&cli, kong.Writers(io.Discard, io.Discard))
				if err != nil {
					t.Fatal(err)
				}
				args := []string{command, uuid}
				if direction != "" {
					args = append(args, "--direction="+direction)
				}
				ctx, err := parser.Parse(args)
				if err != nil {
					t.Fatal(err)
				}
				sp := ScriptPackage{ScriptTemplate: JS_HEADER}
				if err := ctx.Run(&sp); err != nil {
					t.Fatal(err)
				}
				want := ScriptParams{Uuid: uuid}
				wantTemplate := JS_SET_WINDOW_PROPERTY
				var snippets []string
				if maximize {
					if direction == "" {
						direction = "both"
					}
					want.Maximize = command == "maximize-window"
					want.Direction = direction
					wantTemplate = JS_MAXIMIZE_WINDOW
					snippets = []string{"const maximize = " + strconv.FormatBool(want.Maximize) + ";", `const direction = "` + direction + `";`}
				} else {
					want.WindowProperty = "minimized"
					want.PropertyValue = strconv.FormatBool(command == "minimize-window")
					snippets = []string{"targetWindow.minimized = " + want.PropertyValue + ";"}
				}
				if sp.Params != want || sp.ScriptTemplate != JS_HEADER+wantTemplate {
					t.Fatalf("command generated Params=%+v and unexpected template=%t; want %+v", sp.Params, sp.ScriptTemplate != JS_HEADER+wantTemplate, want)
				}
				var script strings.Builder
				if err := prepareScript(&script, sp); err != nil {
					t.Fatal(err)
				}
				quotedUUID, err := jsString(uuid)
				if err != nil {
					t.Fatal(err)
				}
				snippets = append(snippets, "findWindow("+quotedUUID+")")
				for _, snippet := range snippets {
					if !strings.Contains(script.String(), snippet) {
						t.Errorf("generated script is missing %q", snippet)
					}
				}
			})
		}
		t.Run(command+"/invalid arguments", func(t *testing.T) {
			invalid := [][]string{{command}}
			if maximize {
				invalid = append(invalid, []string{command, uuid, "--direction=diagonal"})
			}
			for _, args := range invalid {
				var cli CLI
				parser, err := kong.New(&cli, kong.Writers(io.Discard, io.Discard))
				if err != nil {
					t.Fatal(err)
				}
				if _, err := parser.Parse(args); err == nil {
					t.Errorf("Parse(%q) accepted invalid arguments", args)
				}
			}
		})
	}
}
