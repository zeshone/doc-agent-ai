package docagent

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"
)

// tmplVarRegex matches {{UPPER_CASE}} template variables (same sematics as
// the v2.0.0 generate.js regex /{{([A-Z0-9_]+)}}/g).
var tmplVarRegex = regexp.MustCompile(`\{\{([A-Z0-9_]+)\}\}`)

// yamlList formats a slice of strings as YAML list items (two-space indent).
// Mirrors the v2.0.0 generate.js yamlList() helper.
func yamlList(items []string) string {
	if len(items) == 0 {
		return ""
	}
	lines := make([]string, len(items))
	for i, item := range items {
		lines[i] = "  - " + item
	}
	return strings.Join(lines, "\n")
}

// boolText converts a boolean to "true" / "false" (JavaScript-style).
// Mirrors the v2.0.0 generate.js boolText() helper.
func boolText(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

// funcMap registers template helpers available inside rendered templates.
var funcMap = template.FuncMap{
	"yamlList": yamlList,
	"boolText": boolText,
}

// renderTemplate converts a raw v2.0.0 template (using {{KEY}} syntax) into
// a Go text/template-compatible form, parses it, and executes it with the
// given variable map. Missing variables produce a descriptive error.
func renderTemplate(raw string, vars map[string]string) (string, error) {
	// Convert {{KEY}} to {{.KEY}} so text/template can resolve variables
	// from the map (e.g. {{.NAME}} → map["NAME"]).
	goTmpl := tmplVarRegex.ReplaceAllString(raw, "{{.$1}}")

	tmpl, err := template.New("t").
		Funcs(funcMap).
		Option("missingkey=error").
		Parse(goTmpl)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}
