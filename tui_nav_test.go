package main

// ---------------------------------------------------------------------------
// nav-adjustment tests — TDD RED first
// ---------------------------------------------------------------------------
//
// New navigation model (no Tab/focusZone):
//   - ↑/↓ (and k/j) always move content selection (platform cursor, mode cursor,
//     overwrite cursor) on every screen that has content.
//   - ←/→ always move the footer button-row focus. Both axes are always live.
//   - Enter activates the currently focused button.
//   - Space toggles the focused platform checkbox (platform screen only).
//   - Tab is NOT bound on any screen; it is ignored (no-op).
//   - Welcome: only buttons — ←/→ move [Continue]/[Quit], Enter activates.
//     ↑/↓ are no-ops on welcome.
//   - Path screen EXCEPTION: textinput owns all keys; ←/→ edit the cursor inside
//     the path, printable chars type. Enter = Continue (if path non-empty).
//     Esc = Back. ←/→ do NOT change button focus on the path screen.
//
// Tests cover: platform-select, docs-mode, overwrite, welcome, and path.

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// ---------------------------------------------------------------------------
// Platform-select: ↑/↓ move content, ←/→ move buttons — simultaneously
// ---------------------------------------------------------------------------

// TestNav_PlatformSelect_UpDownMovesCursor asserts that ↑/↓ move the platform
// cursor regardless of whether ←/→ have been used on the button row.
func TestNav_PlatformSelect_UpDownMovesCursor(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)

	// Initial cursor is 0.
	if m.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", m.cursor)
	}

	// Move button focus first (←/→ should not block ↑/↓).
	m = sendSpecialKey(t, m, tea.KeyRight)
	// Now move content cursor down.
	m = sendSpecialKey(t, m, tea.KeyDown)
	if m.cursor != 1 {
		t.Fatalf("↓ should move cursor to 1; got %d", m.cursor)
	}

	m = sendSpecialKey(t, m, tea.KeyUp)
	if m.cursor != 0 {
		t.Fatalf("↑ should move cursor back to 0; got %d", m.cursor)
	}
}

// TestNav_PlatformSelect_LeftRightMoveButtonFocus asserts that ←/→ move the
// button-row focus regardless of the content cursor position.
func TestNav_PlatformSelect_LeftRightMoveButtonFocus(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)

	// Default: Continue focused (index 0).
	if m.platformSelectButtons.focus != 0 {
		t.Fatalf("initial button focus = %d, want 0 (Continue)", m.platformSelectButtons.focus)
	}

	// → moves to Back (index 1).
	m = sendSpecialKey(t, m, tea.KeyRight)
	if m.platformSelectButtons.focus != 1 {
		t.Fatalf("→ should move button focus to 1 (Back); got %d", m.platformSelectButtons.focus)
	}

	// ← moves back to Continue (index 0).
	m = sendSpecialKey(t, m, tea.KeyLeft)
	if m.platformSelectButtons.focus != 0 {
		t.Fatalf("← should move button focus to 0 (Continue); got %d", m.platformSelectButtons.focus)
	}
}

// TestNav_PlatformSelect_EnterActivatesContinue asserts that Enter activates
// the focused button (Continue when focus=0).
func TestNav_PlatformSelect_EnterActivatesContinue(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	// Default: Continue focused.
	m = sendSpecialKey(t, m, tea.KeyEnter)
	if m.step != stepDocsMode {
		t.Fatalf("Enter on Continue should advance to stepDocsMode; got %v", m.step)
	}
}

// TestNav_PlatformSelect_EnterActivatesBack asserts that Enter activates Back
// when button focus is on Back (index 1).
func TestNav_PlatformSelect_EnterActivatesBack(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)

	// Move to Back button.
	m = sendSpecialKey(t, m, tea.KeyRight)
	m = sendSpecialKey(t, m, tea.KeyEnter)

	if m.step != stepWelcome {
		t.Fatalf("Enter on Back should go to stepWelcome; got %v", m.step)
	}
}

// TestNav_PlatformSelect_SpaceTogglesAlways asserts that Space always toggles
// the focused platform checkbox (no zone gating).
func TestNav_PlatformSelect_SpaceTogglesAlways(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)

	// Move button focus to Back first.
	m = sendSpecialKey(t, m, tea.KeyRight)
	initial := m.platforms[0].selected

	// Space should still toggle the checkbox.
	m = sendKey(t, m, " ")
	if m.platforms[0].selected == initial {
		t.Fatalf("Space should toggle checkbox regardless of button focus; unchanged (was %v)", initial)
	}
}

