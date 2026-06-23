package main

// ---------------------------------------------------------------------------
// Overwrite semantics tests — headless + engine layer (PRESERVED)
// ---------------------------------------------------------------------------
//
// TUI-wizard overwrite tests are in tui_overwrite_test.go (slice 4, ADR-5).
// The per-platform queue tests (stepOverwriteConfirm, buildOverwriteQueue,
// advanceOverwriteQueue) have been removed — that code is deleted in slice 4.
//
// The following tests cover the headless and engine layer which are UNTOUCHED:
//   - TestHeadless_ExistingInstall_NoYesFlag_Errors
//   - TestHeadless_ExistingInstall_WithYesFlag_Proceeds
//   - TestExecuteInstall_RespectsOverwriteMap
//   - TestExecuteInstall_OverwriteMapTrue_InstallsNormally
//   - TestRunInstall_NoPlatformsSelected_Errors (defense-in-depth TUI guard)

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
