package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// Legacy command uninstall sweep tests (T-03)
// ---------------------------------------------------------------------------

// TestUninstallSweep_RemovesCurrentAndLegacy seeds the commands dir with a
// current doc-* file and a legacy bare-name file, then runs uninstall and
// asserts both are removed.
func TestUninstallSweep_RemovesCurrentAndLegacy(t *testing.T) {
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

	opencodeHome := filepath.Join(tmpHome, ".config", "opencode")
	if err := os.MkdirAll(opencodeHome, 0755); err != nil {
		t.Fatalf("create opencode dir: %v", err)
	}
	cfg := map[string]any{"agent": map[string]any{}}
	data, _ := json.Marshal(cfg)
	if err := os.WriteFile(filepath.Join(opencodeHome, "opencode.json"), data, 0644); err != nil {
		t.Fatalf("write opencode.json: %v", err)
	}
	plat := newPlatformForTest(t, "opencode", opencodeHome)

	cmdsDir := filepath.Join(opencodeHome, "commands")
	if err := os.MkdirAll(cmdsDir, 0755); err != nil {
		t.Fatalf("create cmds dir: %v", err)
	}

	// Seed: one current file + one legacy file.
	currentFile := filepath.Join(cmdsDir, "doc-prd.md")
	legacyFile := filepath.Join(cmdsDir, "prd.md")
	if err := os.WriteFile(currentFile, []byte("# doc-prd"), 0644); err != nil {
		t.Fatalf("seed current file: %v", err)
	}
	if err := os.WriteFile(legacyFile, []byte("# prd legacy"), 0644); err != nil {
		t.Fatalf("seed legacy file: %v", err)
	}

	details := installedDetails{
		platform: plat,
		commands: []string{"doc-prd"},
	}
	uninstallPlatform(details, manifest)

	if _, err := os.Stat(currentFile); err == nil {
		t.Error("doc-prd.md (current) should have been removed by uninstall")
	}
	if _, err := os.Stat(legacyFile); err == nil {
		t.Error("prd.md (legacy) should have been removed by uninstall sweep")
	}
}

// TestUninstallSweep_OnlyCurrentPresent seeds only the 11 doc-* files and
// verifies they are all removed with no error.
func TestUninstallSweep_OnlyCurrentPresent(t *testing.T) {
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

	opencodeHome := filepath.Join(tmpHome, ".config", "opencode")
	if err := os.MkdirAll(opencodeHome, 0755); err != nil {
		t.Fatalf("create opencode dir: %v", err)
	}
	cfg := map[string]any{"agent": map[string]any{}}
	data, _ := json.Marshal(cfg)
	if err := os.WriteFile(filepath.Join(opencodeHome, "opencode.json"), data, 0644); err != nil {
		t.Fatalf("write opencode.json: %v", err)
	}
	plat := newPlatformForTest(t, "opencode", opencodeHome)

	cmdsDir := filepath.Join(opencodeHome, "commands")
	if err := os.MkdirAll(cmdsDir, 0755); err != nil {
		t.Fatalf("create cmds dir: %v", err)
	}

	// Seed only doc-* files (current naming).
	var currentCmdIDs []string
	for _, cmd := range manifest.Commands {
		p := filepath.Join(cmdsDir, cmd.ID+".md")
		if err := os.WriteFile(p, []byte("# "+cmd.ID), 0644); err != nil {
			t.Fatalf("seed command file %s: %v", cmd.ID, err)
		}
		currentCmdIDs = append(currentCmdIDs, cmd.ID)
	}

	details := installedDetails{
		platform: plat,
		commands: currentCmdIDs,
	}
	// Must not error even if no legacy files exist.
	uninstallPlatform(details, manifest)

	for _, cmd := range manifest.Commands {
		p := filepath.Join(cmdsDir, cmd.ID+".md")
		if _, err := os.Stat(p); err == nil {
			t.Errorf("command file %s.md should have been removed", cmd.ID)
		}
	}
}

