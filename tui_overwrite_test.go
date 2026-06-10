package main

// ---------------------------------------------------------------------------
// Slice 4 — Consolidated overwrite screen tests (TDD RED first)
// ---------------------------------------------------------------------------
//
// ADR-5: Replace the per-platform queue (stepOverwriteConfirm, buildOverwriteQueue,
// advanceOverwriteQueue) with ONE consolidated stepOverwrite screen.
//
// Scenarios:
//   - Screen shown when at least one selected platform is already installed.
//   - Screen skipped (→ stepProgress) when no selected platform is installed.
//   - overwrite-all choice: Overwrite[id]=true for every selected platform.
//   - install-only-missing choice: already-installed excluded from plan.Platforms;
//     when ALL selected are already installed, plan.Platforms == []string{} (not nil)
//     and the wizard routes to stepDone with a "nothing to do" summary.
//   - [Back] from stepOverwrite → prevStep (stepPath for vault, stepDocsMode for in-project).
//   - [Install] → stepProgress.
//   - focusZone works: Tab cycles list↔buttons; left/right moves button focus.
//   - prevStep(stepOverwrite) == stepPath when mode==ModeVault.
//   - prevStep(stepOverwrite) == stepDocsMode when mode==ModeInProject (slice 5 uses this).

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

// ---------------------------------------------------------------------------
// Helpers for this test file
// ---------------------------------------------------------------------------

// newOverwriteModelWithInstalled creates a model at stepOverwrite with platforms
// where the given installedIDs are marked in alreadyInstalled.
func newOverwriteModelWithInstalled(t *testing.T, allIDs []string, installedIDs []string) InstallModel {
	t.Helper()
	dir := t.TempDir()
	var plats []Platform
	for _, id := range allIDs {
		plats = append(plats, newPlatformForTest(t, id, dir+"/"+id))
	}

	installed := make(map[string]bool, len(installedIDs))
	for _, id := range installedIDs {
		installed[id] = true
	}

	m := newInstallModel(AppConfig{}, false, testManifest(), "dist", plats, NoColor())
	// Override the computed alreadyInstalled map with our test fixture.
	m.alreadyInstalled = installed
	// Start on stepOverwrite with all platforms selected.
	m.step = stepOverwrite
	// overwriteChoice=0 = overwrite-all (the default)
	m.overwriteChoice = 0
	// Default button focus=0 = [Install].
	m.overwriteButtons.focus = 0
	m.width = 80
	m.height = 24
	return m
}

// ---------------------------------------------------------------------------
// T4.1 — stepOverwrite appears when some selected platforms already installed
// ---------------------------------------------------------------------------

// TestOverwrite_ScreenShown_WhenSomeInstalled asserts that after docs-mode
// [Continue], if any selected platform is in alreadyInstalled, the wizard
// routes to stepOverwrite (not stepProgress directly).
func TestOverwrite_ScreenShown_WhenSomeInstalled(t *testing.T) {
	dir := t.TempDir()
	plat := newPlatformForTest(t, "opencode", dir+"/opencode")

	m := newInstallModelForTest(AppConfig{}, false, []Platform{plat})
	// Inject alreadyInstalled directly (avoids filesystem dependency).
	m.alreadyInstalled = map[string]bool{"opencode": true}
	// Go to docs-mode step with vault selected, then route Continue → overwrite.
	m.step = stepDocsMode
	m.modeCursor = 0            // vault
	m.docsModeButtons.focus = 0 // Continue (default)

	// Enter = Continue → stepPath (vault).
	m = sendSpecialKey(t, m, tea.KeyEnter)
	if m.step != stepPath {
		t.Fatalf("want stepPath after docs-mode Continue (vault), got %v", m.step)
	}

	// Path entry: type a path and Enter = Continue → should route to stepOverwrite.
	for _, ch := range "/vault/docs/" {
		m = sendKey(t, m, string(ch))
	}
	m = sendSpecialKey(t, m, tea.KeyEnter)

	if m.step != stepOverwrite {
		t.Fatalf("want stepOverwrite after path Continue when platform already installed, got %v", m.step)
	}
}

