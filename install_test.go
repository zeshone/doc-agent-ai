package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Legacy command sweep tests (T-02)
// ---------------------------------------------------------------------------

// legacyIDs is the canonical list of bare command names retired in v4.0.0.
var legacyIDs = []string{"arch", "idea", "rec", "prd", "refine", "tech", "pti", "mod", "feat", "ddd", "to-sdd"}

// setupOpencodeForSweep creates an opencode platform + home dir inside tmpHome.
func setupOpencodeForSweep(t *testing.T, tmpHome string) (Platform, string) {
	t.Helper()
	opencodeHome := filepath.Join(tmpHome, ".config", "opencode")
	if err := os.MkdirAll(opencodeHome, 0755); err != nil {
		t.Fatalf("create opencode dir: %v", err)
	}
	cfg := map[string]any{}
	data, _ := json.Marshal(cfg)
	if err := os.WriteFile(filepath.Join(opencodeHome, "opencode.json"), data, 0644); err != nil {
		t.Fatalf("write opencode.json: %v", err)
	}
	plat := newPlatformForTest(t, "opencode", opencodeHome)
	return plat, opencodeHome
}

// seedLegacyFiles creates legacy *.md files in the opencode commands dir.
func seedLegacyFiles(t *testing.T, cmdsDir string, ids []string) {
	t.Helper()
	if err := os.MkdirAll(cmdsDir, 0755); err != nil {
		t.Fatalf("create commands dir: %v", err)
	}
	for _, id := range ids {
		if err := os.WriteFile(filepath.Join(cmdsDir, id+".md"), []byte("# legacy "+id), 0644); err != nil {
			t.Fatalf("seed legacy file %s: %v", id, err)
		}
	}
}

// TestInstallSweep_FreshInstall verifies no legacy files appear and the new
// doc-* commands are present after a clean install with no pre-existing files.
func TestInstallSweep_FreshInstall(t *testing.T) {
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	defer restoreHome()

	// Generate a real dist so there are actual command files to copy.
	distDir := filepath.Join(t.TempDir(), "dist")
	if err := generate(distDir); err != nil {
		t.Fatalf("generate dist: %v", err)
	}
	manifest, err := readManifestFrom(distDir)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	plat, opencodeHome := setupOpencodeForSweep(t, tmpHome)
	cmdsDir := filepath.Join(opencodeHome, "commands")
	basePath := filepath.ToSlash(filepath.Join(tmpHome, "projects")) + "/"

	if err := installToPlatform(manifest, plat, basePath, distDir); err != nil {
		t.Fatalf("installToPlatform: %v", err)
	}

	// All new doc-* commands present.
	for _, cmd := range manifest.Commands {
		p := filepath.Join(cmdsDir, cmd.ID+".md")
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Errorf("expected command file %s to exist after fresh install", p)
		}
	}

	// No legacy files present.
	for _, id := range legacyIDs {
		p := filepath.Join(cmdsDir, id+".md")
		if _, err := os.Stat(p); err == nil {
			t.Errorf("legacy file %s should not exist after fresh install, but found", p)
		}
	}
}

// TestInstallSweep_ReinstallOverV3 seeds the commands dir with all 11 legacy
// files (simulating a v3.x install) and verifies they are all deleted after
// a v4 install.
func TestInstallSweep_ReinstallOverV3(t *testing.T) {
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	defer restoreHome()

	distDir := filepath.Join(t.TempDir(), "dist")
	if err := generate(distDir); err != nil {
		t.Fatalf("generate dist: %v", err)
	}
	manifest, err := readManifestFrom(distDir)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	plat, opencodeHome := setupOpencodeForSweep(t, tmpHome)
	cmdsDir := filepath.Join(opencodeHome, "commands")

	// Seed all 11 legacy files.
	seedLegacyFiles(t, cmdsDir, legacyIDs)

	basePath := filepath.ToSlash(filepath.Join(tmpHome, "projects")) + "/"
	if err := installToPlatform(manifest, plat, basePath, distDir); err != nil {
		t.Fatalf("installToPlatform: %v", err)
	}

	// All legacy files must be deleted.
	for _, id := range legacyIDs {
		p := filepath.Join(cmdsDir, id+".md")
		if _, err := os.Stat(p); err == nil {
			t.Errorf("legacy file %s.md still present after reinstall — sweep did not run", id)
		}
	}

	// All new doc-* commands must be present.
	for _, cmd := range manifest.Commands {
		p := filepath.Join(cmdsDir, cmd.ID+".md")
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Errorf("expected new command file %s after reinstall", p)
		}
	}
}