// TestNav_PlatformSelect_TabIsNoOp asserts that Tab does NOT change any state.
func TestNav_PlatformSelect_TabIsNoOp(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)

	beforeBtnFocus := m.platformSelectButtons.focus
	beforeCursor := m.cursor
	beforeStep := m.step

	// Send Tab — must be a no-op.
	m = sendSpecialKey(t, m, tea.KeyTab)

	if m.platformSelectButtons.focus != beforeBtnFocus {
		t.Errorf("Tab changed button focus from %d to %d; should be no-op", beforeBtnFocus, m.platformSelectButtons.focus)
	}
	if m.cursor != beforeCursor {
		t.Errorf("Tab changed cursor from %d to %d; should be no-op", beforeCursor, m.cursor)
	}
	if m.step != beforeStep {
		t.Errorf("Tab changed step from %v to %v; should be no-op", beforeStep, m.step)
	}
}

// TestNav_PlatformSelect_BothAxesSimultaneous asserts the dual-axis model:
// ↑/↓ affect content and ←/→ affect buttons in the same session without
// any mode switch.
func TestNav_PlatformSelect_BothAxesSimultaneous(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)

	// Move button focus to Back.
	m = sendSpecialKey(t, m, tea.KeyRight) // button focus = Back
	// Move content cursor down.
	m = sendKey(t, m, "j") // cursor = 1
	// Move content cursor up.
	m = sendKey(t, m, "k") // cursor = 0
	// Move button focus back to Continue.
	m = sendSpecialKey(t, m, tea.KeyLeft) // button focus = Continue

	if m.platformSelectButtons.focus != 0 {
		t.Errorf("button focus = %d, want 0 (Continue)", m.platformSelectButtons.focus)
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
}

// TestNav_PlatformSelect_Continue_BlockedWhenNoneSelected_NoZone asserts that
// the zero-selection guard works without any focusZone dependency.
func TestNav_PlatformSelect_Continue_BlockedWhenNoneSelected_NoZone(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)

	// Deselect all.
	for i := range m.platforms {
		m.platforms[i].selected = false
	}

	// Enter on default Continue button — should be blocked.
	m = sendSpecialKey(t, m, tea.KeyEnter)
	if m.step != stepPlatformSelect {
		t.Fatalf("Continue should be blocked when nothing selected; got %v", m.step)
	}
	if m.notice == "" {
		t.Error("notice should be set when Continue is blocked")
	}
}

// ---------------------------------------------------------------------------
// Docs-mode: ↑/↓ move cursor, ←/→ move buttons — simultaneously
// ---------------------------------------------------------------------------

// TestNav_DocsMode_UpDownMovesModeCursor asserts ↑/↓ move modeCursor always.
func TestNav_DocsMode_UpDownMovesModeCursor(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepDocsMode
	m.modeCursor = 0

	// Move button focus first.
	m = sendSpecialKey(t, m, tea.KeyRight)
	// ↓ moves modeCursor.
	m = sendSpecialKey(t, m, tea.KeyDown)
	if m.modeCursor != 1 {
		t.Fatalf("↓ should move modeCursor to 1; got %d", m.modeCursor)
	}
	// ↑ moves modeCursor back.
	m = sendSpecialKey(t, m, tea.KeyUp)
	if m.modeCursor != 0 {
		t.Fatalf("↑ should move modeCursor to 0; got %d", m.modeCursor)
	}
}

// TestNav_DocsMode_LeftRightMoveButtonFocus asserts ←/→ move docsModeButtons focus.
func TestNav_DocsMode_LeftRightMoveButtonFocus(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepDocsMode

	if m.docsModeButtons.focus != 0 {
		t.Fatalf("initial docsModeButtons.focus = %d, want 0", m.docsModeButtons.focus)
	}

	m = sendSpecialKey(t, m, tea.KeyRight)
	if m.docsModeButtons.focus != 1 {
		t.Fatalf("→ should move docsModeButtons.focus to 1; got %d", m.docsModeButtons.focus)
	}

	m = sendSpecialKey(t, m, tea.KeyLeft)
	if m.docsModeButtons.focus != 0 {
		t.Fatalf("← should move docsModeButtons.focus to 0; got %d", m.docsModeButtons.focus)
	}
}

