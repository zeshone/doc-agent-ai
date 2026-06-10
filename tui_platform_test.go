package main

// ---------------------------------------------------------------------------
// Slice 2 unit tests — TDD RED first
// ---------------------------------------------------------------------------
//
// Tests for:
//   - prevStep / nextStep pure helpers (Step transition table).
//   - focusZone: Tab cycles content ↔ buttons; Enter on list vs buttons.
//   - Platform list: already-installed marks render; checkbox toggle.
//   - Zero-selected guard: [Continue] blocked.
//   - [Back] → Welcome (prevStep) with state preserved.
//   - [Continue] → stepDocsMode (nextStep).
//   - focusZone movement: Tab on list zone → button zone, Tab on buttons → list.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// ---------------------------------------------------------------------------
// prevStep / nextStep table tests
// ---------------------------------------------------------------------------

// TestPrevStep_PlatformSelect_ReturnsWelcome verifies that BACK from
// stepPlatformSelect always goes to stepWelcome, regardless of mode.
func TestPrevStep_PlatformSelect_ReturnsWelcome(t *testing.T) {
	got := prevStep(stepPlatformSelect)
	if got != stepWelcome {
		t.Fatalf("prevStep(stepPlatformSelect) = %v, want stepWelcome", got)
	}
}

// TestPrevStep_DocsMode_ReturnsPlatformSelect verifies BACK from stepDocsMode.
func TestPrevStep_DocsMode_ReturnsPlatformSelect(t *testing.T) {
	got := prevStep(stepDocsMode)
	if got != stepPlatformSelect {
		t.Fatalf("prevStep(stepDocsMode) = %v, want stepPlatformSelect", got)
	}
}

// TestNextStep_PlatformSelect_ReturnsDocsMode verifies Continue from
// stepPlatformSelect advances to stepDocsMode.
func TestNextStep_PlatformSelect_ReturnsDocsMode(t *testing.T) {
	got := nextStep(stepPlatformSelect)
	if got != stepDocsMode {
		t.Fatalf("nextStep(stepPlatformSelect) = %v, want stepDocsMode", got)
	}
}

// TestNextStep_Welcome_ReturnsPlatformSelect verifies Continue from welcome.
func TestNextStep_Welcome_ReturnsPlatformSelect(t *testing.T) {
	got := nextStep(stepWelcome)
	if got != stepPlatformSelect {
		t.Fatalf("nextStep(stepWelcome) = %v, want stepPlatformSelect", got)
	}
}

// ---------------------------------------------------------------------------
// Button focus movement via ←/→ (no Tab)
// ---------------------------------------------------------------------------

// TestPlatformSelect_RightArrow_MovesFocusToBack verifies that → moves button
// focus from Continue (0) to Back (1) on the platform-select screen.
func TestPlatformSelect_RightArrow_MovesFocusToBack(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)

	m = sendSpecialKey(t, m, tea.KeyRight)
	if m.platformSelectButtons.focus != 1 {
		t.Fatalf("→ should move button focus to 1 (Back); got %d", m.platformSelectButtons.focus)
	}
}

// TestPlatformSelect_LeftArrow_MovesFocusToContinue verifies that ← moves button
// focus from Back (1) back to Continue (0).
func TestPlatformSelect_LeftArrow_MovesFocusToContinue(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.platformSelectButtons.focus = 1 // start at Back

	m = sendSpecialKey(t, m, tea.KeyLeft)
	if m.platformSelectButtons.focus != 0 {
		t.Fatalf("← should move button focus to 0 (Continue); got %d", m.platformSelectButtons.focus)
	}
}

// ---------------------------------------------------------------------------
// Platform list — already-installed marks
// ---------------------------------------------------------------------------

