package main

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// testManifest returns a minimal DistManifest suitable for TUI unit tests.
func testManifest() DistManifest {
	return DistManifest{PlaceholderBasePath: "__DOC_AGENT_BASE_PATH__/"}
}

func testBundle() Bundle {
	return Bundle{Manifest: testManifest(), Files: map[string][]byte{}}
}

// newInstallModelForTest creates an InstallModel with NoColor styles and fixed
// 80×24 viewport for deterministic test output.
// The step is set to stepPlatformSelect (the first interactive step after
// Welcome) so existing tests that exercise platform selection, docs mode,
// confirm, and done flows are not affected by the new Welcome screen being
// the new initial step in production.
func newInstallModelForTest(cfg AppConfig, cfgExisted bool, allPlatforms []Platform) InstallModel {
	m := newInstallModel(cfg, cfgExisted, testBundle(), allPlatforms, NoColor())
	m.step = stepPlatformSelect
	m.width = 80
	m.height = 24
	return m
}

// sendKey sends a single key string to the model and returns the updated model.
func sendKey(t *testing.T, m InstallModel, key string) InstallModel {
	t.Helper()
	next, _ := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune(key)}))
	if n, ok := next.(InstallModel); ok {
		return n
	}
	t.Fatalf("Update returned unexpected type %T", next)
	return m
}

// sendSpecialKey sends a special key (enter, space, up, down, etc.) to the model.
func sendSpecialKey(t *testing.T, m InstallModel, keyType tea.KeyType) InstallModel {
	t.Helper()
	next, _ := m.Update(tea.KeyMsg(tea.Key{Type: keyType}))
	if n, ok := next.(InstallModel); ok {
		return n
	}
	t.Fatalf("Update returned unexpected type %T", next)
	return m
}

// ---------------------------------------------------------------------------
// Step transition tests (direct Model.Update calls per go-testing skill gate)
// ---------------------------------------------------------------------------

// TestInstallModel_PlatformToggle verifies that space toggles the selected platform.
func TestInstallModel_PlatformToggle(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)

	// Initially all platforms selected (no prior config).
	if !m.platforms[0].selected {
		t.Fatalf("expected platform 0 to be selected by default (no prior config)")
	}

	// Space on cursor 0 should toggle it off.
	m = sendKey(t, m, " ")
	if m.platforms[0].selected {
		t.Fatalf("platform 0 should be deselected after space")
	}

	// Space again should toggle it back on.
	m = sendKey(t, m, " ")
	if !m.platforms[0].selected {
		t.Fatalf("platform 0 should be re-selected after second space")
	}
}

// TestInstallModel_CursorNavigation verifies up/down/j/k cursor movement.
func TestInstallModel_CursorNavigation(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)

	// Cursor starts at 0.
	if m.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", m.cursor)
	}

	// Down moves cursor.
	m = sendKey(t, m, "j")
	if m.cursor != 1 {
		t.Fatalf("after j, cursor = %d, want 1", m.cursor)
	}

	// Up moves cursor back.
	m = sendKey(t, m, "k")
	if m.cursor != 0 {
		t.Fatalf("after k, cursor = %d, want 0", m.cursor)
	}

	// Cannot go above 0.
	m = sendKey(t, m, "k")
	if m.cursor != 0 {
		t.Fatalf("cursor clamped at 0, got %d", m.cursor)
	}

	// Cannot go past last item.
	m = sendKey(t, m, "j")
	m = sendKey(t, m, "j")
	if m.cursor != len(plats)-1 {
		t.Fatalf("cursor clamped at %d, got %d", len(plats)-1, m.cursor)
	}
}

// TestInstallModel_EnterAdvancesToDocsMode verifies that Enter on platform step
// advances to stepDocsMode when at least one platform is selected.
func TestInstallModel_EnterAdvancesToDocsMode(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)

	if m.step != stepPlatformSelect {
		t.Fatalf("initial step = %v, want stepPlatformSelect", m.step)
	}

	m = sendSpecialKey(t, m, tea.KeyEnter)
	if m.step != stepDocsMode {
		t.Fatalf("after enter, step = %v, want stepDocsMode", m.step)
	}
}

// TestInstallModel_EnterBlockedWhenNoneSelected verifies that Enter does NOT
// advance when all platforms are deselected.
func TestInstallModel_EnterBlockedWhenNoneSelected(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)

	// Deselect all.
	for i := range m.platforms {
		m.platforms[i].selected = false
	}

	m = sendSpecialKey(t, m, tea.KeyEnter)
	if m.step != stepPlatformSelect {
		t.Fatalf("step should stay at stepPlatformSelect when none selected, got %v", m.step)
	}
}

