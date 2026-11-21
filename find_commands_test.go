// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
)

func TestFindPropertyValidation(t *testing.T) {
	properties := strings.Fields(`active demandsAttention dialog fullScreen fullScreenable
		hidden keepAbove keepBelow maximizable minimizable minimized modal noBorder
		normalWindow onAllDesktops onScreenDisplay popupMenu popupWindow resizeable
		skipPager skipSwitcher skipTaskbar specialWindow splash transient utility`)
	for _, property := range properties {
		for _, value := range []string{"true", "false"} {
			t.Run(property+":"+value, func(t *testing.T) {
				cmd := FindCmd{Property: property + ":" + value}
				if err := cmd.Validate(); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
	for _, property := range []string{
		"minimized", "minimized:", ":true", "minimized:true:false",
		"minimized:True", "minimized:FALSE", "minimized:1", "minimized:0",
		"minimized:toggle", "minimized: true", " minimized:true",
		"unknown:true", "all:true", "layer:true", "windowType:false",
		"resourceClass:true", "resourceName:false", "windowRole:true",
		"__proto__:true", "minimized:true;throw new Error()",
	} {
		t.Run("invalid/"+property, func(t *testing.T) {
			cmd := FindCmd{Property: property}
			if err := cmd.Validate(); err == nil {
				t.Fatalf("Validate accepted %q", property)
			}
		})
	}
}

func TestFindFilterArguments(t *testing.T) {
	for _, test := range []struct {
		name    string
		args    []string
		want    ScriptParams
		wantErr bool
	}{
		{
			name: "defaults",
			args: []string{"find", "konsole"},
			want: ScriptParams{SearchTerm: "konsole"},
		},
		{
			name: "combined filters",
			args: []string{"find", "--search-field=caption", "konsole", "--workspace=2", "--output=DP-1", "--property=minimized:false"},
			want: ScriptParams{SearchField: "caption", SearchTerm: "konsole", WorkspaceId: 2, OutputName: "DP-1", WindowProperty: "minimized", PropertyValue: "false"},
		},
		{
			name: "quoted output",
			args: []string{"find", "konsole", `--output=DP-"1\`},
			want: ScriptParams{SearchTerm: "konsole", OutputName: `DP-"1\`},
		},
		{
			name: "short flags",
			args: []string{"find", "konsole", "-w", "2", "-o", "DP-1", "-p", "keepAbove:true"},
			want: ScriptParams{SearchTerm: "konsole", WorkspaceId: 2, OutputName: "DP-1", WindowProperty: "keepAbove", PropertyValue: "true"},
		},
		{name: "invalid value", args: []string{"find", "konsole", "--property=minimized:toggle"}, wantErr: true},
		{name: "unsupported property", args: []string{"find", "konsole", "--property=resourceClass:true"}, wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var cli CLI
			parser, err := kong.New(&cli, kong.Writers(io.Discard, io.Discard))
			if err != nil {
				t.Fatal(err)
			}
			ctx, err := parser.Parse(test.args)
			if test.wantErr {
				if err == nil {
					t.Fatal("Parse accepted an invalid property filter")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var sp ScriptPackage
			if err := ctx.Run(&sp); err != nil {
				t.Fatal(err)
			}
			if sp.Params != test.want {
				t.Errorf("Params = %+v, want %+v", sp.Params, test.want)
			}
			var script strings.Builder
			if err := prepareScript(&script, sp); err != nil {
				t.Fatal(err)
			}
			output, err := jsString(test.want.OutputName)
			if err != nil {
				t.Fatal(err)
			}
			for _, snippet := range []string{
				fmt.Sprintf("filterWindows(workspace.windowList(), %s, %d)", output, test.want.WorkspaceId),
				fmt.Sprintf("const property = %q;", test.want.WindowProperty),
				fmt.Sprintf(`const propertyValue = %q === "true";`, test.want.PropertyValue),
			} {
				if !strings.Contains(script.String(), snippet) {
					t.Errorf("generated script is missing %q", snippet)
				}
			}
		})
	}
}

func TestFindSearchFields(t *testing.T) {
	for _, field := range []string{"", "resourceClass", "resourceName", "caption", "invalid"} {
		t.Run("field="+field, func(t *testing.T) {
			var cli CLI
			parser, err := kong.New(&cli, kong.Writers(io.Discard, io.Discard))
			if err != nil {
				t.Fatal(err)
			}
			args := []string{"find", "konsole"}
			if field != "" {
				args = append(args, "--search-field="+field)
			}
			ctx, err := parser.Parse(args)
			if field == "invalid" {
				if err == nil {
					t.Fatal("Parse accepted an invalid search field")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			sp := ScriptPackage{ScriptTemplate: JS_HEADER}
			if err := ctx.Run(&sp); err != nil {
				t.Fatal(err)
			}
			var script strings.Builder
			if err := prepareScript(&script, sp); err != nil {
				t.Fatal(err)
			}
			want := `const searchFields = ["resourceClass", "resourceName", "caption"];`
			if field != "" {
				want = `const searchFields = ["` + field + `"];`
			}
			if !strings.Contains(script.String(), want) {
				t.Errorf("generated script is missing %q", want)
			}
		})
	}
}
