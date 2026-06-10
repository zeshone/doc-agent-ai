package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

// ---------------------------------------------------------------------------
// TTY detection
// ---------------------------------------------------------------------------

// isTerminal returns true if both stdin and stdout refer to a real TTY.
// Uses golang.org/x/term which works on all supported OSes (Linux, macOS,
// Windows) without cgo.
func isTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) &&
		term.IsTerminal(int(os.Stdout.Fd()))
}

// ---------------------------------------------------------------------------
// runInstallTUI — entry point for the install wizard
// ---------------------------------------------------------------------------

// runInstallTUI bootstraps the Bubbletea install wizard and blocks until the
// user completes or cancels it. It handles:
//   - Dist generation (auto-generate if missing)
//   - Config loading (pre-fill defaults)
//   - Platform detection
//   - Running the Bubbletea program
//
// Returns an error if dist cannot be loaded or the install engine fails.
// Returns (nil) when the user cancels (tea.Quit without error).
func runInstallTUI() error {
	// Step 0: Ensure dist/ exists (auto-generate if missing).
	distDir := "dist"
	if _, err := os.Stat(filepath.Join(distDir, "manifest.json")); os.IsNotExist(err) {
		fmt.Fprintf(os.Stdout, "  Auto-generating dist/ from embedded source...\n")
		if err := generate(distDir); err != nil {
			return fmt.Errorf("auto-generate dist: %w", err)
		}
	}

	// Step 1: Read and validate manifest.
	manifest, err := readManifestFrom(distDir)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}

	if missing := validateDist(manifest, distDir); len(missing) > 0 {
		fmt.Fprintln(os.Stderr, "  dist/ is incomplete — run `doc-agent-ai generate` first.")
		for _, m := range missing {
			fmt.Fprintln(os.Stderr, "    missing: "+m)
		}
		return fmt.Errorf("incomplete dist: %d missing artifacts", len(missing))
	}

	// Step 2: Load config for pre-fill defaults.
	cfg, cfgExisted, err := loadConfig()
	if err != nil {
		// Non-fatal: warn and proceed without defaults.
		fmt.Fprintf(os.Stderr, "  warning: could not read config: %v\n", err)
		cfg = AppConfig{}
		cfgExisted = false
	}

	// Step 3: Detect platforms.
	allPlatforms := detectAllPlatforms(manifest)
	if len(allPlatforms) == 0 {
		return fmt.Errorf("no supported platform detected — install opencode, claude, copilot, qwen, or pi first")
	}

	// Step 4: Launch the Bubbletea program.
	model := newInstallModel(cfg, cfgExisted, manifest, distDir, allPlatforms, NewStyles())
	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	// Propagate any install error captured in the final model state.
	if m, ok := finalModel.(InstallModel); ok {
		if m.err != nil {
			return m.err
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// runUninstallTUI — entry point for the uninstall wizard
// ---------------------------------------------------------------------------

// runUninstallTUI bootstraps the Bubbletea uninstall wizard and blocks until
// the user completes or cancels it.
func runUninstallTUI() error {
	// Step 0: Ensure dist/ exists.
	distDir := "dist"
	if _, err := os.Stat(filepath.Join(distDir, "manifest.json")); os.IsNotExist(err) {
		fmt.Fprintf(os.Stdout, "  Auto-generating dist/ from embedded source...\n")
		if err := generate(distDir); err != nil {
			return fmt.Errorf("auto-generate dist: %w", err)
		}
	}

	// Step 1: Read manifest.
	manifest, err := readManifestFrom(distDir)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}

	// Step 2: Detect platforms.
	allPlatforms := detectAllPlatforms(manifest)

	// Step 3: Check what's installed.
	installed := checkWhatIsInstalled(manifest, allPlatforms)
	if len(installed) == 0 {
		fmt.Println("\n  doc-agent-ai does not appear to be installed on detected platforms.")
		fmt.Println("  Nothing to uninstall.")
		return nil
	}

	// Step 4: Launch the Bubbletea uninstall program.
	model := newUninstallModel(installed, manifest, NewStyles())
	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	// Propagate any error from the final model state.
	if m, ok := finalModel.(UninstallModel); ok {
		if m.err != nil {
			return m.err
		}
	}

	return nil
}