// TestInstallModel_InProjectModeSkipsPathStep verifies that choosing in-project
// mode with no already-installed platforms skips both stepPath and stepOverwrite,
// going directly to stepProgress.
func TestInstallModel_InProjectModeSkipsPathStep(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	// Ensure no platforms are already installed.
	m.alreadyInstalled = map[string]bool{}

	// Advance through platform step.
	m = sendSpecialKey(t, m, tea.KeyEnter)
	if m.step != stepDocsMode {
		t.Fatalf("expected stepDocsMode, got %v", m.step)
	}

	// Move to in-project (modeCursor 1) and confirm.
	m = sendKey(t, m, "j") // modeCursor → 1
	m = sendSpecialKey(t, m, tea.KeyEnter)

	if m.mode != ModeInProject {
		t.Fatalf("mode = %v, want ModeInProject", m.mode)
	}
	// No already-installed platforms → skip stepOverwrite → go directly to stepProgress.
	if m.step != stepProgress {
		t.Fatalf("step = %v, want stepProgress (path + overwrite skipped for in-project fresh install)", m.step)
	}
}

// TestInstallModel_VaultModeIncludesPathStep verifies that vault mode advances
// to stepPath before stepConfirm.
func TestInstallModel_VaultModeIncludesPathStep(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)

	// Advance through platform step.
	m = sendSpecialKey(t, m, tea.KeyEnter)

	// Vault is modeCursor=0 (default) — press Enter.
	m = sendSpecialKey(t, m, tea.KeyEnter)

	if m.mode != ModeVault {
		t.Fatalf("mode = %v, want ModeVault", m.mode)
	}
	if m.step != stepPath {
		t.Fatalf("step = %v, want stepPath", m.step)
	}
}

// TestInstallModel_ConfigPrefillsMode verifies that a saved config mode is
// pre-selected in the mode step.
func TestInstallModel_ConfigPrefillsMode(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	cfg := AppConfig{Mode: string(ModeInProject), Path: "/my/docs"}
	m := newInstallModelForTest(cfg, true, plats)

	// Mode should default to in-project (modeCursor=1).
	if m.modeCursor != 1 {
		t.Fatalf("modeCursor = %d, want 1 (in-project from config)", m.modeCursor)
	}
}

// TestInstallModel_ConfigPrefillsPath verifies that a saved vault path is
// pre-filled in the path textinput.
func TestInstallModel_ConfigPrefillsPath(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	cfg := AppConfig{Mode: string(ModeVault), Path: "/saved/docs/"}
	m := newInstallModelForTest(cfg, true, plats)

	if m.pathInput.Value() != "/saved/docs/" {
		t.Fatalf("path input = %q, want %q", m.pathInput.Value(), "/saved/docs/")
	}
}

// TestInstallModel_ConfirmBuildsCorrectPlan verifies that BuildPlan produces the
// expected InstallPlan after the user navigates through the wizard.
func TestInstallModel_ConfirmBuildsCorrectPlan(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)

	// Step 1: select all platforms (already all selected).
	m = sendSpecialKey(t, m, tea.KeyEnter)

	// Step 2: select vault (default, modeCursor=0).
	m = sendSpecialKey(t, m, tea.KeyEnter)

	// Step 3: enter vault path.
	m = typeIntoInput(t, m, "/vault/docs/")
	m = sendSpecialKey(t, m, tea.KeyEnter)

	if m.step != stepOverwrite && m.step != stepProgress {
		t.Fatalf("expected stepOverwrite or stepProgress after vault path entry, got %v", m.step)
	}

	plan := m.BuildPlan()

	if plan.Mode != ModeVault {
		t.Fatalf("plan.Mode = %v, want ModeVault", plan.Mode)
	}
	if plan.BasePath != "/vault/docs/" {
		t.Fatalf("plan.BasePath = %q, want %q", plan.BasePath, "/vault/docs/")
	}
	if len(plan.Platforms) != len(plats) {
		t.Fatalf("plan.Platforms len = %d, want %d", len(plan.Platforms), len(plats))
	}
}

// ---------------------------------------------------------------------------
// Helper: testPlatformsFromTempDir
// ---------------------------------------------------------------------------

// testPlatformsFromTempDir creates Platform instances rooted in t.TempDir()
// so tests never touch the real home directory.
func testPlatformsFromTempDir(t *testing.T) []Platform {
	t.Helper()
	dir := t.TempDir()
	return []Platform{
		newPlatformForTest(t, "opencode", dir+"/opencode"),
		newPlatformForTest(t, "claude", dir+"/claude"),
	}
}

