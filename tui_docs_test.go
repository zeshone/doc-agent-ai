package main

// ---------------------------------------------------------------------------
// Slice 3 unit tests — TDD RED first
// ---------------------------------------------------------------------------
//
// Tests for:
//   - Docs-mode screen: radio-style selection toggles; Tab cycles list ↔ buttons.
//   - In-project mode: skips stepPath and goes directly to install (or stepOverwrite).
//   - Vault mode: includes stepPath before install (or stepOverwrite).
//   - Config pre-fill: mode and vault path pre-filled from AppConfig.
//   - [Back] from docs-mode → stepPlatformSelect (prevStep).
//   - [Back] from stepPath → stepDocsMode (Esc).
//   - [Continue] from docs-mode with vault → stepPath.
//   - [Continue] from docs-mode with in-project → install or stepOverwrite.
//   - Mode-switch notice: shown on docs-mode screen when PrevMode != selected mode
//     (cfgExisted=true, cfg.Mode != selected mode); NOT shown on fresh install.
//   - focusZone movement on docs-mode screen.

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// ---------------------------------------------------------------------------
// Docs-mode screen: radio-style selection
// ---------------------------------------------------------------------------

// TestDocsMode_UpDownMovesModeCursor verifies that up/down/j/k moves
// modeCursor in the docs-mode step.
func TestDocsMode_UpDownMovesModeCursor(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepDocsMode
	m.modeCursor = 0 // vault

	// Down → in-project.
	m = sendKey(t, m, "j")
	if m.modeCursor != 1 {
		t.Fatalf("after j, modeCursor = %d, want 1 (in-project)", m.modeCursor)
	}

	// Down again → clamped at 1.
	m = sendKey(t, m, "j")
	if m.modeCursor != 1 {
		t.Fatalf("modeCursor should clamp at 1, got %d", m.modeCursor)
	}

	// Up → vault.
	m = sendKey(t, m, "k")
	if m.modeCursor != 0 {
		t.Fatalf("after k, modeCursor = %d, want 0 (vault)", m.modeCursor)
	}

	// Up again → clamped at 0.
	m = sendKey(t, m, "k")
	if m.modeCursor != 0 {
		t.Fatalf("modeCursor should clamp at 0, got %d", m.modeCursor)
	}
}

// TestDocsMode_UpArrow_MovesModeCursor verifies the up/down arrow keys work.
func TestDocsMode_UpArrow_MovesModeCursor(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepDocsMode
	m.modeCursor = 0

	m = sendSpecialKey(t, m, tea.KeyDown)
	if m.modeCursor != 1 {
		t.Fatalf("after down arrow, modeCursor = %d, want 1", m.modeCursor)
	}

	m = sendSpecialKey(t, m, tea.KeyUp)
	if m.modeCursor != 0 {
		t.Fatalf("after up arrow, modeCursor = %d, want 0", m.modeCursor)
	}
}

// ---------------------------------------------------------------------------
// Button focus movement via ←/→ on docs-mode screen (no Tab)
// ---------------------------------------------------------------------------

// TestDocsMode_RightArrow_MovesFocusToBack verifies → moves button focus to Back.
func TestDocsMode_RightArrow_MovesFocusToBack(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepDocsMode

	m = sendSpecialKey(t, m, tea.KeyRight)
	if m.docsModeButtons.focus != 1 {
		t.Fatalf("→ should move docsModeButtons.focus to 1; got %d", m.docsModeButtons.focus)
	}
}

// TestDocsMode_LeftArrow_MovesFocusToContinue verifies ← moves button focus back.
func TestDocsMode_LeftArrow_MovesFocusToContinue(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepDocsMode
	m.docsModeButtons.focus = 1 // start at Back

	m = sendSpecialKey(t, m, tea.KeyLeft)
	if m.docsModeButtons.focus != 0 {
		t.Fatalf("← should move docsModeButtons.focus to 0; got %d", m.docsModeButtons.focus)
	}
}

// ---------------------------------------------------------------------------
// Continue from docs-mode: vault → stepPath; in-project → install/overwrite
// ---------------------------------------------------------------------------

// TestDocsMode_Continue_Vault_GoesToPath verifies that Continue from docs-mode
// with vault selected advances to stepPath.
func TestDocsMode_Continue_Vault_GoesToPath(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepDocsMode
	m.modeCursor = 0            // vault
	m.docsModeButtons.focus = 0 // Continue (default)

	m = sendSpecialKey(t, m, tea.KeyEnter)
	if m.step != stepPath {
		t.Fatalf("Continue (vault) should go to stepPath, got %v", m.step)
	}
	if m.mode != ModeVault {
		t.Fatalf("mode should be ModeVault, got %v", m.mode)
	}
}