// TestUninstallSweep_OnlyLegacyPresent seeds only the 11 bare-name legacy
// files (user on v3.x, never reinstalled) and verifies all are removed.
func TestUninstallSweep_OnlyLegacyPresent(t *testing.T) {
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

	opencodeHome := filepath.Join(tmpHome, ".config", "opencode")
	if err := os.MkdirAll(opencodeHome, 0755); err != nil {
		t.Fatalf("create opencode dir: %v", err)
	}
	cfg := map[string]any{"agent": map[string]any{}}
	data, _ := json.Marshal(cfg)
	if err := os.WriteFile(filepath.Join(opencodeHome, "opencode.json"), data, 0644); err != nil {
		t.Fatalf("write opencode.json: %v", err)
	}
	plat := newPlatformForTest(t, "opencode", opencodeHome)

	cmdsDir := filepath.Join(opencodeHome, "commands")
	if err := os.MkdirAll(cmdsDir, 0755); err != nil {
		t.Fatalf("create cmds dir: %v", err)
	}

	// Seed only legacy (bare-name) files.
	legacyNames := []string{"arch", "idea", "rec", "prd", "refine", "tech", "pti", "mod", "feat", "ddd", "to-sdd"}
	for _, id := range legacyNames {
		p := filepath.Join(cmdsDir, id+".md")
		if err := os.WriteFile(p, []byte("# legacy "+id), 0644); err != nil {
			t.Fatalf("seed legacy file %s: %v", id, err)
		}
	}

	// No current commands present — commands list is empty; sweep runs anyway via LegacyCommandIds.
	details := installedDetails{
		platform: plat,
		commands: nil,
	}
	uninstallPlatform(details, manifest)

	for _, id := range legacyNames {
		p := filepath.Join(cmdsDir, id+".md")
		if _, err := os.Stat(p); err == nil {
			t.Errorf("legacy file %s.md should have been removed by uninstall sweep", id)
		}
	}
}

// ---------------------------------------------------------------------------
// 7.7 Uninstall from mixed content
// ---------------------------------------------------------------------------