// TestPlatformSelect_AlreadyInstalled_ShowsMark verifies that a platform with
// an existing install is visually marked in the platform-select view.
func TestPlatformSelect_AlreadyInstalled_ShowsMark(t *testing.T) {
	dir := t.TempDir()
	plat := newPlatformForTest(t, "claude", dir+"/claude")

	// Pre-create an agent file so checkAlreadyInstalled finds it.
	agentsDir := plat.AgentsDir()
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("create agentsDir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "doc-arch.md"), []byte("content"), 0644); err != nil {
		t.Fatalf("write dummy agent: %v", err)
	}

	m := newInstallModel(AppConfig{}, false, testManifest(), "dist", []Platform{plat}, NoColor())
	m.manifest.Roles = []DistRole{{ID: "doc-arch"}}
	m.step = stepPlatformSelect
	m.width = 80
	m.height = 24
	// Rebuild alreadyInstalled map with the seeded manifest.
	m.alreadyInstalled = buildAlreadyInstalledMap(m.manifest, []Platform{plat})

	view := m.View()

	// The view must contain an "installed" marker somewhere.
	if !strings.Contains(strings.ToLower(view), "installed") {
		t.Errorf("platform-select view should contain 'installed' marker for already-installed platform;\ngot:\n%s", view)
	}
}

// TestPlatformSelect_FreshInstall_NoInstalledMark verifies that a fresh install
// (no existing agents) does NOT show an already-installed mark.
func TestPlatformSelect_FreshInstall_NoInstalledMark(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	m.manifest.Roles = []DistRole{{ID: "doc-arch"}}
	// alreadyInstalled is empty by default (no pre-existing files).
	view := m.View()

	if strings.Contains(strings.ToLower(view), "already installed") {
		t.Errorf("platform-select view should NOT contain 'already installed' for fresh install;\ngot:\n%s", view)
	}
}

// ---------------------------------------------------------------------------
// Checkbox toggle
// ---------------------------------------------------------------------------

// TestPlatformSelect_SpaceTogglesCheckbox verifies that Space toggles the
// selected state of the focused item (always active, no zone gating).
func TestPlatformSelect_SpaceTogglesCheckbox(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	initial := m.platforms[0].selected

	m = sendKey(t, m, " ")
	if m.platforms[0].selected == initial {
		t.Fatalf("Space should toggle checkbox; selected unchanged (was %v)", initial)
	}
}

// TestPlatformSelect_SpaceTogglesEvenAfterButtonMove verifies that Space toggles
// the checkbox even after ←/→ have moved button focus.
func TestPlatformSelect_SpaceTogglesEvenAfterButtonMove(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	// Move button focus to Back.
	m = sendSpecialKey(t, m, tea.KeyRight)
	initial := m.platforms[0].selected

	m = sendKey(t, m, " ")
	if m.platforms[0].selected == initial {
		t.Fatalf("Space should toggle checkbox even after button move; selected unchanged (was %v)", initial)
	}
}

// ---------------------------------------------------------------------------
// Zero-selected guard
// ---------------------------------------------------------------------------

// TestPlatformSelect_Continue_BlockedWhenNoneSelected verifies that Enter with
// Continue focused does NOT advance when zero platforms are selected.
func TestPlatformSelect_Continue_BlockedWhenNoneSelected(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)

	// Deselect all platforms.
	for i := range m.platforms {
		m.platforms[i].selected = false
	}

	m.platformSelectButtons.focus = 0 // Continue (default)

	// Send Enter — should be blocked.
	m = sendSpecialKey(t, m, tea.KeyEnter)
	if m.step != stepPlatformSelect {
		t.Fatalf("step should stay at stepPlatformSelect when no platforms selected, got %v", m.step)
	}
	// A notice should be set.
	if m.notice == "" {
		t.Error("notice should be set when Continue is blocked due to zero selections")
	}
}

// ---------------------------------------------------------------------------
// [Back] → Welcome
// ---------------------------------------------------------------------------

// TestPlatformSelect_Back_ReturnsToWelcome verifies that activating [Back] from
// stepPlatformSelect returns to stepWelcome (BACK = prevStep).
func TestPlatformSelect_Back_ReturnsToWelcome(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)

	// Move to Back button (index 1) via →, then Enter.
	m.platformSelectButtons.focus = 1 // Back

	m = sendSpecialKey(t, m, tea.KeyEnter)
	if m.step != stepWelcome {
		t.Fatalf("activating Back should go to stepWelcome, got %v", m.step)
	}
}