// TestOverwrite_ScreenSkipped_WhenNoInstalled asserts that if NO selected
// platform is already installed the wizard skips stepOverwrite and goes to stepProgress.
func TestOverwrite_ScreenSkipped_WhenNoInstalled(t *testing.T) {
	dir := t.TempDir()
	plat := newPlatformForTest(t, "opencode", dir+"/opencode")

	m := newInstallModelForTest(AppConfig{}, false, []Platform{plat})
	m.alreadyInstalled = map[string]bool{} // nothing installed
	m.step = stepDocsMode
	m.modeCursor = 0            // vault
	m.docsModeButtons.focus = 0 // Continue (default)

	m = sendSpecialKey(t, m, tea.KeyEnter) // → stepPath
	if m.step != stepPath {
		t.Fatalf("want stepPath, got %v", m.step)
	}

	// Type path + Enter.
	for _, ch := range "/vault/docs/" {
		m = sendKey(t, m, string(ch))
	}
	m = sendSpecialKey(t, m, tea.KeyEnter)

	// No installed platforms → should go straight to stepProgress.
	if m.step != stepProgress {
		t.Fatalf("want stepProgress (overwrite skipped) for fresh install, got %v", m.step)
	}
}

// Also test the in-project path skips stepOverwrite when nothing installed.
func TestOverwrite_ScreenSkipped_InProject_WhenNoInstalled(t *testing.T) {
	dir := t.TempDir()
	plat := newPlatformForTest(t, "opencode", dir+"/opencode")

	m := newInstallModelForTest(AppConfig{}, false, []Platform{plat})
	m.alreadyInstalled = map[string]bool{}
	m.step = stepDocsMode
	m.modeCursor = 1            // in-project
	m.docsModeButtons.focus = 0 // Continue (default)

	m = sendSpecialKey(t, m, tea.KeyEnter) // Continue in-project → should go to stepProgress

	if m.step != stepProgress {
		t.Fatalf("want stepProgress (overwrite skipped for fresh in-project), got %v", m.step)
	}
}

// TestOverwrite_ScreenShown_InProject_WhenInstalled asserts that in-project mode
// ALSO routes to stepOverwrite if a selected platform is already installed.
func TestOverwrite_ScreenShown_InProject_WhenInstalled(t *testing.T) {
	dir := t.TempDir()
	plat := newPlatformForTest(t, "opencode", dir+"/opencode")

	m := newInstallModelForTest(AppConfig{}, false, []Platform{plat})
	m.alreadyInstalled = map[string]bool{"opencode": true}
	m.step = stepDocsMode
	m.modeCursor = 1            // in-project
	m.docsModeButtons.focus = 0 // Continue (default)

	m = sendSpecialKey(t, m, tea.KeyEnter) // Continue in-project

	if m.step != stepOverwrite {
		t.Fatalf("want stepOverwrite for in-project with installed platform, got %v", m.step)
	}
}

// ---------------------------------------------------------------------------
// T4.2 — overwrite-all populates Overwrite map
// ---------------------------------------------------------------------------

// TestOverwrite_OverwriteAll_PopulatesOverwriteMap verifies that when overwriteChoice=0
// (overwrite-all) and [Install] is activated, plan.Overwrite has true for every
// selected platform and stepProgress is entered.
func TestOverwrite_OverwriteAll_PopulatesOverwriteMap(t *testing.T) {
	m := newOverwriteModelWithInstalled(t, []string{"opencode", "claude"}, []string{"opencode", "claude"})
	m.overwriteChoice = 0 // overwrite-all
	// [Install] is default focus (index 0) — Enter activates it.
	m = sendSpecialKey(t, m, tea.KeyEnter)

	if m.step != stepProgress {
		t.Fatalf("want stepProgress after [Install], got %v", m.step)
	}

	plan := m.BuildPlan()
	if !plan.Overwrite["opencode"] {
		t.Errorf("plan.Overwrite[\"opencode\"] = false; want true (overwrite-all)")
	}
	if !plan.Overwrite["claude"] {
		t.Errorf("plan.Overwrite[\"claude\"] = false; want true (overwrite-all)")
	}
}

