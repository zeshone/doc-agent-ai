package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	configpkg "github.com/zeshone/doc-agent-ai/internal/config"
	installpkg "github.com/zeshone/doc-agent-ai/internal/install"
	"golang.org/x/term"
)

// ---------------------------------------------------------------------------
// TTY detection
// ---------------------------------------------------------------------------

// IsTerminal returns true if both stdin and stdout refer to a real TTY.
// Uses golang.org/x/term which works on all supported OSes (Linux, macOS,
// Windows) without cgo.
func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) &&
		term.IsTerminal(int(os.Stdout.Fd()))
}

// ---------------------------------------------------------------------------
// runInstallTUI — entry point for the install wizard
// ---------------------------------------------------------------------------

// runInstallTUI bootstraps the Bubbletea install wizard and blocks until the
// user completes or cancels it. It validates the provided bundle, loads saved
// config for defaults, detects platforms, and runs the Bubbletea program.
//
// Returns an error if the bundle cannot be validated or the install engine fails.
// Returns (nil) when the user cancels (tea.Quit without error).
func RunInstallTUI(bundle installpkg.Bundle) error {
	manifest := bundle.Manifest
	if missing := installpkg.ValidateBundle(bundle); len(missing) > 0 {
		return fmt.Errorf("incomplete bundle: %s", installpkg.SummarizeMissingArtifacts(missing))
	}

	// Step 2: Load config for pre-fill defaults.
	cfg, cfgExisted, err := configpkg.Load()
	if err != nil {
		// Non-fatal: warn and proceed without defaults.
		fmt.Fprintf(os.Stderr, "  warning: could not read config: %v\n", err)
		cfg = configpkg.AppConfig{}
		cfgExisted = false
	}

	// Step 3: Detect platforms.
	allPlatforms := installpkg.DetectAllPlatforms(manifest)
	if len(allPlatforms) == 0 {
		return fmt.Errorf("no supported platform detected — install opencode, claude, copilot, qwen, or pi first")
	}

	// Step 4: Launch the Bubbletea program.
	model := newInstallModel(cfg, cfgExisted, bundle, allPlatforms, NewStyles())
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
func RunUninstallTUI(bundle installpkg.Bundle) error {
	manifest := bundle.Manifest

	// Step 2: Detect platforms.
	allPlatforms := installpkg.DetectAllPlatforms(manifest)

	// Step 3: Check what's installed.
	installed := installpkg.CheckWhatIsInstalled(manifest, allPlatforms)
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

func RunApp(bundle installpkg.Bundle) error {
	var err error
	model := RootModel{screen: screenHome, bundle: bundle, styles: NewStyles(), width: 80, height: 24}
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err = p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}
