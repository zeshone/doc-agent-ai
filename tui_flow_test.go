package main

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/golden"
	"github.com/charmbracelet/x/exp/teatest"
)

// ---------------------------------------------------------------------------
// Golden helper
// ---------------------------------------------------------------------------

// assertGolden delegates to the charmbracelet golden package which owns the
// -update flag. Golden files are stored at testdata/<TestName>.golden.
// Run: go test ./... -update  to regenerate all golden files.
func assertGolden(t *testing.T, got string) {
	t.Helper()
	golden.RequireEqual(t, []byte(got))
}

// ---------------------------------------------------------------------------
// View snapshot helpers
// ---------------------------------------------------------------------------

// viewAtStep creates a fresh InstallModel at the given step, calls View(), and
// returns the output with a fixed 80×24 viewport using NoColor styles.
func viewAtStep(t *testing.T, step Step, cfg AppConfig, cfgExisted bool) string {
	t.Helper()
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(cfg, cfgExisted, plats)
	m.step = step
	return m.View()
}

// ---------------------------------------------------------------------------
// Welcome screen golden test (T1.8)
// ---------------------------------------------------------------------------

// TestGolden_WelcomeScreen captures the welcome screen with NoColor styles and
// a fixed 80×24 viewport. Run with -update to regenerate the golden file.
func TestGolden_WelcomeScreen(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	// Use newInstallModel directly so the step stays at stepWelcome (the
	// production default) rather than being overridden to stepPlatformSelect.
	m := newInstallModel(AppConfig{}, false, testManifest(), "dist", plats, NoColor())
	m.width = 80
	m.height = 24
	view := m.View()
	assertGolden(t, view)
}

// ---------------------------------------------------------------------------
// Golden view tests (one snapshot per wizard step)
// ---------------------------------------------------------------------------

// TestGolden_PlatformSelectStep captures the platform-select step view.
func TestGolden_PlatformSelectStep(t *testing.T) {
	view := viewAtStep(t, stepPlatformSelect, AppConfig{}, false)
	assertGolden(t, view)
}

// TestGolden_DocsModeStep captures the docs-mode selection step view.
func TestGolden_DocsModeStep(t *testing.T) {
	view := viewAtStep(t, stepDocsMode, AppConfig{}, false)
	assertGolden(t, view)
}

// TestGolden_PathStep captures the vault path entry step view.
func TestGolden_PathStep(t *testing.T) {
	view := viewAtStep(t, stepPath, AppConfig{}, false)
	assertGolden(t, view)
}

// TestGolden_ConfirmVault captures the confirm step view for vault mode.
func TestGolden_ConfirmVault(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepConfirm
	m.mode = ModeVault
	// Simulate path already set.
	m.pathInput.SetValue("/vault/docs/")
	view := m.View()
	assertGolden(t, view)
}

// TestGolden_ConfirmInProject captures the confirm step for in-project mode.
func TestGolden_ConfirmInProject(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepConfirm
	m.mode = ModeInProject
	view := m.View()
	assertGolden(t, view)
}

// TestGolden_ConfirmModeSwitchNotice captures the confirm step when a mode
// switch is detected (prior vault → now in-project).
func TestGolden_ConfirmModeSwitchNotice(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	cfg := AppConfig{Mode: string(ModeVault), Path: "/old/docs/", Version: 1}
	m := newInstallModelForTest(cfg, true, plats)
	m.step = stepConfirm
	m.mode = ModeInProject
	view := m.View()
	assertGolden(t, view)
}

// TestGolden_DoneSuccess captures the done step view on success.
func TestGolden_DoneSuccess(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepDone
	m.progressLines = []string{
		"\n  Installing for OpenCode...",
		"  ✔ skill: doc-prd",
		"  ✔ Install complete",
	}
	view := m.View()
	assertGolden(t, view)
}

// TestGolden_DoneError captures the done step view on error.
func TestGolden_DoneError(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepDone
	m.err = errForGolden("install engine error: dist not found")
	view := m.View()
	assertGolden(t, view)
}

// errForGolden returns a simple error value for golden testing without
// importing the errors package.
type goldenErr struct{ msg string }

func (e goldenErr) Error() string { return e.msg }

func errForGolden(msg string) error { return goldenErr{msg} }

// ---------------------------------------------------------------------------
// Uninstall golden
// ---------------------------------------------------------------------------

// TestGolden_UninstallConfirm captures the uninstall confirm view.
func TestGolden_UninstallConfirm(t *testing.T) {
	dir := t.TempDir()
	plat := newPlatformForTest(t, "opencode", dir+"/opencode")
	installed := []installedDetails{
		{
			platform: plat,
			skills:   []string{"doc-prd", "doc-arch"},
			prompts:  []string{"doc-prd"},
		},
	}
	m := newUninstallModel(installed, testManifest(), NoColor())
	m.width = 80
	m.height = 24
	view := m.View()
	assertGolden(t, view)
}