// TestOverwrite_OverwriteAll_PartiallyInstalled verifies that overwrite-all sets
// Overwrite[id]=true for ALL selected platforms, including those not in alreadyInstalled.
func TestOverwrite_OverwriteAll_PartiallyInstalled(t *testing.T) {
	// opencode installed, claude fresh
	m := newOverwriteModelWithInstalled(t, []string{"opencode", "claude"}, []string{"opencode"})
	m.overwriteChoice = 0
	// [Install] is default focus — Enter activates it.
	m = sendSpecialKey(t, m, tea.KeyEnter)

	if m.step != stepProgress {
		t.Fatalf("want stepProgress, got %v", m.step)
	}
	plan := m.BuildPlan()
	if !plan.Overwrite["opencode"] {
		t.Errorf("plan.Overwrite[\"opencode\"] = false; want true")
	}
	// For overwrite-all, all selected get Overwrite=true.
	if !plan.Overwrite["claude"] {
		t.Errorf("plan.Overwrite[\"claude\"] = false; want true (overwrite-all sets all)")
	}
}

// ---------------------------------------------------------------------------
// T4.3 — install-only-missing excludes installed platforms from plan.Platforms
// ---------------------------------------------------------------------------

// TestOverwrite_InstallOnlyMissing_ExcludesInstalled verifies that choosing
// install-only-missing (overwriteChoice=1) removes already-installed platforms
// from plan.Platforms.
func TestOverwrite_InstallOnlyMissing_ExcludesInstalled(t *testing.T) {
	// opencode installed, claude fresh
	m := newOverwriteModelWithInstalled(t, []string{"opencode", "claude"}, []string{"opencode"})
	m.overwriteChoice = 1        // install-only-missing
	m.overwriteButtons.focus = 0 // [Install]
	// Enter activates [Install].
	m = sendSpecialKey(t, m, tea.KeyEnter)

	if m.step != stepProgress {
		t.Fatalf("want stepProgress after [Install], got %v", m.step)
	}
	plan := m.BuildPlan()
	for _, id := range plan.Platforms {
		if id == "opencode" {
			t.Errorf("plan.Platforms must not contain 'opencode' (already installed, install-only-missing)")
		}
	}
	// claude must be in the plan.
	found := false
	for _, id := range plan.Platforms {
		if id == "claude" {
			found = true
		}
	}
	if !found {
		t.Errorf("plan.Platforms must contain 'claude' (not already installed)")
	}
	// Overwrite map must NOT contain opencode.
	if plan.Overwrite["opencode"] {
		t.Errorf("plan.Overwrite[\"opencode\"] should not be true for install-only-missing")
	}
}

// ---------------------------------------------------------------------------
// T4.4 — install-only-missing all-already-present → []string{} (not nil) → Done
// ---------------------------------------------------------------------------

// TestOverwrite_InstallOnlyMissing_AllPresent_EmptyPlatforms is the CRITICAL guard:
// when ALL selected platforms are already installed and the user picks
// install-only-missing, plan.Platforms must be []string{} (non-nil, empty),
// and the wizard must route to stepDone with a "nothing to do" summary — NEVER
// let nil reach executeInstall (the engine reads nil as "all detected").
func TestOverwrite_InstallOnlyMissing_AllPresent_EmptyPlatforms(t *testing.T) {
	// Both platforms already installed.
	m := newOverwriteModelWithInstalled(t, []string{"opencode", "claude"}, []string{"opencode", "claude"})
	m.overwriteChoice = 1        // install-only-missing
	m.overwriteButtons.focus = 0 // [Install]
	m = sendSpecialKey(t, m, tea.KeyEnter)

	// Must route to stepDone (nothing to install), NOT stepProgress.
	if m.step != stepDone {
		t.Fatalf("want stepDone when all-already-installed + install-only-missing, got %v", m.step)
	}

	plan := m.BuildPlan()
	// CRITICAL: Platforms must be non-nil and empty — never nil.
	if plan.Platforms == nil {
		t.Fatal("plan.Platforms is nil; it MUST be []string{} (empty non-nil) to prevent engine reading nil as 'all detected'")
	}
	if len(plan.Platforms) != 0 {
		t.Errorf("plan.Platforms = %v; want empty []string{} when all already installed", plan.Platforms)
	}

	// The done summary must convey "nothing to do".
	view := m.View()
	if !strings.Contains(view, "already") && !strings.Contains(view, "nothing") && !strings.Contains(view, "Nothing") {
		t.Errorf("done view should mention already-installed / nothing to do; got:\n%s", view)
	}
}

// ---------------------------------------------------------------------------
// T4.5 — [Back] from stepOverwrite returns to correct previous step
// ---------------------------------------------------------------------------