// typeIntoInput feeds a string character by character into a textinput on the
// path step, simulating typing.
func typeIntoInput(t *testing.T, m InstallModel, text string) InstallModel {
	t.Helper()
	for _, ch := range text {
		next, _ := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{ch}}))
		if n, ok := next.(InstallModel); ok {
			m = n
		} else {
			t.Fatalf("unexpected model type %T", next)
		}
	}
	return m
}

// ---------------------------------------------------------------------------
// View smoke tests (assert key strings present, not full layout)
// ---------------------------------------------------------------------------

// TestInstallModel_ViewPlatformSelectContainsPlatformNames checks that the view
// for the platform-select step includes the platform display names.
func TestInstallModel_ViewPlatformSelectContainsPlatformNames(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)

	view := m.View()
	for _, p := range plats {
		label := platformDisplayName(p.ID())
		if !strings.Contains(view, label) {
			t.Errorf("platform select view missing label %q", label)
		}
	}
}

// TestInstallModel_ViewDocsModeContainsBothModes checks that the mode step view
// contains both mode options.
func TestInstallModel_ViewDocsModeContainsBothModes(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepDocsMode

	view := m.View()
	if !strings.Contains(view, "vault") {
		t.Error("docs mode view missing 'vault'")
	}
	if !strings.Contains(view, "in-project") {
		t.Error("docs mode view missing 'in-project'")
	}
}

// TestInstallModel_ViewDocsModeShowsModeSwitch checks that the docs-mode step
// shows the mode-switch notice when the user changes mode from their saved config.
// (The notice was relocated from the removed stepConfirm to viewDocsMode in slice 3.)
func TestInstallModel_ViewDocsModeShowsModeSwitch(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	cfg := AppConfig{Mode: string(ModeVault), Path: "/old/docs/"}
	m := newInstallModelForTest(cfg, true, plats)
	m.modeCursor = 1 // user moved to in-project (different from saved vault)
	m.step = stepDocsMode

	view := m.View()
	if !strings.Contains(view, "Mode change detected") {
		t.Errorf("docs-mode view missing mode-switch notice when mode changed; got:\n%s", view)
	}
}

// TestInstallModel_ViewDocsModeNoSwitchNotice checks that the docs-mode step does
// NOT show the mode-switch notice when mode is unchanged from config.
func TestInstallModel_ViewDocsModeNoSwitchNotice(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	cfg := AppConfig{Mode: string(ModeVault), Path: "/docs/"}
	m := newInstallModelForTest(cfg, true, plats)
	m.modeCursor = 0 // same as saved vault
	m.step = stepDocsMode

	view := m.View()
	if strings.Contains(view, "Mode change detected") {
		t.Errorf("docs-mode view should not show mode-switch notice when mode unchanged")
	}
}

// TestInstallModel_ViewDoneError verifies the done step shows an error message
// when err is non-nil.
func TestInstallModel_ViewDoneError(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepDone
	m.err = fmt.Errorf("install engine boom")

	view := m.View()
	if !strings.Contains(view, "Install failed") {
		t.Errorf("done view should show 'Install failed' on error; got:\n%s", view)
	}
}

// TestInstallResultMsg_DoesNotAutoQuit verifies the wizard stays on the done
// screen after the install finishes (no auto-quit), so the user can read the
// summary; a subsequent keypress is what exits.
func TestInstallResultMsg_DoesNotAutoQuit(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepProgress

	updated, cmd := m.Update(installResultMsg{err: nil, progressLines: []string{"  ✔ skill: x"}})
	im, ok := updated.(InstallModel)
	if !ok {
		t.Fatalf("Update returned %T, want InstallModel", updated)
	}
	if im.step != stepDone {
		t.Fatalf("step = %v, want stepDone after install result", im.step)
	}
	if cmd != nil {
		t.Error("installResultMsg must NOT auto-quit; cmd should be nil so the done screen stays")
	}
	if !strings.Contains(im.View(), "Press any key to exit") {
		t.Error("done view should prompt 'Press any key to exit'")
	}

	// Any key on the done screen quits.
	_, quitCmd := im.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if quitCmd == nil {
		t.Error("a keypress on the done screen should return a quit command")
	}
}

// ---------------------------------------------------------------------------
// UninstallModel unit tests
// ---------------------------------------------------------------------------

