package main

import (
	"encoding/json"
	"fmt"
	"io"
	"text/template"
)

// parameters that are passed to the script template
type ScriptParams struct {
	DbusAddr              string
	Debug                 bool
	ScriptName            string
	SearchTerm            string
	SearchField           string
	IncludeSpecialWindows bool
	ShowCaptions          bool
	ShowPids              bool
	Uuid                  string
	WorkspaceId           int
	X                     int
	Y                     int
	Width                 int
	Height                int
	WindowProperty        string
	PropertyValue         string
	P1                    string
	P2                    string
	P3                    string
	P4                    string
	P5                    string
	P6                    string
	ClientArea            bool
	OutputName            string
	Opacity               float64
	Delta                 float64
	RelativeX             float64
	RelativeY             float64
	RelativeWidth         float64
	RelativeHeight        float64
	LeavesOnly            bool
	TilePath              string
	Edge                  string
	Direction             string
}

type ScriptPackage struct {
	ScriptTemplate string
	Params         ScriptParams
	Custom         bool
}

// jsString returns value as a quoted JavaScript string literal. JSON string
// literals are also valid JavaScript string literals and safely escape values
// that would otherwise alter the generated script.
func jsString(value string) (string, error) {
	quoted, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(quoted), nil
}

func prepareScript(w io.Writer, sp ScriptPackage) error {
	if !sp.Custom {
		sp.ScriptTemplate += JS_FOOTER
	}
	tmpl, err := template.New("kwin_script").
		Funcs(template.FuncMap{
			"jsString": jsString,
		}).
		Parse(sp.ScriptTemplate)
	if err != nil {
		return fmt.Errorf("Error parsing script template: %w", err)
	}
	if err := tmpl.Execute(w, sp.Params); err != nil {
		return fmt.Errorf("Error executing script template: %w", err)
	}
	return nil
}