// TestOverwrite_Back_VaultMode_ReturnsToPrevStep verifies [Back] from stepOverwrite
// goes to prevStep(stepOverwrite) which equals stepPath when mode==ModeVault.
func TestOverwrite_Back_VaultMode_ReturnsToPrevStep(t *testing.T) {
	m := newOverwriteModelWithInstalled(t, []string{"opencode"}, []string{"opencode"})
	m.mode = ModeVault

	// → moves button focus to Back (index 1), Enter activates it.
	m = sendSpecialKey(t, m, tea.KeyRight)
	m = sendSpecialKey(t, m, tea.KeyEnter)

	if m.step != prevStep(stepOverwrite) {
		t.Fatalf("want %v (prevStep(stepOverwrite)) after Back, got %v", prevStep(stepOverwrite), m.step)
	}
}

// TestOverwrite_Back_InProjectMode_ReturnsToDocsMode verifies [Back] from
// stepOverwrite goes to stepDocsMode when mode==ModeInProject.
func TestOverwrite_Back_InProjectMode_ReturnsToDocsMode(t *testing.T) {
	m := newOverwriteModelWithInstalled(t, []string{"opencode"}, []string{"opencode"})
	m.mode = ModeInProject

	m = sendSpecialKey(t, m, tea.KeyRight)
	m = sendSpecialKey(t, m, tea.KeyEnter)

	if m.step != stepDocsMode {
		t.Fatalf("want stepDocsMode after Back (in-project), got %v", m.step)
	}
}

// ---------------------------------------------------------------------------
// T4.6 — ←/→ move buttons; ↑/↓ move radio choice (both axes always live)
// ---------------------------------------------------------------------------

// TestOverwrite_RightArrow_MovesFocusToBack verifies → moves button focus to Back.
func TestOverwrite_RightArrow_MovesFocusToBack(t *testing.T) {
	m := newOverwriteModelWithInstalled(t, []string{"opencode"}, []string{"opencode"})

	m = sendSpecialKey(t, m, tea.KeyRight)
	if m.overwriteButtons.focus != 1 {
		t.Errorf("→ should move overwriteButtons.focus to 1 (Back); got %d", m.overwriteButtons.focus)
	}
}

// TestOverwrite_LeftArrow_MovesFocusToInstall verifies ← moves button focus back to Install.
func TestOverwrite_LeftArrow_MovesFocusToInstall(t *testing.T) {
	m := newOverwriteModelWithInstalled(t, []string{"opencode"}, []string{"opencode"})
	m.overwriteButtons.focus = 1 // start at Back

	m = sendSpecialKey(t, m, tea.KeyLeft)
	if m.overwriteButtons.focus != 0 {
		t.Errorf("← should move overwriteButtons.focus to 0 (Install); got %d", m.overwriteButtons.focus)
	}
}

// TestOverwrite_RadioChoice_DownMovesToInstallOnly verifies down/j moves
// overwriteChoice from 0 (overwrite-all) to 1 (install-only-missing).
// ↑/↓ always work regardless of button focus position.
func TestOverwrite_RadioChoice_DownMovesToInstallOnly(t *testing.T) {
	m := newOverwriteModelWithInstalled(t, []string{"opencode"}, []string{"opencode"})
	m.overwriteChoice = 0

	m = sendKey(t, m, "j")
	if m.overwriteChoice != 1 {
		t.Errorf("overwriteChoice = %d, want 1 after j", m.overwriteChoice)
	}
}

// TestOverwrite_RadioChoice_UpClampsAtZero verifies up/k does not go below 0.
func TestOverwrite_RadioChoice_UpClampsAtZero(t *testing.T) {
	m := newOverwriteModelWithInstalled(t, []string{"opencode"}, []string{"opencode"})
	m.overwriteChoice = 0

	m = sendKey(t, m, "k")
	if m.overwriteChoice != 0 {
		t.Errorf("overwriteChoice = %d, want 0 (clamped)", m.overwriteChoice)
	}
}

// ---------------------------------------------------------------------------
// T4.7 — prevStep(stepOverwrite) is stepPath (vault default)
// ---------------------------------------------------------------------------

// TestPrevStep_OverwriteIsPath verifies the pure step helper.
func TestPrevStep_OverwriteIsPath(t *testing.T) {
	got := prevStep(stepOverwrite)
	if got != stepPath {
		t.Errorf("prevStep(stepOverwrite) = %v, want stepPath", got)
	}
}

