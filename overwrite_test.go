package main

// ---------------------------------------------------------------------------
// C2 — per-platform overwrite confirmation tests (TDD RED → GREEN)
// ---------------------------------------------------------------------------
//
// Spec F1 requires parity with the old bufio flow: when checkAlreadyInstalled
// detects an existing install for a selected platform, the user must be
// prompted (wizard) or errors raised (headless without --yes).
//
// These tests are written RED-first; they will fail until the implementation
// is added to tui_model.go, headless.go, and execute_install.go.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// ---------------------------------------------------------------------------
// Wizard overwrite step
// ---------------------------------------------------------------------------

// TestInstallModel_OverwriteStep_AppearsAfterPlatformSelectWhenAlreadyInstalled
// verifies that when a selected platform has an existing install, the wizard
// transitions to a stepOverwriteConfirm step instead of directly to stepDocsMode.
func TestInstallModel_OverwriteStep_AppearsAfterPlatformSelectWhenAlreadyInstalled(t *testing.T) {
	dir := t.TempDir()
	plat := newPlatformForTest(t, "claude", dir+"/claude")

	// Pre-create an agent file so checkAlreadyInstalled finds something.
	agentsDir := plat.AgentsDir()
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("create agentsDir: %v", err)
	}
	// Write a dummy agent file matching a real role name from the manifest.
	if err := os.WriteFile(filepath.Join(agentsDir, "doc-arch.md"), []byte("content"), 0644); err != nil {
		t.Fatalf("write dummy agent: %v", err)
	}

	m := newInstallModelForTest(AppConfig{}, false, []Platform{plat})
	m.manifest = testManifest()
	// Inject a role into the manifest so checkAlreadyInstalled finds "doc-arch".
	m.manifest.Roles = []DistRole{{ID: "doc-arch"}}

	// Press Enter to confirm platform selection.
	m = sendSpecialKey(t, m, tea.KeyEnter)

	// The step MUST be stepOverwriteConfirm, not stepDocsMode.
	if m.step != stepOverwriteConfirm {
		t.Fatalf("expected stepOverwriteConfirm after enter when platform already installed, got step=%v", m.step)
	}
}

// TestInstallModel_OverwriteStep_SkipsPlatformOnNo verifies that pressing 'n'
// in the overwrite step deselects the platform and does not include it in BuildPlan.
func TestInstallModel_OverwriteStep_SkipsPlatformOnNo(t *testing.T) {
	dir := t.TempDir()
	plat := newPlatformForTest(t, "claude", dir+"/claude")

	agentsDir := plat.AgentsDir()
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("create agentsDir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "doc-arch.md"), []byte("content"), 0644); err != nil {
		t.Fatalf("write dummy agent: %v", err)
	}

	m := newInstallModelForTest(AppConfig{}, false, []Platform{plat})
	m.manifest.Roles = []DistRole{{ID: "doc-arch"}}

	m = sendSpecialKey(t, m, tea.KeyEnter) // confirm platform select
	if m.step != stepOverwriteConfirm {
		t.Fatalf("expected stepOverwriteConfirm, got %v", m.step)
	}

	// 'n' → skip this platform.
	m = sendKey(t, m, "n")

	// Should advance past overwrite step (to docsMode or confirm depending on remaining platforms).
	if m.step == stepOverwriteConfirm {
		t.Fatalf("step should have advanced past stepOverwriteConfirm after 'n'")
	}

	// The plan must NOT include "claude".
	plan := m.BuildPlan()
	for _, id := range plan.Platforms {
		if id == "claude" {
			t.Errorf("plan.Platforms should not contain 'claude' after skipping overwrite")
		}
	}
}

// TestInstallModel_OverwriteStep_ProceedsOnYes verifies that pressing 'y'
// in the overwrite step keeps the platform selected and records it in Overwrite map.
func TestInstallModel_OverwriteStep_ProceedsOnYes(t *testing.T) {
	dir := t.TempDir()
	plat := newPlatformForTest(t, "claude", dir+"/claude")

	agentsDir := plat.AgentsDir()
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("create agentsDir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "doc-arch.md"), []byte("content"), 0644); err != nil {
		t.Fatalf("write dummy agent: %v", err)
	}

	m := newInstallModelForTest(AppConfig{}, false, []Platform{plat})
	m.manifest.Roles = []DistRole{{ID: "doc-arch"}}

	m = sendSpecialKey(t, m, tea.KeyEnter)
	if m.step != stepOverwriteConfirm {
		t.Fatalf("expected stepOverwriteConfirm, got %v", m.step)
	}

	// 'y' → keep this platform.
	m = sendKey(t, m, "y")

	// The step should have advanced.
	if m.step == stepOverwriteConfirm {
		t.Fatalf("step should have advanced past stepOverwriteConfirm after 'y'")
	}

	// overwriteConsent for "claude" must be true.
	if !m.overwriteConsent["claude"] {
		t.Errorf("overwriteConsent[\"claude\"] = false; want true after pressing 'y'")
	}
}