// TestUninstallMixedContent creates a temp HOME with:
//   - opencode platform with doc-agent-ai artifacts (skills, prompts, commands,
//     agents in opencode.json) + non-doc-agent-ai noise
//   - claude platform with doc-agent-ai artifacts + noise
// Then runs uninstall logic and verifies:
//   - Only doc-agent-ai artifacts are removed
//   - Non-doc-agent-ai files survive
//   - Empty dirs are pruned (stopping at home dir)
//   - opencode.json preserves non-doc-agent-ai entries
func TestUninstallMixedContent(t *testing.T) {
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	defer restoreHome()

	// Generate dist into a temp location
	distDir := filepath.Join(t.TempDir(), "dist")
	if err := generate(distDir); err != nil {
		t.Fatalf("generate dist: %v", err)
	}

	manifest, err := readManifestFrom(distDir)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	// ======== Create opencode platform ========
	opencodeHome := filepath.Join(tmpHome, ".config", "opencode")
	if err := os.MkdirAll(opencodeHome, 0755); err != nil {
		t.Fatalf("create opencode dir: %v", err)
	}

	// Write pre-populated opencode.json with a non-doc-agent-ai agent
	nonDocAgentID := "my-custom-agent"
	nonDocSetting := "should-survive-uninstall"
	preConfig := map[string]any{
		"customSetting": nonDocSetting,
		"agent": map[string]any{
			nonDocAgentID: map[string]any{
				"description": "A user-defined agent",
				"mode":        "auto-edit",
				"prompt":      "custom prompt string",
				"tools":       map[string]bool{"bash": true},
			},
		},
	}
	configPath := filepath.Join(opencodeHome, "opencode.json")
	cfgData, _ := json.MarshalIndent(preConfig, "", "  ")
	if err := os.WriteFile(configPath, cfgData, 0644); err != nil {
		t.Fatalf("write opencode.json: %v", err)
	}

	opencode := newPlatformForTest(t, "opencode", opencodeHome)

	// ======== Create claude platform ========
	claudeHome := filepath.Join(tmpHome, ".claude")
	if err := os.MkdirAll(claudeHome, 0755); err != nil {
		t.Fatalf("create claude dir: %v", err)
	}
	claude := newPlatformForTest(t, "claude", claudeHome)

	basePath := filepath.ToSlash(filepath.Join(tmpHome, "projects")) + "/"

	// ======== Install doc-agent-ai to both platforms ========
	if err := installPlatforms(manifest, []Platform{opencode, claude}, basePath, distDir); err != nil {
		t.Fatalf("installPlatforms: %v", err)
	}

	// ======== Plant non-doc-agent-ai noise ========
	// opencode noise: a custom skill dir that is NOT in the manifest
	noiseSkillDir := filepath.Join(opencode.SkillsDir(), "my-own-skill")
	if err := os.MkdirAll(noiseSkillDir, 0755); err != nil {
		t.Fatalf("create noise skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(noiseSkillDir, "SKILL.md"), []byte("# My Skill\n"), 0644); err != nil {
		t.Fatalf("write noise skill file: %v", err)
	}

	// opencode noise: a custom command file
	noiseCmdPath := filepath.Join(opencodeHome, "commands", "my-cmd.md")
	if err := os.MkdirAll(filepath.Dir(noiseCmdPath), 0755); err != nil {
		t.Fatalf("create noise cmd dir: %v", err)
	}
	if err := os.WriteFile(noiseCmdPath, []byte("# My Command"), 0644); err != nil {
		t.Fatalf("write noise cmd: %v", err)
	}

	// claude noise: a custom agent file in agents/
	noiseAgentPath := filepath.Join(claudeHome, "agents", "custom-agent.md")
	if err := os.MkdirAll(filepath.Dir(noiseAgentPath), 0755); err != nil {
		t.Fatalf("create noise agent dir: %v", err)
	}
	if err := os.WriteFile(noiseAgentPath, []byte("# My Agent"), 0644); err != nil {
		t.Fatalf("write noise agent: %v", err)
	}

	// claude noise: a full unknown skill directory
	noiseClaudeSkill := filepath.Join(claude.SkillsDir(), "completely-unrelated")
	if err := os.MkdirAll(noiseClaudeSkill, 0755); err != nil {
		t.Fatalf("create claude noise skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(noiseClaudeSkill, "README.md"), []byte("nope"), 0644); err != nil {
		t.Fatalf("write claude noise skill file: %v", err)
	}

	// ======== Verify pre-uninstall state (sanity) ========
	// opencode.json has both doc-agent-ai AND custom agent
	var config map[string]any
	raw, _ := os.ReadFile(configPath)
	json.Unmarshal(raw, &config)
	agents, _ := config["agent"].(map[string]any)
	if _, exists := agents[nonDocAgentID]; !exists {
		t.Fatal("precondition: custom agent missing before uninstall")
	}
	hasDocAgent := false
	for _, role := range manifest.Roles {
		if _, exists := agents[role.ID]; exists {
			hasDocAgent = true
			break
		}
	}
	if !hasDocAgent {
		t.Fatal("precondition: no doc-agent-ai agents in opencode.json before uninstall")
	}

	// opencode noise skill exists
	if _, err := os.Stat(filepath.Join(noiseSkillDir, "SKILL.md")); os.IsNotExist(err) {
		t.Fatal("precondition: noise skill file missing before uninstall")
	}

	// claude noise agent exists
	if _, err := os.Stat(noiseAgentPath); os.IsNotExist(err) {
		t.Fatal("precondition: claude noise agent missing before uninstall")
	}

	// ======== Run uninstall ========
	installed := checkWhatIsInstalled(manifest, []Platform{opencode, claude})

	if len(installed) == 0 {
		t.Fatal("checkWhatIsInstalled returned empty — expected both platforms to have artifacts")
	}
	if len(installed) != 2 {
		t.Errorf("expected 2 platforms with artifacts, got %d", len(installed))
	}

	for _, details := range installed {
		uninstallPlatform(details, manifest)
	}

	// ======== Verify post-uninstall state ========

	// --- opencode: non-doc-agent-ai agent survives ---
	raw2, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read opencode.json after uninstall: %v", err)
	}
	var config2 map[string]any
	if err := json.Unmarshal(raw2, &config2); err != nil {
		t.Fatalf("parse opencode.json after uninstall: %v", err)
	}

	// Custom top-level key survives
	if v, ok := config2["customSetting"]; !ok || v != nonDocSetting {
		t.Errorf("customSetting lost after uninstall: got %v", v)
	}

	agents2, _ := config2["agent"].(map[string]any)
	if agents2 == nil {
		t.Fatal("agent key missing after uninstall (custom agent should still be there)")
	}

	// Non-doc-agent-ai agent survives
	if _, exists := agents2[nonDocAgentID]; !exists {
		t.Errorf("non-doc-agent-ai agent %q removed by uninstall — should have survived", nonDocAgentID)
	}

	// All doc-agent-ai agents removed
	for _, role := range manifest.Roles {
		if _, exists := agents2[role.ID]; exists {
			t.Errorf("doc-agent-ai agent %s still present after uninstall", role.ID)
		}
	}

	// --- opencode: doc-agent-ai skills removed ---
	for _, skill := range manifest.Skills {
		skillDir := filepath.Join(opencode.SkillsDir(), skill)
		if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
			t.Errorf("doc-agent-ai skill dir not removed: %s", skillDir)
		}
	}

	// --- opencode: non-doc-agent-ai skill survives ---
	if _, err := os.Stat(filepath.Join(noiseSkillDir, "SKILL.md")); os.IsNotExist(err) {
		t.Error("non-doc-agent-ai skill file removed — should have survived")
	}

	// --- opencode: doc-agent-ai commands removed ---
	cmdsDir := filepath.Join(opencodeHome, "commands")
	for _, cmd := range manifest.Commands {
		cmdPath := filepath.Join(cmdsDir, cmd.ID+".md")
		if _, err := os.Stat(cmdPath); !os.IsNotExist(err) {
			t.Errorf("doc-agent-ai command not removed: %s", cmdPath)
		}
	}

	// --- opencode: non-doc-agent-ai command survives ---
	if _, err := os.Stat(noiseCmdPath); os.IsNotExist(err) {
		t.Error("non-doc-agent-ai command file removed — should have survived")
	}

	// --- opencode: doc-agent-ai prompts removed ---
	for _, role := range manifest.Roles {
		promptPath := filepath.Join(opencode.PromptsDir(), role.ID+".md")
		if _, err := os.Stat(promptPath); !os.IsNotExist(err) {
			t.Errorf("doc-agent-ai prompt not removed: %s", promptPath)
		}
	}

	// --- opencode: skill registry removed ---
	registryPath := filepath.Join(opencodeHome, ".atl", "skill-registry.md")
	if _, err := os.Stat(registryPath); !os.IsNotExist(err) {
		t.Errorf("skill-registry.md not removed: %s", registryPath)
	}

	// --- opencode: empty prompts/doc dir pruned, but prompts/ survives if empty ---
	promptsDir := opencode.PromptsDir()
	if _, err := os.Stat(promptsDir); !os.IsNotExist(err) {
		// prompts/doc may still exist if it has non-doc-agent-ai files, but
		// in our case it should be empty and pruned
		entries, _ := os.ReadDir(promptsDir)
		if len(entries) > 0 {
			t.Errorf("prompts/doc dir should be empty after uninstall, got %d entries", len(entries))
		}
	}

	// --- opencode: empty skills/ dir should survive since it has noise skill ---
	if _, err := os.Stat(opencode.SkillsDir()); os.IsNotExist(err) {
		t.Error("skills/ dir removed entirely — should survive because noise skill exists")
	}

	// --- claude: doc-agent-ai skills removed ---
	for _, skill := range manifest.Skills {
		skillDir := filepath.Join(claude.SkillsDir(), skill)
		if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
			t.Errorf("claude skill dir not removed: %s", skillDir)
		}
	}

	// --- claude: non-doc-agent-ai skill survives ---
	if _, err := os.Stat(filepath.Join(noiseClaudeSkill, "README.md")); os.IsNotExist(err) {
		t.Error("claude non-doc-agent-ai skill file removed — should have survived")
	}

	// --- claude: doc-agent-ai agents removed ---
	for _, role := range manifest.Roles {
		agentFile := agentFileFor("claude", role)
		if agentFile == "" {
			continue
		}
		agentPath := filepath.Join(claude.AgentsDir(), filepath.Base(agentFile))
		if _, err := os.Stat(agentPath); !os.IsNotExist(err) {
			t.Errorf("claude doc-agent-ai agent not removed: %s", agentPath)
		}
	}

	// --- claude: non-doc-agent-ai agent survives ---
	if _, err := os.Stat(noiseAgentPath); os.IsNotExist(err) {
		t.Error("claude non-doc-agent-ai agent removed — should have survived")
	}

	// --- claude: prompt files removed ---
	for _, role := range manifest.Roles {
		promptPath := filepath.Join(claude.PromptsDir(), role.ID+".md")
		if _, err := os.Stat(promptPath); !os.IsNotExist(err) {
			t.Errorf("claude prompt not removed: %s", promptPath)
		}
	}

	// --- claude: skill registry removed ---
	claudeRegistryPath := filepath.Join(claudeHome, ".atl", "skill-registry.md")
	if _, err := os.Stat(claudeRegistryPath); !os.IsNotExist(err) {
		t.Errorf("claude skill-registry.md not removed: %s", claudeRegistryPath)
	}

	// --- claude: prompts/doc dir pruned ---
	claudePromptsDir := claude.PromptsDir()
	if _, err := os.Stat(claudePromptsDir); !os.IsNotExist(err) {
		entries, _ := os.ReadDir(claudePromptsDir)
		if len(entries) > 0 {
			t.Errorf("claude prompts/doc dir should be empty after uninstall, got %d entries", len(entries))
		}
	}

	// --- claude: empty agents/ dir pruned (only had doc-agent-ai and noise, noise survives) ---
	// agents/ should survive because noise agent exists
	if _, err := os.Stat(claude.AgentsDir()); os.IsNotExist(err) {
		t.Error("claude agents/ dir removed entirely — should survive because noise agent exists")
	}
}