// TestNav_DocsMode_EnterActivatesContinue asserts Enter activates the focused button.
func TestNav_DocsMode_EnterActivatesContinue(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepDocsMode
	m.modeCursor = 0 // vault

	// Enter with default focus=0 (Continue) → stepPath.
	m = sendSpecialKey(t, m, tea.KeyEnter)
	if m.step != stepPath {
		t.Fatalf("Enter on Continue (vault) should go to stepPath; got %v", m.step)
	}
}

// TestNav_DocsMode_EnterActivatesBack asserts Enter on Back goes to stepPlatformSelect.
func TestNav_DocsMode_EnterActivatesBack(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepDocsMode

	// → to Back, then Enter.
	m = sendSpecialKey(t, m, tea.KeyRight)
	m = sendSpecialKey(t, m, tea.KeyEnter)

	if m.step != stepPlatformSelect {
		t.Fatalf("Enter on Back should go to stepPlatformSelect; got %v", m.step)
	}
}

// TestNav_DocsMode_TabIsNoOp asserts Tab does NOT change any state.
func TestNav_DocsMode_TabIsNoOp(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepDocsMode

	beforeBtnFocus := m.docsModeButtons.focus
	beforeCursor := m.modeCursor
	beforeStep := m.step

	m = sendSpecialKey(t, m, tea.KeyTab)

	if m.docsModeButtons.focus != beforeBtnFocus {
		t.Errorf("Tab changed docsModeButtons.focus from %d to %d; no-op expected", beforeBtnFocus, m.docsModeButtons.focus)
	}
	if m.modeCursor != beforeCursor {
		t.Errorf("Tab changed modeCursor from %d to %d; no-op expected", beforeCursor, m.modeCursor)
	}
	if m.step != beforeStep {
		t.Errorf("Tab changed step; no-op expected")
	}
}

// ---------------------------------------------------------------------------
// Overwrite: ↑/↓ move choice, ←/→ move buttons — simultaneously
// ---------------------------------------------------------------------------

// TestNav_Overwrite_UpDownMovesChoice asserts ↑/↓ move overwriteChoice always.
func TestNav_Overwrite_UpDownMovesChoice(t *testing.T) {
	m := newOverwriteModelWithInstalled(t, []string{"opencode"}, []string{"opencode"})

	// Move button focus first.
	m = sendSpecialKey(t, m, tea.KeyRight)
	// ↓ moves overwriteChoice.
	m = sendSpecialKey(t, m, tea.KeyDown)
	if m.overwriteChoice != 1 {
		t.Fatalf("↓ should move overwriteChoice to 1; got %d", m.overwriteChoice)
	}
	// ↑ moves overwriteChoice back.
	m = sendSpecialKey(t, m, tea.KeyUp)
	if m.overwriteChoice != 0 {
		t.Fatalf("↑ should move overwriteChoice to 0; got %d", m.overwriteChoice)
	}
}

// TestNav_Overwrite_LeftRightMoveButtonFocus asserts ←/→ move overwriteButtons focus.
func TestNav_Overwrite_LeftRightMoveButtonFocus(t *testing.T) {
	m := newOverwriteModelWithInstalled(t, []string{"opencode"}, []string{"opencode"})

	if m.overwriteButtons.focus != 0 {
		t.Fatalf("initial overwriteButtons.focus = %d, want 0", m.overwriteButtons.focus)
	}

	m = sendSpecialKey(t, m, tea.KeyRight)
	if m.overwriteButtons.focus != 1 {
		t.Fatalf("→ should move overwriteButtons.focus to 1; got %d", m.overwriteButtons.focus)
	}

	m = sendSpecialKey(t, m, tea.KeyLeft)
	if m.overwriteButtons.focus != 0 {
		t.Fatalf("← should move overwriteButtons.focus to 0; got %d", m.overwriteButtons.focus)
	}
}

// TestNav_Overwrite_EnterActivatesInstall asserts Enter on default [Install] starts progress.
func TestNav_Overwrite_EnterActivatesInstall(t *testing.T) {
	m := newOverwriteModelWithInstalled(t, []string{"opencode"}, []string{"opencode"})
	// Default: Install focused.
	m = sendSpecialKey(t, m, tea.KeyEnter)
	if m.step != stepProgress {
		t.Fatalf("Enter on Install should go to stepProgress; got %v", m.step)
	}
}