// TestInstallSweep_Idempotent seeds only a partial set of legacy files and
// verifies install removes them with no error for absent ones.
func TestInstallSweep_Idempotent(t *testing.T) {
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	defer restoreHome()

	distDir := filepath.Join(t.TempDir(), "dist")
	if err := generate(distDir); err != nil {
		t.Fatalf("generate dist: %v", err)
	}
	manifest, err := readManifestFrom(distDir)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	plat, opencodeHome := setupOpencodeForSweep(t, tmpHome)
	cmdsDir := filepath.Join(opencodeHome, "commands")

	// Seed only two legacy files.
	seedLegacyFiles(t, cmdsDir, []string{"prd", "arch"})

	basePath := filepath.ToSlash(filepath.Join(tmpHome, "projects")) + "/"
	// Must not return an error even if most legacy files are absent.
	if err := installToPlatform(manifest, plat, basePath, distDir); err != nil {
		t.Fatalf("installToPlatform (idempotent test): %v", err)
	}

	// The two seeded files must be gone.
	for _, id := range []string{"prd", "arch"} {
		p := filepath.Join(cmdsDir, id+".md")
		if _, err := os.Stat(p); err == nil {
			t.Errorf("legacy file %s.md should have been swept, still exists", id)
		}
	}
}

// ---------------------------------------------------------------------------
// 7.5 Install file tree test
// ---------------------------------------------------------------------------