// TestInstallModel_OverwriteStep_NotShownForFreshInstall verifies that the
// overwrite step is skipped when no existing install is detected (fresh install).
func TestInstallModel_OverwriteStep_NotShownForFreshInstall(t *testing.T) {
	// No pre-existing agent files.
	dir := t.TempDir()
	plat := newPlatformForTest(t, "claude", dir+"/claude")

	m := newInstallModelForTest(AppConfig{}, false, []Platform{plat})
	m.manifest.Roles = []DistRole{{ID: "doc-arch"}}

	// No agents dir → nothing installed → no overwrite step.
	m = sendSpecialKey(t, m, tea.KeyEnter)

	if m.step != stepDocsMode {
		t.Fatalf("expected stepDocsMode for fresh install (no overwrite), got %v", m.step)
	}
}

// TestInstallModel_BuildPlan_PopulatesOverwriteMap verifies that BuildPlan sets
// Overwrite map entries for platforms that got overwrite consent.
func TestInstallModel_BuildPlan_PopulatesOverwriteMap(t *testing.T) {
	dir := t.TempDir()
	plat := newPlatformForTest(t, "claude", dir+"/claude")

	agentsDir := plat.AgentsDir()
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("create agentsDir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "doc-arch.md"), []byte("content"), 0644); err != nil {
		t.Fatalf("write dummy agent: %v", err)
	}

	m := newInstallModelForTest(AppConfig{}, false, []Platform{plat})
	m.manifest.Roles = []DistRole{{ID: "doc-arch"}}

	m = sendSpecialKey(t, m, tea.KeyEnter) // platform select → overwrite step
	m = sendKey(t, m, "y")                  // consent to overwrite

	plan := m.BuildPlan()
	if !plan.Overwrite["claude"] {
		t.Errorf("BuildPlan().Overwrite[\"claude\"] = false; want true after overwrite consent")
	}
}

// ---------------------------------------------------------------------------
// Headless overwrite semantics
// ---------------------------------------------------------------------------

// TestHeadless_ExistingInstall_NoYesFlag_Errors verifies that runHeadlessInstall
// returns an error when a platform already has agents installed and --yes is not set.
func TestHeadless_ExistingInstall_NoYesFlag_Errors(t *testing.T) {
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	t.Cleanup(restoreHome)

	distDir := filepath.Join(t.TempDir(), "dist")
	if err := generate(distDir); err != nil {
		t.Fatalf("generate dist: %v", err)
	}

	// Create an opencode home so it is detected.
	opencodeHome := filepath.Join(tmpHome, ".config", "opencode")
	if err := os.MkdirAll(opencodeHome, 0755); err != nil {
		t.Fatalf("create opencode home: %v", err)
	}
	// Seed opencode.json with an existing agent so checkAlreadyInstalled returns results.
	manifest, err := readManifestFrom(distDir)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	existingAgentIDs := make([]string, 0, len(manifest.Roles))
	for _, r := range manifest.Roles {
		existingAgentIDs = append(existingAgentIDs, r.ID)
		if len(existingAgentIDs) == 1 {
			break // one existing agent is enough
		}
	}

	// Write opencode.json with existing agent using the real opencode format:
	// {"agent": {"role-id": {...}}} — this is what GetAgentIDs reads.
	agentMap := make(map[string]any, len(existingAgentIDs))
	for _, id := range existingAgentIDs {
		agentMap[id] = map[string]any{}
	}
	cfgData, _ := json.Marshal(map[string]any{"agent": agentMap})
	if err := os.WriteFile(filepath.Join(opencodeHome, "opencode.json"), cfgData, 0644); err != nil {
		t.Fatalf("write opencode.json: %v", err)
	}

	flags := FlagSet{
		Platforms: "opencode",
		DocsMode:  "vault",
		Path:      "/test/docs/",
		Yes:       false, // NO --yes
	}

	err = runHeadlessInstall(flags, distDir)
	if err == nil {
		t.Fatal("expected error when existing install detected and --yes not provided, got nil")
	}
	// Error must mention "overwrite" or "--yes".
	errMsg := err.Error()
	if !containsAny(errMsg, "overwrite", "--yes", "already installed") {
		t.Errorf("error message %q should mention overwrite or --yes", errMsg)
	}
}