// TestPlatformSelect_Back_PreservesSelections verifies that selections are
// preserved when the user navigates Back to Welcome and then returns.
func TestPlatformSelect_Back_PreservesSelections(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)

	// Deselect platform 0.
	m.platforms[0].selected = false

	// Go Back to Welcome.
	m.platformSelectButtons.focus = 1 // Back
	m = sendSpecialKey(t, m, tea.KeyEnter)
	if m.step != stepWelcome {
		t.Fatalf("expected stepWelcome after Back, got %v", m.step)
	}

	// Selections must be preserved on the model.
	if m.platforms[0].selected {
		t.Error("platform 0 should still be deselected after going Back")
	}
}

// ---------------------------------------------------------------------------
// [Continue] → stepDocsMode
// ---------------------------------------------------------------------------

// TestPlatformSelect_Continue_AdvancesToDocsMode verifies that activating
// [Continue] with ≥1 platform selected advances to stepDocsMode.
func TestPlatformSelect_Continue_AdvancesToDocsMode(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)
	// At least one platform is selected by default.
	// Continue is default focus (index 0).

	m = sendSpecialKey(t, m, tea.KeyEnter)
	if m.step != stepDocsMode {
		t.Fatalf("activating Continue should go to stepDocsMode, got %v", m.step)
	}
}

// ---------------------------------------------------------------------------
// Compact header on inner screens
// ---------------------------------------------------------------------------

// TestPlatformSelect_View_ContainsCompactHeader verifies that the platform-select
// view starts with the compact Zeen header (not the full welcome lockup).
func TestPlatformSelect_View_ContainsCompactHeader(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)

	view := m.View()
	// The compact header renders "Zeen" on the first line.
	if !strings.Contains(view, "Zeen") {
		t.Errorf("platform-select view should contain compact header 'Zeen';\ngot:\n%s", view)
	}
}

// ---------------------------------------------------------------------------
// Button row rendered in platform-select view
// ---------------------------------------------------------------------------

// TestPlatformSelect_View_ContainsContinueAndBack verifies that the platform-
// select view contains both [Continue] and [Back] in the footer.
func TestPlatformSelect_View_ContainsContinueAndBack(t *testing.T) {
	plats := testPlatformsFromTempDir(t)
	m := newInstallModelForTest(AppConfig{}, false, plats)

	view := m.View()
	if !strings.Contains(view, "Continue") {
		t.Errorf("platform-select view missing 'Continue';\ngot:\n%s", view)
	}
	if !strings.Contains(view, "Back") {
		t.Errorf("platform-select view missing 'Back';\ngot:\n%s", view)
	}
}

// ---------------------------------------------------------------------------
// Golden: platform-select with already-installed marks
// ---------------------------------------------------------------------------

// TestGolden_PlatformSelectWithInstalled captures the platform-select view when
// one platform shows an already-installed mark.
func TestGolden_PlatformSelectWithInstalled(t *testing.T) {
	dir := t.TempDir()
	plat := newPlatformForTest(t, "claude", dir+"/claude")

	// Pre-create an agent file so checkAlreadyInstalled finds it.
	agentsDir := plat.AgentsDir()
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("create agentsDir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "doc-arch.md"), []byte("content"), 0644); err != nil {
		t.Fatalf("write dummy agent: %v", err)
	}

	m := newInstallModel(AppConfig{}, false, testManifest(), "dist", []Platform{plat}, NoColor())
	m.manifest.Roles = []DistRole{{ID: "doc-arch"}}
	m.step = stepPlatformSelect
	m.width = 80
	m.height = 24
	m.alreadyInstalled = buildAlreadyInstalledMap(m.manifest, []Platform{plat})

	view := m.View()
	assertGolden(t, view)
}