// TestUninstallNothingInstalled verifies that running uninstall on platforms
// with no doc-agent-ai artifacts returns no installed details.
func TestUninstallNothingInstalled(t *testing.T) {
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	defer restoreHome()

	manifest := makeTestManifest()

	// Create opencode with opencode.json but NO doc-agent-ai artifacts
	opencodeHome := filepath.Join(tmpHome, ".config", "opencode")
	if err := os.MkdirAll(opencodeHome, 0755); err != nil {
		t.Fatalf("create opencode dir: %v", err)
	}
	preConfig := map[string]any{
		"agent": map[string]any{},
	}
	cfgData, _ := json.MarshalIndent(preConfig, "", "  ")
	if err := os.WriteFile(filepath.Join(opencodeHome, "opencode.json"), cfgData, 0644); err != nil {
		t.Fatalf("write opencode.json: %v", err)
	}
	opencode := newPlatformForTest(t, "opencode", opencodeHome)

	installed := checkWhatIsInstalled(manifest, []Platform{opencode})

	if len(installed) != 0 {
		t.Errorf("expected no installed artifacts, got %d: %v", len(installed), installed)
	}
}

// TestUninstallEmptyDirPruning verifies that after removing all artifacts,
// empty parent directories are cleaned up (stopping at the platform home).
func TestUninstallEmptyDirPruning(t *testing.T) {
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	defer restoreHome()

	// Generate dist
	distDir := filepath.Join(t.TempDir(), "dist")
	if err := generate(distDir); err != nil {
		t.Fatalf("generate dist: %v", err)
	}

	manifest, err := readManifestFrom(distDir)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	// Create qwen platform (no opencode.json noise, just install)
	qwenHome := filepath.Join(tmpHome, ".qwen")
	if err := os.MkdirAll(qwenHome, 0755); err != nil {
		t.Fatalf("create qwen dir: %v", err)
	}
	qwen := newPlatformForTest(t, "qwen", qwenHome)

	basePath := filepath.ToSlash(filepath.Join(tmpHome, "projects")) + "/"

	// Install to qwen only (skills, prompts, agents)
	if err := installPlatforms(manifest, []Platform{qwen}, basePath, distDir); err != nil {
		t.Fatalf("installPlatforms: %v", err)
	}

	// Verify pre-uninstall: all dirs exist
	skillsDir := qwen.SkillsDir()
	promptsDir := qwen.PromptsDir()
	agentsDir := qwen.AgentsDir()

	for _, d := range []string{skillsDir, promptsDir, agentsDir} {
		if _, err := os.Stat(d); os.IsNotExist(err) {
			t.Fatalf("precondition: %s should exist after install", d)
		}
	}

	// Uninstall
	installed := checkWhatIsInstalled(manifest, []Platform{qwen})
	for _, details := range installed {
		uninstallPlatform(details, manifest)
	}

	// Verify post-uninstall: skills/, prompts/doc/, agents/ should all be
	// empty and pruned (no noise files in this test)
	for _, d := range []string{skillsDir, promptsDir, agentsDir} {
		if _, err := os.Stat(d); !os.IsNotExist(err) {
			t.Errorf("directory not pruned after uninstall: %s", d)
		}
	}

	// But the platform home itself should survive
	if _, err := os.Stat(qwenHome); os.IsNotExist(err) {
		t.Errorf("platform home dir removed: %s", qwenHome)
	}
}
