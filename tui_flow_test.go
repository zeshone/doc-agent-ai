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

// TestGolden_OverwriteConsolidated captures the consolidated overwrite screen.
// Uses a model with one already-installed platform among two selected.
func TestGolden_OverwriteConsolidated(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	// Mark the first platform as already installed.
	m.alreadyInstalled = map[string]bool{plats[0].ID(): true}
	m.step = stepOverwrite
	m.overwriteChoice = 0        // overwrite-all (default)
	m.overwriteButtons.focus = 0 // [Install] focused
	view := m.View()
	assertGolden(t, view)
}

// TestGolden_ProgressMidState captures the progress step at a mid-state:
// first platform Installing, second platform Pending. Uses NoColor + fixed
// dimensions for deterministic golden output (no ANSI, no animation frames).
func TestGolden_ProgressMidState(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	if len(plats) < 2 {
		t.Skip("need at least 2 platforms for mid-state golden")
	}
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepProgress
	m.installing = true
	m.checklist = []checklistItem{
		{platformID: plats[0].ID(), state: stateChecklist_Installing},
		{platformID: plats[1].ID(), state: stateChecklist_Pending},
	}
	m.checklistCursor = 1
	view := m.View()
	assertGolden(t, view)
}

// TestGolden_ProgressSkipped captures the progress step when one platform is
// skipped (install only missing with an already-installed platform).
func TestGolden_ProgressSkipped(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	if len(plats) < 2 {
		t.Skip("need at least 2 platforms for skipped golden")
	}
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepProgress
	m.installing = true
	m.checklist = []checklistItem{
		{platformID: plats[0].ID(), state: stateChecklist_Skipped},
		{platformID: plats[1].ID(), state: stateChecklist_Done},
	}
	m.checklistCursor = 2
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

// TestTUIFlow_InProject exercises the in-project navigation using teatest:
// platform select (all) → docs-mode (in-project) → stepOverwrite (with an
// already-installed platform, so no engine goroutine during teardown).
//
// We verify wizard navigation only; engine correctness is in execute_install_test.go.
func TestTUIFlow_InProject(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	// Mark a platform as already installed so the flow routes to stepOverwrite
	// rather than stepProgress (which runs the engine goroutine during teardown).
	m.alreadyInstalled = map[string]bool{plats[0].ID(): true}

	tm := teatest.NewTestModel(
		t,
		m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Enter on platform select: Continue is default focus → stepDocsMode.
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	// Navigate to in-project (j = down) in docs-mode.
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune("j")}))
	// Enter → in-project Continue (default focus=0) → stepOverwrite (platform already installed).
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))

	// Ctrl+C to quit from overwrite screen.
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyCtrlC}))

	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
	final := tm.FinalModel(t).(InstallModel)

	if final.mode != ModeInProject {
		t.Fatalf("expected mode=in-project after j+enter, got %v", final.mode)
	}
	// With installed platform: in-project → stepOverwrite (not stepPath, not stepProgress).
	if final.step != stepOverwrite {
		t.Fatalf("expected stepOverwrite (in-project, platform already installed), got %v", final.step)
	}
}

// TestTUIFlow_Vault exercises the vault flow navigation.
// With an already-installed platform, vault path entry leads to stepOverwrite
// (engine goroutine never runs, no temp-dir cleanup issues on Windows).
func TestTUIFlow_Vault(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	// Mark a platform as already installed so path-entry routes to stepOverwrite,
	// not stepProgress (which would run the engine goroutine during teardown).
	m.alreadyInstalled = map[string]bool{plats[0].ID(): true}

	tm := teatest.NewTestModel(
		t,
		m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Platform step → Enter (Continue is default focus=0) → stepDocsMode.
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	// DocsMode: vault is default (modeCursor=0), Enter (Continue default) → stepPath.
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))

	// Now at stepPath — type a vault path.
	for _, ch := range "/my/vault/" {
		tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{ch}}))
	}
	// Enter → Continue (path non-empty) → some platforms installed → stepOverwrite.
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))

	// Ctrl+C to quit from overwrite screen.
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyCtrlC}))

	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	final := tm.FinalModel(t).(InstallModel)

	if final.mode != ModeVault {
		t.Fatalf("expected mode=vault, got %v", final.mode)
	}
	// With an installed platform, path entry → stepOverwrite (not stepProgress).
	if final.step != stepOverwrite {
		t.Fatalf("expected stepOverwrite after vault path (platform already installed), got %v", final.step)
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
// Slice 2 teatest: Back navigation
// ---------------------------------------------------------------------------

// TestTUIFlow_PlatformSelect_BackReturnsToWelcome exercises the BACK path using
// teatest: from platform-select, → to focus Back, Enter → must return to stepWelcome.
func TestTUIFlow_PlatformSelect_BackReturnsToWelcome(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	// → moves to Back button (index 1) — no Tab needed.
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyRight}))
	// Enter → activate Back → stepWelcome.
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	// Ctrl+C to quit from Welcome.
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyCtrlC}))

	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
	final := tm.FinalModel(t).(InstallModel)

	if final.step != stepWelcome {
		t.Fatalf("expected stepWelcome after Back, got %v", final.step)
	}
}