// TestNextStep_PathIsOverwrite verifies nextStep(stepPath) produces the right chain.
// Note: next step from path depends on whether overwrite is needed, but the pure
// helper must reflect the design chain.
func TestNextStep_PathToOverwrite(t *testing.T) {
	// nextStep table: Path -> Overwrite (default +1 in the new enum).
	got := nextStep(stepPath)
	if got != stepOverwrite {
		t.Errorf("nextStep(stepPath) = %v, want stepOverwrite", got)
	}
}

// ---------------------------------------------------------------------------
// T4.8 — View contains expected elements
// ---------------------------------------------------------------------------

// TestOverwrite_View_ContainsInstalledList verifies the overwrite screen lists the
// already-installed platforms.
func TestOverwrite_View_ContainsInstalledList(t *testing.T) {
	m := newOverwriteModelWithInstalled(t, []string{"opencode", "claude"}, []string{"opencode"})

	view := m.View()
	if !strings.Contains(view, "opencode") {
		t.Errorf("overwrite view missing 'opencode' in installed list; got:\n%s", view)
	}
	// Should mention that 1 platform is already installed.
	if !strings.Contains(view, "already") {
		t.Errorf("overwrite view should mention 'already'; got:\n%s", view)
	}
}

// TestOverwrite_View_ContainsBothChoices verifies the view shows both radio options.
func TestOverwrite_View_ContainsBothChoices(t *testing.T) {
	m := newOverwriteModelWithInstalled(t, []string{"opencode"}, []string{"opencode"})

	view := m.View()
	if !strings.Contains(view, "Overwrite all") {
		t.Errorf("overwrite view missing 'Overwrite all'; got:\n%s", view)
	}
	if !strings.Contains(view, "Install only missing") {
		t.Errorf("overwrite view missing 'Install only missing'; got:\n%s", view)
	}
}

// TestOverwrite_View_ContainsInstallAndBackButtons verifies the footer has
// [Install] and Back.
func TestOverwrite_View_ContainsInstallAndBackButtons(t *testing.T) {
	m := newOverwriteModelWithInstalled(t, []string{"opencode"}, []string{"opencode"})

	view := m.View()
	if !strings.Contains(view, "Install") {
		t.Errorf("overwrite view missing 'Install' button; got:\n%s", view)
	}
	if !strings.Contains(view, "Back") {
		t.Errorf("overwrite view missing 'Back' button; got:\n%s", view)
	}
}

// ---------------------------------------------------------------------------
// T4.9 — [Install] → stepProgress (unit style, avoids engine goroutine in teardown)
// ---------------------------------------------------------------------------

// TestTUIFlow_Overwrite_InstallAdvancesToProgress verifies that activating
// [Install] on the overwrite screen transitions the model to stepProgress.
// Uses direct Update() calls (not teatest) to avoid the engine goroutine
// running during teardown.
func TestTUIFlow_Overwrite_InstallAdvancesToProgress(t *testing.T) {
	m := newOverwriteModelWithInstalled(t, []string{"opencode"}, []string{"opencode"})
	m.overwriteChoice = 0        // overwrite-all
	m.overwriteButtons.focus = 0 // [Install] is default focus

	// Enter → [Install] → commitOverwriteAndInstall → stepProgress.
	result, _ := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	final, ok := result.(InstallModel)
	if !ok {
		t.Fatalf("Update returned %T, want InstallModel", result)
	}
	if final.step != stepProgress {
		t.Fatalf("want stepProgress after [Install] on overwrite screen, got %v", final.step)
	}
}

// TestTUIFlow_Overwrite_BackReturnsToPath exercises the Back button from
// stepOverwrite via teatest (goes to stepPath, engine never runs).
func TestTUIFlow_Overwrite_BackReturnsToPath(t *testing.T) {
	m := newOverwriteModelWithInstalled(t, []string{"opencode"}, []string{"opencode"})
	m.mode = ModeVault

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	// → moves to Back (index 1), Enter activates it.
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyRight}))
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	// Ctrl+C to quit from stepPath.
	tm.Send(tea.KeyMsg(tea.Key{Type: tea.KeyCtrlC}))

	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
	final := tm.FinalModel(t).(InstallModel)

	if final.step != stepPath {
		t.Fatalf("want stepPath after Back from overwrite (vault mode), got %v", final.step)
	}
}
