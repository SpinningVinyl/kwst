package main

import (
	"strings"
	"testing"
	"text/template"
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