// TestInstallFileTree creates mock platform dirs, generates dist, runs
// non-interactive install, and verifies all expected files exist at
// correct paths per platform (opencode + claude).
func TestInstallFileTree(t *testing.T) {
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	defer restoreHome()

	// Generate dist into a temp location (avoid race with other tests using "dist/")
	distDir := filepath.Join(t.TempDir(), "dist")
	if err := generate(distDir); err != nil {
		t.Fatalf("generate dist: %v", err)
	}

	manifest, err := readManifestFrom(distDir)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	// --- Create mock opencode ---
	opencodeHome := filepath.Join(tmpHome, ".config", "opencode")
	if err := os.MkdirAll(opencodeHome, 0755); err != nil {
		t.Fatalf("create opencode dir: %v", err)
	}
	// Write minimal opencode.json so Detect() passes
	cfg := map[string]any{}
	data, _ := json.Marshal(cfg)
	if err := os.WriteFile(filepath.Join(opencodeHome, "opencode.json"), data, 0644); err != nil {
		t.Fatalf("write opencode.json: %v", err)
	}

	// --- Create mock claude ---
	claudeHome := filepath.Join(tmpHome, ".claude")
	if err := os.MkdirAll(claudeHome, 0755); err != nil {
		t.Fatalf("create claude dir: %v", err)
	}

	// --- Build test platforms ---
	opencode := newPlatformForTest(t, "opencode", opencodeHome)
	claude := newPlatformForTest(t, "claude", claudeHome)

	basePath := filepath.ToSlash(filepath.Join(tmpHome, "projects")) + "/"

	// --- Non-interactive install ---
	if err := installPlatforms(manifest, []Platform{opencode, claude}, basePath, distDir); err != nil {
		t.Fatalf("installPlatforms: %v", err)
	}

	// ======== Verify opencode ========

	// Skills
	for _, skill := range manifest.Skills {
		skillDir := filepath.Join(opencode.SkillsDir(), skill)
		entries, err := os.ReadDir(skillDir)
		if err != nil || len(entries) == 0 {
			t.Errorf("opencode skill %s: no files at %s (err=%v)", skill, skillDir, err)
		}
	}

	// Prompts
	for _, role := range manifest.Roles {
		promptPath := filepath.Join(opencode.PromptsDir(), role.ID+".md")
		if _, err := os.Stat(promptPath); os.IsNotExist(err) {
			t.Errorf("opencode prompt %s missing: %s", role.ID, promptPath)
		}
		// Verify placeholder replaced (only when placeholder was present —
		// doc-arch does not use BASE_PATH in its content)
		content, _ := os.ReadFile(promptPath)
		if strings.Contains(string(content), manifest.PlaceholderBasePath) {
			t.Errorf("opencode prompt %s: placeholder not replaced (%s still present)", role.ID, manifest.PlaceholderBasePath)
		}
	}

	// Commands
	cmdsDir := filepath.Join(opencode.HomeDir(), "commands")
	for _, cmd := range manifest.Commands {
		cmdPath := filepath.Join(cmdsDir, cmd.ID+".md")
		if _, err := os.Stat(cmdPath); os.IsNotExist(err) {
			t.Errorf("opencode command %s missing: %s", cmd.ID, cmdPath)
		}
		// Verify placeholder replaced (only arch.md uses BASE_PATH)
		content, _ := os.ReadFile(cmdPath)
		if strings.Contains(string(content), manifest.PlaceholderBasePath) {
			t.Errorf("opencode command %s: placeholder not replaced", cmd.ID)
		}
	}

	// opencode.json patched
	opencodeJSON := filepath.Join(opencode.HomeDir(), "opencode.json")
	configData, err := os.ReadFile(opencodeJSON)
	if err != nil {
		t.Fatalf("read opencode.json: %v", err)
	}
	var config map[string]any
	if err := json.Unmarshal(configData, &config); err != nil {
		t.Fatalf("parse opencode.json: %v", err)
	}
	agents, _ := config["agent"].(map[string]any)
	if agents == nil {
		t.Fatal("opencode.json has no agent key after install")
	}
	for _, role := range manifest.Roles {
		if _, exists := agents[role.ID]; !exists {
			t.Errorf("opencode.json missing agent entry for %s", role.ID)
		}
	}

	// Skill registry for opencode
	registryPath := filepath.Join(opencode.HomeDir(), ".atl", "skill-registry.md")
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		t.Errorf("opencode skill-registry.md missing: %s", registryPath)
	} else {
		content, _ := os.ReadFile(registryPath)
		if !strings.Contains(string(content), basePath) {
			t.Errorf("opencode skill registry does not contain base path: %s", basePath)
		}
	}

	// ======== Verify claude ========

	// Skills
	for _, skill := range manifest.Skills {
		skillDir := filepath.Join(claude.SkillsDir(), skill)
		entries, err := os.ReadDir(skillDir)
		if err != nil || len(entries) == 0 {
			t.Errorf("claude skill %s: no files at %s (err=%v)", skill, skillDir, err)
		}
	}

	// Prompts
	for _, role := range manifest.Roles {
		promptPath := filepath.Join(claude.PromptsDir(), role.ID+".md")
		if _, err := os.Stat(promptPath); os.IsNotExist(err) {
			t.Errorf("claude prompt %s missing: %s", role.ID, promptPath)
		}
		// Verify placeholder replaced (doc-arch does not use BASE_PATH)
		content, _ := os.ReadFile(promptPath)
		if strings.Contains(string(content), manifest.PlaceholderBasePath) {
			t.Errorf("claude prompt %s: placeholder not replaced", role.ID)
		}
	}

	// Agents
	for _, role := range manifest.Roles {
		agentFile := agentFileFor("claude", role)
		if agentFile == "" {
			continue
		}
		agentPath := filepath.Join(claude.AgentsDir(), filepath.Base(agentFile))
		if _, err := os.Stat(agentPath); os.IsNotExist(err) {
			t.Errorf("claude agent %s missing: %s", role.ID, agentPath)
		}
		// Verify placeholder replaced (doc-arch agent may not use BASE_PATH)
		content, _ := os.ReadFile(agentPath)
		if strings.Contains(string(content), manifest.PlaceholderBasePath) {
			t.Errorf("claude agent %s: placeholder not replaced", role.ID)
		}
	}

	// Skill registry for claude
	registryPath = filepath.Join(claude.HomeDir(), ".atl", "skill-registry.md")
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		t.Errorf("claude skill-registry.md missing: %s", registryPath)
	} else {
		content, _ := os.ReadFile(registryPath)
		if !strings.Contains(string(content), basePath) {
			t.Errorf("claude skill registry does not contain base path: %s", basePath)
		}
	}

	// claude should have no commands
	cmdsDir = filepath.Join(claude.HomeDir(), "commands")
	if _, err := os.Stat(cmdsDir); !os.IsNotExist(err) {
		t.Errorf("claude should not have a commands dir, but found: %s", cmdsDir)
	}
}

// ---------------------------------------------------------------------------
// 7.6 opencode.json patch roundtrip
// ---------------------------------------------------------------------------

