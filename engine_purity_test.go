package docagent

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
// (internal/tui/tui_*.go) are the only files allowed to import charm.
//
// Paths are relative to the repo root. Each file must exist — a missing path
// is a hard failure (not a skip) so that a future rename is caught immediately.
var engineFiles = []string{
	// Root package — generate pipeline and headless CLI
	"generate.go",
	"template.go",
	"headless.go",
	// internal/install — pure install engine
	"internal/install/install.go",
	"internal/install/platform.go",
	"internal/install/manifest.go",
	"internal/install/reporter.go",
	"internal/install/execute_install.go",
	"internal/install/uninstall.go",
	// internal/config — plan resolution, config I/O
	"internal/config/config.go",
	"internal/config/plan.go",
	"internal/config/resolve.go",
	// CLI entry point — must delegate to charm-free wrappers only
	"cmd/doc-agent-ai/main.go",
}

// TestEnginePurity_NoCharmImports asserts that none of the engine-layer Go
// source files import any charmbracelet/* package. This prevents architectural
// erosion and ensures the engine stays charm-free so it can be unit-tested
// without a TTY and consumed by non-TUI headless paths.
//
// To add charm to the project: place it ONLY in internal/tui/tui_*.go files.
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

		// All listed engine files must exist. A missing file means the path
		// is stale and the guard is no longer protecting that code — hard fail.
		if _, err := os.Stat(path); err != nil {
			t.Errorf("CHECKED: engine file not found — update engineFiles if the file was moved or renamed: %s", filename)
			continue
		}

		t.Logf("CHECKED: %s", filename)

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
						"Engine files must not import charm. Place TUI/charm code in internal/tui/tui_*.go files only.",
					filename, importPath,
				)
			}
		}
	}
}