// TestTUIFlow_Welcome_Continue_ToPlatformSelect exercises the Welcome Continue
// button advancing to stepPlatformSelect.
func TestTUIFlow_Welcome_Continue_ToPlatformSelect(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModel(AppConfig{}, false, testManifest(), "dist", plats, NoColor())
	m.width = 80
	m.height = 24
	// Start at Welcome (production default).

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	// Enter on Welcome → Continue (index 0, already focused) → stepPlatformSelect.
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	// Ctrl+C to quit from platform select.
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyCtrlC}))

	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
	final := tm.FinalModel(t).(InstallModel)

	if final.step != stepPlatformSelect {
		t.Fatalf("expected stepPlatformSelect after Welcome Continue, got %v", final.step)
	}
}

// ---------------------------------------------------------------------------
// Slice 3 teatest flows: docs-mode navigation
// ---------------------------------------------------------------------------

// TestTUIFlow_DocsMode_BackReturnsToPlatformSelect exercises the BACK path from
// docs-mode back to platform-select via teatest.
func TestTUIFlow_DocsMode_BackReturnsToPlatformSelect(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepDocsMode

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	// → moves to Back button (index 1).
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyRight}))
	// Enter → activate Back → stepPlatformSelect.
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	// Ctrl+C to quit.
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyCtrlC}))

	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
	final := tm.FinalModel(t).(InstallModel)

	if final.step != stepPlatformSelect {
		t.Fatalf("expected stepPlatformSelect after Back from docs-mode, got %v", final.step)
	}
}

// TestTUIFlow_DocsMode_VaultGoesToPath exercises the vault path via teatest:
// docs-mode (vault selected) → Continue → stepPath.
func TestTUIFlow_DocsMode_VaultGoesToPath(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepDocsMode
	m.modeCursor = 0 // vault

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	// Enter → Continue (default focus=0) → stepPath.
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	// Ctrl+C to quit from path.
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyCtrlC}))

	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
	final := tm.FinalModel(t).(InstallModel)

	if final.step != stepPath {
		t.Fatalf("expected stepPath after vault Continue, got %v", final.step)
	}
	if final.mode != ModeVault {
		t.Fatalf("expected ModeVault, got %v", final.mode)
	}
}

// TestTUIFlow_DocsMode_InProjectSkipsPath exercises the in-project path via
// teatest: docs-mode (in-project selected) → Continue → stepOverwrite (with an
// already-installed platform, so overwrite is shown — but path is skipped).
// Verifies that the wizard does NOT navigate through stepPath for in-project mode.
func TestTUIFlow_DocsMode_InProjectSkipsPath(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	// Mark a platform as already installed so the flow routes to stepOverwrite
	// rather than stepProgress (which would run the engine goroutine in teardown).
	m.alreadyInstalled = map[string]bool{plats[0].ID(): true}
	m.step = stepDocsMode
	m.modeCursor = 0 // start at vault

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	// j moves to in-project, Enter → Continue (default focus=0) → stepOverwrite.
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune("j")}))
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	// Ctrl+C to quit from overwrite screen.
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyCtrlC}))

	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
	final := tm.FinalModel(t).(InstallModel)

	if final.mode != ModeInProject {
		t.Fatalf("expected ModeInProject, got %v", final.mode)
	}
	// In-project skips stepPath → should be at stepOverwrite (not stepPath, not stepProgress).
	if final.step == stepPath {
		t.Fatal("in-project Continue must NOT navigate through stepPath")
	}
	if final.step != stepOverwrite {
		t.Fatalf("expected stepOverwrite (in-project with installed platform), got %v", final.step)
	}
}

// TestTUIFlow_Path_BackReturnsToDocsMode exercises the Esc (Back) path from
// stepPath back to stepDocsMode via teatest.
func TestTUIFlow_Path_BackReturnsToDocsMode(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepPath

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	// Esc = Back on path screen (textinput-owned; no button focus).
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyEsc}))
	// Ctrl+C to quit.
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyCtrlC}))

	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
	final := tm.FinalModel(t).(InstallModel)

	if final.step != stepDocsMode {
		t.Fatalf("expected stepDocsMode after Esc from path, got %v", final.step)
	}
}

// ---------------------------------------------------------------------------
// View output content tests (not golden, just key string assertions)
// ---------------------------------------------------------------------------

// TestViewContainsZeenHeader verifies the compact header is present on inner screens.
// The Welcome screen has its own branded lockup; inner screens use renderCompactHeader
// which renders "Zeen" (not the legacy "doc-agent-ai" banner).
func TestViewContainsZeenHeader(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	view := m.View()
	if !strings.Contains(view, "Zeen") {
		t.Errorf("inner screen header missing 'Zeen'; got:\n%s", view)
	}
}