// TestDocsMode_Continue_InProject_SkipsPath verifies that Continue from docs-mode
// with in-project selected and no already-installed platforms skips both stepPath
// and stepOverwrite, going directly to stepProgress.
func TestDocsMode_Continue_InProject_SkipsPath(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepDocsMode
	m.modeCursor = 1                       // in-project
	m.alreadyInstalled = map[string]bool{} // fresh install
	m.docsModeButtons.focus = 0            // Continue (default)

	m = sendSpecialKey(t, m, tea.KeyEnter)
	// Fresh install, in-project: skip path and overwrite → go straight to progress.
	if m.step != stepProgress {
		t.Fatalf("Continue (in-project, fresh) should skip path+overwrite and go to stepProgress, got %v", m.step)
	}
	if m.mode != ModeInProject {
		t.Fatalf("mode should be ModeInProject, got %v", m.mode)
	}
}

// ---------------------------------------------------------------------------
// [Back] navigation
// ---------------------------------------------------------------------------

// TestDocsMode_Back_ReturnsToPlatformSelect verifies that [Back] from docs-mode
// returns to stepPlatformSelect (prevStep).
func TestDocsMode_Back_ReturnsToPlatformSelect(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepDocsMode
	m.docsModeButtons.focus = 1 // Back

	m = sendSpecialKey(t, m, tea.KeyEnter)
	if m.step != stepPlatformSelect {
		t.Fatalf("Back from docs-mode should go to stepPlatformSelect, got %v", m.step)
	}
}

// TestPath_Back_ReturnsToDocsMode verifies that Esc from stepPath returns to stepDocsMode.
// (Back is now triggered by Esc on the path screen, not by button focus.)
func TestPath_Back_ReturnsToDocsMode(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepPath

	// On the path screen, Esc = Back.
	m = sendKey(t, m, "esc")
	if m.step != stepDocsMode {
		t.Fatalf("Esc from stepPath should go to stepDocsMode, got %v", m.step)
	}
}

// TestPath_TypingB_InsertsIntoPath verifies that typing 'b'/'B' while entering
// the vault path inserts the character (it must NOT be hijacked as a Back
// shortcut — vault paths routinely contain 'b', e.g. C:\Users\bob\docs).
func TestPath_TypingB_InsertsIntoPath(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepPath
	m.pathInput.Focus()

	m = sendKey(t, m, "b")
	if m.step != stepPath {
		t.Fatalf("typing 'b' must stay on stepPath, got %v", m.step)
	}
	if !strings.Contains(m.pathInput.Value(), "b") {
		t.Errorf("typing 'b' should insert it into the path; value = %q", m.pathInput.Value())
	}
}

// TestPath_Esc_ReturnsToDocsMode verifies that Esc on the path step returns to
// stepDocsMode (non-alphabetic Back shortcut that doesn't collide with typing).
func TestPath_Esc_ReturnsToDocsMode(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepPath

	m = sendKey(t, m, "esc")
	if m.step != stepDocsMode {
		t.Fatalf("Esc from stepPath should go to stepDocsMode, got %v", m.step)
	}
}

// ---------------------------------------------------------------------------
// Config pre-fill: mode and vault path
// ---------------------------------------------------------------------------

// TestDocsMode_ConfigPrefillsVaultMode verifies that a saved vault config
// pre-selects vault in the docs-mode screen (modeCursor=0).
func TestDocsMode_ConfigPrefillsVaultMode(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	cfg := AppConfig{Mode: string(ModeVault), Path: "/my/vault/"}
	m := newInstallModelForTest(cfg, true, plats)

	if m.modeCursor != 0 {
		t.Fatalf("modeCursor = %d, want 0 (vault from config)", m.modeCursor)
	}
	if m.mode != ModeVault {
		t.Fatalf("mode = %v, want ModeVault from config", m.mode)
	}
}

// TestDocsMode_ConfigPrefillsInProjectMode verifies that a saved in-project config
// pre-selects in-project in the docs-mode screen (modeCursor=1).
func TestDocsMode_ConfigPrefillsInProjectMode(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	cfg := AppConfig{Mode: string(ModeInProject)}
	m := newInstallModelForTest(cfg, true, plats)

	if m.modeCursor != 1 {
		t.Fatalf("modeCursor = %d, want 1 (in-project from config)", m.modeCursor)
	}
	if m.mode != ModeInProject {
		t.Fatalf("mode = %v, want ModeInProject from config", m.mode)
	}
}

// TestDocsMode_ConfigPrefillsVaultPath verifies that a saved vault path is
// pre-filled in the path textinput.
func TestDocsMode_ConfigPrefillsVaultPath(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	cfg := AppConfig{Mode: string(ModeVault), Path: "/prefilled/path/"}
	m := newInstallModelForTest(cfg, true, plats)

	if m.pathInput.Value() != "/prefilled/path/" {
		t.Fatalf("pathInput = %q, want %q", m.pathInput.Value(), "/prefilled/path/")
	}
}