// TestNav_Overwrite_EnterActivatesBack asserts Enter on Back returns to path step.
func TestNav_Overwrite_EnterActivatesBack(t *testing.T) {
	m := newOverwriteModelWithInstalled(t, []string{"opencode"}, []string{"opencode"})
	m.mode = ModeVault

	m = sendSpecialKey(t, m, tea.KeyRight) // focus Back
	m = sendSpecialKey(t, m, tea.KeyEnter)

	if m.step != stepPath {
		t.Fatalf("Enter on Back should go to stepPath; got %v", m.step)
	}
}

// TestNav_Overwrite_TabIsNoOp asserts Tab does NOT change any state.
func TestNav_Overwrite_TabIsNoOp(t *testing.T) {
	m := newOverwriteModelWithInstalled(t, []string{"opencode"}, []string{"opencode"})

	beforeBtnFocus := m.overwriteButtons.focus
	beforeChoice := m.overwriteChoice
	beforeStep := m.step

	m = sendSpecialKey(t, m, tea.KeyTab)

	if m.overwriteButtons.focus != beforeBtnFocus {
		t.Errorf("Tab changed overwriteButtons.focus; no-op expected")
	}
	if m.overwriteChoice != beforeChoice {
		t.Errorf("Tab changed overwriteChoice; no-op expected")
	}
	if m.step != beforeStep {
		t.Errorf("Tab changed step; no-op expected")
	}
}

// ---------------------------------------------------------------------------
// Welcome: ←/→ move buttons, Enter activates, ↑/↓ no-op
// ---------------------------------------------------------------------------

// TestNav_Welcome_LeftRightMoveButtonFocus asserts ←/→ move welcomeButtons focus.
func TestNav_Welcome_LeftRightMoveButtonFocus(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModel(AppConfig{}, false, testManifest(), "dist", plats, NoColor())
	m.step = stepWelcome
	m.width = 80
	m.height = 24

	if m.welcomeButtons.focus != 0 {
		t.Fatalf("initial welcomeButtons.focus = %d, want 0", m.welcomeButtons.focus)
	}

	m = sendSpecialKey(t, m, tea.KeyRight)
	if m.welcomeButtons.focus != 1 {
		t.Fatalf("→ should move welcomeButtons.focus to 1; got %d", m.welcomeButtons.focus)
	}

	m = sendSpecialKey(t, m, tea.KeyLeft)
	if m.welcomeButtons.focus != 0 {
		t.Fatalf("← should move welcomeButtons.focus to 0; got %d", m.welcomeButtons.focus)
	}
}

// TestNav_Welcome_EnterActivatesContinue asserts Enter on Continue (default focus=0)
// advances to stepPlatformSelect.
func TestNav_Welcome_EnterActivatesContinue(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModel(AppConfig{}, false, testManifest(), "dist", plats, NoColor())
	m.step = stepWelcome
	m.width = 80
	m.height = 24

	m = sendSpecialKey(t, m, tea.KeyEnter)
	if m.step != stepPlatformSelect {
		t.Fatalf("Enter on Continue should go to stepPlatformSelect; got %v", m.step)
	}
}

// TestNav_Welcome_UpDownNoOp asserts ↑/↓ do not change state on the welcome screen.
func TestNav_Welcome_UpDownNoOp(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModel(AppConfig{}, false, testManifest(), "dist", plats, NoColor())
	m.step = stepWelcome
	m.width = 80
	m.height = 24

	beforeFocus := m.welcomeButtons.focus
	beforeStep := m.step

	m = sendSpecialKey(t, m, tea.KeyDown)
	m = sendSpecialKey(t, m, tea.KeyUp)

	if m.welcomeButtons.focus != beforeFocus {
		t.Errorf("↑/↓ changed welcomeButtons.focus; no-op expected")
	}
	if m.step != beforeStep {
		t.Errorf("↑/↓ changed step on welcome; no-op expected")
	}
}

// ---------------------------------------------------------------------------
// Path screen EXCEPTION: textinput owns keys, ←/→ do NOT move button focus
// ---------------------------------------------------------------------------