// TestOpencodeJsonPatchRoundtrip pre-populates opencode.json with non-doc-agent-ai
// agents, runs install (PatchConfig), verifies doc-agent-ai agents are added AND
// existing agents survive. Then runs RemoveConfig (uninstall-like removal) and
// verifies doc-agent-ai agents are gone but existing agents are still intact.
func TestOpencodeJsonPatchRoundtrip(t *testing.T) {
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	defer restoreHome()

	home := filepath.Join(tmpHome, ".config", "opencode")
	if err := os.MkdirAll(home, 0755); err != nil {
		t.Fatalf("create opencode dir: %v", err)
	}

	// Keep track of the exact non-doc-agent-ai agent we own.
	const existingAgentID = "custom-agent"
	const existingAgentDesc = "my own agent"

	// Pre-populate opencode.json with a non-doc-agent-ai agent AND a custom
	// top-level key so we can verify roundtrip fidelity.
	preConfig := map[string]any{
		"customSetting": "should-survive",
		"agent": map[string]any{
			existingAgentID: map[string]any{
				"description": existingAgentDesc,
				"mode":        "auto-edit",
				"prompt":      "custom prompt",
				"tools":       map[string]bool{"bash": true},
			},
		},
	}
	preData, _ := json.MarshalIndent(preConfig, "", "  ")
	configPath := filepath.Join(home, "opencode.json")
	if err := os.WriteFile(configPath, preData, 0644); err != nil {
		t.Fatalf("write pre-populated opencode.json: %v", err)
	}

	// Build a manifest that includes doc-agent-ai roles.
	manifest := makeTestManifest()
	basePath := "/home/user/projects/"

	plat := newPlatformForTest(t, "opencode", home)

	// ----- Phase 1: Install (PatchConfig) -----
	if err := plat.PatchConfig(manifest, basePath, nil); err != nil {
		t.Fatalf("PatchConfig (install): %v", err)
	}

	// Verify post-install state
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read after patch: %v", err)
	}
	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("parse after patch: %v", err)
	}

	// Custom top-level key survived
	if v, ok := config["customSetting"]; !ok || v != "should-survive" {
		t.Errorf("customSetting lost: got %v", config["customSetting"])
	}

	agents, _ := config["agent"].(map[string]any)
	if agents == nil {
		t.Fatal("agent key missing after patch")
	}

	// Existing agent survived
	existing, ok := agents[existingAgentID].(map[string]any)
	if !ok {
		t.Errorf("pre-existing agent %q removed by install", existingAgentID)
	} else if existing["description"] != existingAgentDesc {
		t.Errorf("pre-existing agent description changed: got %v", existing["description"])
	}

	// All doc-agent-ai roles are present
	for _, role := range manifest.Roles {
		if _, exists := agents[role.ID]; !exists {
			t.Errorf("doc-agent-ai agent %s missing after install", role.ID)
		}
	}

	// ----- Phase 2: Uninstall (RemoveConfig) -----
	if err := plat.RemoveConfig(manifest, nil); err != nil {
		t.Fatalf("RemoveConfig (uninstall): %v", err)
	}

	// Verify post-uninstall state
	data2, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read after remove: %v", err)
	}
	var config2 map[string]any
	if err := json.Unmarshal(data2, &config2); err != nil {
		t.Fatalf("parse after remove: %v", err)
	}

	// Custom top-level key still survived
	if v, ok := config2["customSetting"]; !ok || v != "should-survive" {
		t.Errorf("customSetting lost after uninstall: got %v", config2["customSetting"])
	}

	agents2, _ := config2["agent"].(map[string]any)
	if agents2 == nil {
		t.Fatal("agent key missing after uninstall (should still exist if custom agents remain)")
	}

	// Existing agent still there
	existing2, ok2 := agents2[existingAgentID].(map[string]any)
	if !ok2 {
		t.Errorf("pre-existing agent %q removed by uninstall", existingAgentID)
	} else if existing2["description"] != existingAgentDesc {
		t.Errorf("pre-existing agent description changed after uninstall: got %v", existing2["description"])
	}

	// All doc-agent-ai roles are gone
	for _, role := range manifest.Roles {
		if _, exists := agents2[role.ID]; exists {
			t.Errorf("doc-agent-ai agent %s still present after uninstall", role.ID)
		}
	}
}