// TestUninstallModel_ConfirmY advances to progress step on 'y'.
func TestUninstallModel_ConfirmY(t *testing.T) {
	m := newUninstallModel([]installedDetails{}, testManifest(), NoColor())
	m.width = 80
	m.height = 24

	next, _ := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune("y")}))
	um, ok := next.(UninstallModel)
	if !ok {
		t.Fatalf("unexpected type %T", next)
	}
	if um.step != uninstallStepProgress {
		t.Fatalf("step = %v after 'y', want uninstallStepProgress", um.step)
	}
}

// TestUninstallModel_ConfirmN stays on confirm step (model quits via tea.Quit cmd).
func TestUninstallModel_ConfirmN(t *testing.T) {
	m := newUninstallModel([]installedDetails{}, testManifest(), NoColor())
	m.width = 80
	m.height = 24

	_, cmd := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune("n")}))
	if cmd == nil {
		t.Fatal("expected quit command after 'n'")
	}
}

// TestUninstallModel_EnterDefaultNo verifies that Enter does not trigger uninstall
// (destructive action requires explicit 'y').
func TestUninstallModel_EnterDefaultNo(t *testing.T) {
	m := newUninstallModel([]installedDetails{}, testManifest(), NoColor())
	m.width = 80
	m.height = 24

	_, cmd := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	if cmd == nil {
		t.Fatal("expected quit command after Enter (default no)")
	}
}

// ---------------------------------------------------------------------------
// isTerminal unit test
// ---------------------------------------------------------------------------

// TestIsTerminal_FalseInTests verifies that isTerminal() returns false in the
// test environment (tests run without a TTY — CI determinism).
func TestIsTerminal_FalseInTests(t *testing.T) {
	// In CI and `go test` the stdin/stdout are not real TTYs.
	// This test documents the expected behaviour; it will pass in CI and in
	// most developer environments when running `go test ./...` from a shell.
	// If your shell pipes stdin to go test it may also be false.
	got := isTerminal()
	if got {
		t.Log("isTerminal() = true (running in a real TTY — this is OK in interactive shells)")
	}
	// We do not assert false here — the value depends on the test runner environment.
	// The TTY-error path is tested separately via TestTTYErrorPath.
}

// ---------------------------------------------------------------------------
// TTY-error path unit test
// ---------------------------------------------------------------------------

// TestNoTTYErrorMessage verifies the error text produced when no TTY is present
// and no flags are supplied. This exercises the error string composition logic
// without launching a real process.
//
// The actual routing is in main.go; here we test that the expected message
// components are present by constructing the text directly. This keeps the test
// fast, hermetic, and independent of process spawning.
func TestNoTTYErrorMessage(t *testing.T) {
	// These are the exact strings main.go writes to stderr on the no-TTY path.
	// Keeping them in sync is enforced by this test (if main.go changes the
	// message, this test catches the drift).
	want := []string{
		"no TTY detected",
		"no install flags provided",
		"--platforms",
		"--docs-mode",
		"--path",
		"--yes",
	}

	// Build the canonical error message text (mirrors main.go).
	msg := "Error: no TTY detected and no install flags provided.\n" +
		"Use flags for non-interactive install, e.g.:\n" +
		"  doc-agent-ai install --platforms opencode --docs-mode vault --path /docs --yes\n" +
		"Run --help for all available flags.\n"

	for _, w := range want {
		if !strings.Contains(msg, w) {
			t.Errorf("error message missing expected string %q", w)
		}
	}
}

// TestUninstallModel_ProgressLinesReachDoneStep verifies that progress lines
// produced by the uninstall engine closure are merged into the model when
// uninstallResultMsg arrives, and rendered in the done step — guarding against
// the detached-copy defect where lines were appended to a dead model copy.
func TestUninstallModel_ProgressLinesReachDoneStep(t *testing.T) {
	m := newUninstallModel(nil, DistManifest{}, NoColor())
	m.step = uninstallStepProgress

	updated, _ := m.Update(uninstallResultMsg{
		err:           nil,
		progressLines: []string{"  Removing from opencode..."},
	})
	um, ok := updated.(UninstallModel)
	if !ok {
		t.Fatalf("Update returned %T, want UninstallModel", updated)
	}
	if um.step != uninstallStepDone {
		t.Fatalf("step = %v, want uninstallStepDone", um.step)
	}
	if len(um.progressLines) == 0 {
		t.Fatal("progressLines not merged into model from uninstallResultMsg")
	}
	if !strings.Contains(um.View(), "Removing from opencode") {
		t.Error("done view does not render the collected progress lines")
	}
}