// TestNav_Path_LeftRightDoNotMoveButtonFocus asserts that ←/→ on the path screen
// do NOT change pathButtons focus (they edit the textinput cursor instead).
func TestNav_Path_LeftRightDoNotMoveButtonFocus(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepPath

	beforeFocus := m.pathButtons.focus

	m = sendSpecialKey(t, m, tea.KeyRight)
	if m.pathButtons.focus != beforeFocus {
		t.Errorf("→ on path screen changed pathButtons.focus from %d to %d; must not change",
			beforeFocus, m.pathButtons.focus)
	}

	m = sendSpecialKey(t, m, tea.KeyLeft)
	if m.pathButtons.focus != beforeFocus {
		t.Errorf("← on path screen changed pathButtons.focus from %d to %d; must not change",
			beforeFocus, m.pathButtons.focus)
	}
}

// TestNav_Path_EnterWithPathAdvances asserts Enter with a non-empty path advances.
func TestNav_Path_EnterWithPathAdvances(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepPath
	m.alreadyInstalled = map[string]bool{} // no installed → goes to stepProgress

	// Type a path.
	m = typeIntoInput(t, m, "/vault/docs/")

	m = sendSpecialKey(t, m, tea.KeyEnter)
	if m.step != stepProgress {
		t.Fatalf("Enter on path with non-empty path should advance; got %v", m.step)
	}
}

// TestNav_Path_EscReturnsToDocsMode asserts Esc from path returns to stepDocsMode.
func TestNav_Path_EscReturnsToDocsMode(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepPath

	m = sendKey(t, m, "esc")
	if m.step != stepDocsMode {
		t.Fatalf("Esc from path should go to stepDocsMode; got %v", m.step)
	}
}

// TestNav_Path_TypingInsertsIntoInput asserts printable characters type into the path.
func TestNav_Path_TypingInsertsIntoInput(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepPath
	m.pathInput.SetValue("")

	m = sendKey(t, m, "x")
	if !strings.Contains(m.pathInput.Value(), "x") {
		t.Errorf("typing 'x' should insert into path input; value = %q", m.pathInput.Value())
	}
}

// TestNav_Path_TypingB_InsertsAndStays specifically guards that 'b' types into
// the path (no longer a Back shortcut) — vault paths routinely contain 'b'
// (e.g. C:\Users\bob\docs). Back on the path screen is Esc only.
func TestNav_Path_TypingB_InsertsAndStays(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepPath
	m.pathInput.SetValue("")

	m = sendKey(t, m, "b")
	if m.step != stepPath {
		t.Fatalf("typing 'b' must stay on stepPath, got %v", m.step)
	}
	if !strings.Contains(m.pathInput.Value(), "b") {
		t.Errorf("typing 'b' should insert into path input; value = %q", m.pathInput.Value())
	}
}

// ---------------------------------------------------------------------------
// View hint strings: must not contain "Tab" on content+button screens
// ---------------------------------------------------------------------------

// TestNav_HintStrings_PlatformSelect_NoTab asserts the platform-select hint
// does NOT reference "Tab" and DOES reference the new navigation model.
func TestNav_HintStrings_PlatformSelect_NoTab(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)

	view := m.View()
	if strings.Contains(view, "Tab") {
		t.Errorf("platform-select hint must not mention 'Tab' in new nav model; got:\n%s", view)
	}
	// Must mention ←/→ for buttons.
	if !strings.Contains(view, "←") || !strings.Contains(view, "→") {
		t.Errorf("platform-select hint should mention ←/→ for button navigation; got:\n%s", view)
	}
}

// TestNav_HintStrings_DocsMode_NoTab asserts the docs-mode hint has no "Tab".
func TestNav_HintStrings_DocsMode_NoTab(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepDocsMode

	view := m.View()
	if strings.Contains(view, "Tab") {
		t.Errorf("docs-mode hint must not mention 'Tab'; got:\n%s", view)
	}
}

// TestNav_HintStrings_Overwrite_NoTab asserts the overwrite hint has no "Tab".
func TestNav_HintStrings_Overwrite_NoTab(t *testing.T) {
	m := newOverwriteModelWithInstalled(t, []string{"opencode"}, []string{"opencode"})

	view := m.View()
	if strings.Contains(view, "Tab") {
		t.Errorf("overwrite hint must not mention 'Tab'; got:\n%s", view)
	}
}

// TestNav_HintStrings_Path_NoTab asserts the path screen hint has no "Tab".
func TestNav_HintStrings_Path_NoTab(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.step = stepPath

	view := m.View()
	if strings.Contains(view, "Tab") {
		t.Errorf("path screen hint must not mention 'Tab'; got:\n%s", view)
	}
}