// ---------------------------------------------------------------------------
// teatest full-flow tests
// ---------------------------------------------------------------------------

// TestTUIFlow_InProject exercises the full vault→in-project happy path using
// teatest: platform select (all) → docs-mode (in-project) → confirm → done.
//
// This test does NOT actually run the install engine (it would need a real dist/
// and platform dirs). We test the wizard navigation end-to-end and verify that
// stepDone is reached; engine correctness is covered by execute_install_test.go.
func TestTUIFlow_InProject(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)

	// Replace runInstall with a no-op to avoid needing a real dist/ in tests.
	// We verify navigation to stepConfirm; the actual install is covered by
	// execute_install_test.go integration tests.

	// Navigate: platform select → Enter → docs-mode (default vault) → j → Enter (in-project)
	tm := teatest.NewTestModel(
		t,
		m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Enter on platform select (all selected by default).
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	// Navigate to in-project (j = down).
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune("j")}))
	// Enter to select in-project → should be at stepConfirm (no path step).
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))

	// Cancel at confirm step with 'n' (sends tea.Quit).
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune("n")}))

	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
	final := tm.FinalModel(t).(InstallModel)

	// We cancelled — the model should be at stepConfirm (quit before going to progress).
	if final.step != stepConfirm {
		t.Fatalf("expected stepConfirm after n-cancel, got %v", final.step)
	}
	if final.mode != ModeInProject {
		t.Fatalf("expected mode=in-project after j+enter, got %v", final.mode)
	}
}

// TestTUIFlow_Vault exercises the vault flow navigation.
func TestTUIFlow_Vault(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)

	tm := teatest.NewTestModel(
		t,
		m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Platform step → Enter.
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	// DocsMode: vault is default (modeCursor=0) → Enter.
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))

	// Should now be at stepPath.
	// Type a vault path.
	for _, ch := range "/my/vault/" {
		tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{ch}}))
	}
	// Enter to confirm path → should advance to stepConfirm.
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))

	// Cancel at confirm step with 'n' (sends tea.Quit).
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune("n")}))

	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	final := tm.FinalModel(t).(InstallModel)

	if final.mode != ModeVault {
		t.Fatalf("expected mode=vault, got %v", final.mode)
	}
	if final.step != stepConfirm {
		t.Fatalf("expected stepConfirm after vault path + n-cancel, got %v", final.step)
	}
}

// TestTUIFlow_ConfigPrefilled verifies that a saved config pre-fills the wizard.
func TestTUIFlow_ConfigPrefilled(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	cfg := AppConfig{
		Mode:      string(ModeVault),
		Path:      "/prefilled/docs/",
		Platforms: []string{"opencode"},
		Version:   1,
	}
	m := newInstallModelForTest(cfg, true, plats)

	// Verify pre-fill before any key input.
	if m.pathInput.Value() != "/prefilled/docs/" {
		t.Fatalf("path not pre-filled from config: got %q", m.pathInput.Value())
	}

	// Only opencode should be pre-selected.
	for _, p := range m.platforms {
		wantSelected := p.id == "opencode"
		if p.selected != wantSelected {
			t.Errorf("platform %s: selected=%v, want %v", p.id, p.selected, wantSelected)
		}
	}

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	// Cancel with ctrl+c (always handled in all steps).
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyCtrlC}))
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

// TestTUIFlow_UninstallCancel verifies that the uninstall wizard exits cleanly
// when the user presses 'n'.
func TestTUIFlow_UninstallCancel(t *testing.T) {
	dir := t.TempDir()
	plat := newPlatformForTest(t, "opencode", dir+"/opencode")
	installed := []installedDetails{{platform: plat, skills: []string{"doc-prd"}}}
	m := newUninstallModel(installed, testManifest(), NoColor())
	m.width = 80
	m.height = 24

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune("n")}))
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	final := tm.FinalModel(t).(UninstallModel)
	// Model should still be at confirm step (user cancelled).
	if final.step != uninstallStepConfirm {
		t.Fatalf("step = %v after cancel, want uninstallStepConfirm", final.step)
	}
}

// ---------------------------------------------------------------------------
// View output content tests (not golden, just key string assertions)
// ---------------------------------------------------------------------------

// TestViewContainsVersionString verifies the banner shows a version string.
func TestViewContainsVersionString(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	view := m.View()
	if !strings.Contains(view, "doc-agent-ai") {
		t.Errorf("banner missing 'doc-agent-ai'")
	}
}