// ---------------------------------------------------------------------------
// Mode-switch notice on docs-mode screen
// ---------------------------------------------------------------------------

// TestDocsMode_ModeSwitchNotice_ShownOnChange verifies that the mode-switch
// notice is shown on the docs-mode screen when the selected mode differs from
// the saved config mode (PrevMode != selected).
func TestDocsMode_ModeSwitchNotice_ShownOnChange(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	// Prior config = vault; user has now moved cursor to in-project.
	cfg := AppConfig{Mode: string(ModeVault), Path: "/old/docs/", Version: 1}
	m := newInstallModelForTest(cfg, true, plats)
	m.step = stepDocsMode
	m.modeCursor = 1 // in-project selected

	view := m.View()
	if !strings.Contains(view, "Mode change detected") {
		t.Errorf("docs-mode view should show mode-switch notice when mode differs from config;\ngot:\n%s", view)
	}
}

// TestDocsMode_ModeSwitchNotice_HiddenOnFreshInstall verifies that the
// mode-switch notice is NOT shown when there is no prior config (fresh install).
func TestDocsMode_ModeSwitchNotice_HiddenOnFreshInstall(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	// cfgExisted=false → no prior mode to compare against.
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepDocsMode

	view := m.View()
	if strings.Contains(view, "Mode change detected") {
		t.Errorf("docs-mode view should NOT show mode-switch notice on fresh install;\ngot:\n%s", view)
	}
}

// TestDocsMode_ModeSwitchNotice_HiddenWhenModeUnchanged verifies that the
// mode-switch notice is NOT shown when the selected mode matches the saved config.
func TestDocsMode_ModeSwitchNotice_HiddenWhenModeUnchanged(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	cfg := AppConfig{Mode: string(ModeVault), Path: "/docs/", Version: 1}
	m := newInstallModelForTest(cfg, true, plats)
	m.step = stepDocsMode
	m.modeCursor = 0 // vault — same as saved config

	view := m.View()
	if strings.Contains(view, "Mode change detected") {
		t.Errorf("docs-mode view should NOT show mode-switch notice when mode unchanged;\ngot:\n%s", view)
	}
}

// ---------------------------------------------------------------------------
// View: docs-mode screen contains buttons
// ---------------------------------------------------------------------------

// TestDocsMode_View_ContainsContinueAndBack verifies that the docs-mode view
// contains both [Continue] and [Back] in the footer button row.
func TestDocsMode_View_ContainsContinueAndBack(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepDocsMode

	view := m.View()
	if !strings.Contains(view, "Continue") {
		t.Errorf("docs-mode view missing 'Continue';\ngot:\n%s", view)
	}
	if !strings.Contains(view, "Back") {
		t.Errorf("docs-mode view missing 'Back';\ngot:\n%s", view)
	}
}

// TestDocsMode_View_ContainsCompactHeader verifies that the docs-mode view
// contains the compact Zeen header.
func TestDocsMode_View_ContainsCompactHeader(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepDocsMode

	view := m.View()
	if !strings.Contains(view, "Zeen") {
		t.Errorf("docs-mode view should contain compact Zeen header;\ngot:\n%s", view)
	}
}

// TestPath_View_ContainsContinueAndBack verifies that the path-entry view
// contains both [Continue] and [Back] in the footer button row.
func TestPath_View_ContainsContinueAndBack(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepPath

	view := m.View()
	if !strings.Contains(view, "Continue") {
		t.Errorf("path view missing 'Continue';\ngot:\n%s", view)
	}
	if !strings.Contains(view, "Back") {
		t.Errorf("path view missing 'Back';\ngot:\n%s", view)
	}
}

// ---------------------------------------------------------------------------
// prevStep: docs-mode → platform-select
// ---------------------------------------------------------------------------

// TestPrevStep_DocsMode_ReturnsPlatformSelectFromSlice3 re-asserts the frozen
// prevStep contract required by slice 3 navigation.
func TestPrevStep_DocsMode_ReturnsPlatformSelectFromSlice3(t *testing.T) {
	got := prevStep(stepDocsMode)
	if got != stepPlatformSelect {
		t.Fatalf("prevStep(stepDocsMode) = %v, want stepPlatformSelect", got)
	}
}

// TestPrevStep_Path_ReturnsDocsMode verifies the new BACK edge: stepPath → stepDocsMode.
func TestPrevStep_Path_ReturnsDocsMode(t *testing.T) {
	got := prevStep(stepPath)
	if got != stepDocsMode {
		t.Fatalf("prevStep(stepPath) = %v, want stepDocsMode", got)
	}
}
