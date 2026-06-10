package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Engine purity guard (T2a-4)
// ---------------------------------------------------------------------------

// engineFiles lists all Go source files that belong to the "engine" layer.
// These files MUST NOT import any charmbracelet/* package. The TUI files
// (tui_*.go, added in slice 2b) are the only files allowed to import charm.
var engineFiles = []string{
	"install.go",
	"platform.go",
	"generate.go",
	"config.go",
	"plan.go",
	"resolve.go",
	"uninstall.go",
	"manifest.go",
	"template.go",
	"reporter.go",
	"headless.go",
	"execute_install.go",
}

// TestEnginePurity_NoCharmImports asserts that none of the engine-layer Go
// source files import any charmbracelet/* package. This prevents architectural
// erosion and ensures the engine stays charm-free so it can be unit-tested
// without a TTY and consumed by non-TUI headless paths.
//
// To add charm to the project: place it ONLY in tui_*.go files (slice 2b).
func TestEnginePurity_NoCharmImports(t *testing.T) {
	// Locate the repo root by finding this test file's directory.
	// go test sets the working directory to the package directory so we can
	// use "." directly, but we resolve explicitly for robustness.
	repoRoot, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	fset := token.NewFileSet()

	for _, filename := range engineFiles {
		path := filepath.Join(repoRoot, filename)

		// If the file doesn't exist yet (future engine files added before
		// their test is written) we skip rather than fail — the guard is
		// additive; adding to engineFiles makes it strict immediately.
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Logf("SKIP: engine file not found (will guard when created): %s", filename)
			continue
		}

		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Errorf("parse %s: %v", filename, err)
			continue
		}

		for _, imp := range f.Imports {
			// Import paths are quoted strings; strip the surrounding quotes.
			importPath := strings.Trim(imp.Path.Value, `"`)
			if strings.HasPrefix(importPath, "github.com/charmbracelet") {
				t.Errorf(
					"engine purity violation: %s imports charmbracelet package %q\n"+
						"Engine files must not import charm. Place TUI/charm code in tui_*.go files only.",
					filename, importPath,
				)
			}
		}
	}
}