// TestHeadless_ExistingInstall_WithYesFlag_Proceeds verifies that runHeadlessInstall
// succeeds when a platform already has agents installed and --yes is set.
func TestHeadless_ExistingInstall_WithYesFlag_Proceeds(t *testing.T) {
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	t.Cleanup(restoreHome)

	distDir := filepath.Join(t.TempDir(), "dist")
	if err := generate(distDir); err != nil {
		t.Fatalf("generate dist: %v", err)
	}

	opencodeHome := filepath.Join(tmpHome, ".config", "opencode")
	if err := os.MkdirAll(opencodeHome, 0755); err != nil {
		t.Fatalf("create opencode home: %v", err)
	}
	manifest, err := readManifestFrom(distDir)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	// Write opencode.json with one existing agent using the real opencode format.
	var firstID string
	for _, r := range manifest.Roles {
		firstID = r.ID
		break
	}
	cfgData, _ := json.Marshal(map[string]any{"agent": map[string]any{firstID: map[string]any{}}})
	if err := os.WriteFile(filepath.Join(opencodeHome, "opencode.json"), cfgData, 0644); err != nil {
		t.Fatalf("write opencode.json: %v", err)
	}

	flags := FlagSet{
		Platforms: "opencode",
		DocsMode:  "vault",
		Path:      filepath.ToSlash(filepath.Join(tmpHome, "docs")) + "/",
		Yes:       true, // --yes provided
	}

	err = runHeadlessInstall(flags, distDir)
	if err != nil {
		t.Fatalf("runHeadlessInstall with --yes should succeed on existing install, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// executeInstall overwrite map enforcement
// ---------------------------------------------------------------------------

// TestExecuteInstall_RespectsOverwriteMap verifies that executeInstall skips
// platforms that have existing installs but no entry in plan.Overwrite.
func TestExecuteInstall_RespectsOverwriteMap(t *testing.T) {
	tmpHome, distDir, manifest, opencodePlat := setupExecuteInstallFixture(t)

	// Pre-install one agent into opencode so checkAlreadyInstalled finds it.
	// opencode.json uses {"agent": {"role-id": {}}} format.
	opencodeHome := filepath.Join(tmpHome, ".config", "opencode")
	var firstID string
	for _, r := range manifest.Roles {
		firstID = r.ID
		break
	}
	cfgData, _ := json.Marshal(map[string]any{"agent": map[string]any{firstID: map[string]any{}}})
	if err := os.WriteFile(filepath.Join(opencodeHome, "opencode.json"), cfgData, 0644); err != nil {
		t.Fatalf("write opencode.json: %v", err)
	}

	basePath := filepath.ToSlash(filepath.Join(tmpHome, "docs")) + "/"
	plan := InstallPlan{
		Platforms: []string{"opencode"},
		Mode:      ModeVault,
		BasePath:  basePath,
		PrevMode:  ModeVault,
		Yes:       false,
		// Overwrite map is empty → opencode has existing install but no consent.
		Overwrite: map[string]bool{},
	}

	r := newBufferReporter()
	err := executeInstall(manifest, plan, distDir, []Platform{opencodePlat}, r)
	if err == nil {
		t.Fatal("expected error when existing install found and Overwrite map has no consent for the platform")
	}
}

// TestExecuteInstall_OverwriteMapTrue_InstallsNormally verifies that when
// Overwrite["opencode"] = true, executeInstall proceeds normally.
func TestExecuteInstall_OverwriteMapTrue_InstallsNormally(t *testing.T) {
	tmpHome, distDir, manifest, opencodePlat := setupExecuteInstallFixture(t)

	opencodeHome := filepath.Join(tmpHome, ".config", "opencode")
	var firstID string
	for _, r := range manifest.Roles {
		firstID = r.ID
		break
	}
	cfgData, _ := json.Marshal(map[string]any{"agent": map[string]any{firstID: map[string]any{}}})
	if err := os.WriteFile(filepath.Join(opencodeHome, "opencode.json"), cfgData, 0644); err != nil {
		t.Fatalf("write opencode.json: %v", err)
	}

	basePath := filepath.ToSlash(filepath.Join(tmpHome, "docs")) + "/"
	plan := InstallPlan{
		Platforms: []string{"opencode"},
		Mode:      ModeVault,
		BasePath:  basePath,
		PrevMode:  ModeVault,
		Yes:       false,
		Overwrite: map[string]bool{"opencode": true},
	}

	r := newBufferReporter()
	err := executeInstall(manifest, plan, distDir, []Platform{opencodePlat}, r)
	if err != nil {
		t.Fatalf("executeInstall should succeed when Overwrite[platform]=true; got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

// containsAny reports whether s contains any of the given substrings.
func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
