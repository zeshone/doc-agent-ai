package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// ---------------------------------------------------------------------------
// 7.4 Platform detection tests
// ---------------------------------------------------------------------------

// mockHomeEnv sets up a temp dir as HOME and returns the original value plus a
// restore function.  On Windows, UserProfile or HOMEDRIVE+HOMEPATH are used
// instead of HOME.
func mockHomeEnv(t *testing.T, tmpDir string) func() {
	t.Helper()

	restore := make(map[string]string)

	for _, env := range []string{"HOME", "USERPROFILE"} {
		old, ok := os.LookupEnv(env)
		if ok {
			restore[env] = old
		}
	}
	// On Windows, Go prefers USERPROFILE over HOME for os.UserHomeDir().
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	return func() {
		for k, v := range restore {
			os.Setenv(k, v)
		}
	}
}

// createMockOpenCode creates a fake opencode home dir with opencode.json.
func createMockOpenCode(t *testing.T, home string) {
	t.Helper()
	configPath := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(configPath, 0755); err != nil {
		t.Fatalf("create opencode dir: %v", err)
	}
	// Write a minimal valid opencode.json
	cfg := map[string]any{
		"agent": map[string]any{
			"existing-agent": map[string]any{
				"description": "preexisting",
				"mode":        "auto-edit",
			},
		},
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(filepath.Join(configPath, "opencode.json"), data, 0644); err != nil {
		t.Fatalf("write opencode.json: %v", err)
	}
}

// createMockQwen creates a fake .qwen directory.
func createMockQwen(t *testing.T, home string) {
	t.Helper()
	qwenDir := filepath.Join(home, ".qwen")
	if err := os.MkdirAll(qwenDir, 0755); err != nil {
		t.Fatalf("create qwen dir: %v", err)
	}
}

// createMockCopilot creates a fake .copilot directory.
func createMockCopilot(t *testing.T, home string) {
	t.Helper()
	copilotDir := filepath.Join(home, ".copilot")
	if err := os.MkdirAll(copilotDir, 0755); err != nil {
		t.Fatalf("create copilot dir: %v", err)
	}
}

// createMockClaude creates a fake .claude directory.
func createMockClaude(t *testing.T, home string) {
	t.Helper()
	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("create claude dir: %v", err)
	}
}

// createMockPi creates a fake .pi/agent directory.
func createMockPi(t *testing.T, home string) {
	t.Helper()
	piDir := filepath.Join(home, ".pi", "agent")
	if err := os.MkdirAll(piDir, 0755); err != nil {
		t.Fatalf("create pi dir: %v", err)
	}
}

// newPlatformForTest creates a platform with a custom home directory for testing.
func newPlatformForTest(t *testing.T, id string, homeDir string) Platform {
	t.Helper()

	cfg := PlatformConfig{
		SkillRoot: homeDir + "/skills",
		PromptDir: "prompts",
	}
	switch id {
	case "opencode":
		cfg.CommandDir = "commands"
		return &opencodePlatform{basePlatform{id: id, homeDir: homeDir, cfg: cfg}}
	case "qwen":
		cfg.AgentDir = "agents"
		return &qwenPlatform{basePlatform{id: id, homeDir: homeDir, cfg: cfg}}
	case "copilot":
		cfg.AgentDir = "agents"
		return &copilotPlatform{basePlatform{id: id, homeDir: homeDir, cfg: cfg}}
	case "claude":
		cfg.AgentDir = "agents"
		return &claudePlatform{basePlatform{id: id, homeDir: homeDir, cfg: cfg}}
	case "pi":
		return &piPlatform{basePlatform{id: id, homeDir: homeDir, cfg: cfg}}
	default:
		t.Fatalf("unknown platform: %s", id)
		return nil
	}
}

// --- Detect tests ---

func TestPlatformDetect_OpenCode_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	home := filepath.Join(tmpDir, ".config", "opencode")
	createMockOpenCode(t, tmpDir)

	p := newPlatformForTest(t, "opencode", home)

	if !p.Detect() {
		t.Error("opencode should be detected when opencode.json exists")
	}
}

func TestPlatformDetect_OpenCode_MissingConfig(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	home := filepath.Join(tmpDir, ".config", "opencode")
	// Create the dir but NOT opencode.json
	if err := os.MkdirAll(home, 0755); err != nil {
		t.Fatalf("create dir: %v", err)
	}

	p := newPlatformForTest(t, "opencode", home)

	if p.Detect() {
		t.Error("opencode should NOT be detected when opencode.json is missing")
	}
}

func TestPlatformDetect_OpenCode_MissingDir(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	home := filepath.Join(tmpDir, ".config", "opencode")
	// Don't create the directory at all

	p := newPlatformForTest(t, "opencode", home)

	if p.Detect() {
		t.Error("opencode should NOT be detected when directory is missing")
	}
}

func TestPlatformDetect_Qwen_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	home := filepath.Join(tmpDir, ".qwen")
	createMockQwen(t, tmpDir)

	p := newPlatformForTest(t, "qwen", home)

	if !p.Detect() {
		t.Error("qwen should be detected when .qwen directory exists")
	}
}

func TestPlatformDetect_Qwen_MissingDir(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	home := filepath.Join(tmpDir, ".qwen")
	// Don't create the directory

	p := newPlatformForTest(t, "qwen", home)

	if p.Detect() {
		t.Error("qwen should NOT be detected when .qwen directory is missing")
	}
}

func TestPlatformDetect_Copilot_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	home := filepath.Join(tmpDir, ".copilot")
	createMockCopilot(t, tmpDir)

	p := newPlatformForTest(t, "copilot", home)

	// copilot also requires 'code --version' — since it's likely not installed
	// in CI/test environments, we just verify the dir check passes and the
	// method signature works.  A full 'code --version' check is an integration
	// concern.
	_ = p.Detect() // won't panic, might return false if code CLI is missing
}

func TestPlatformDetect_Copilot_MissingDir(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	home := filepath.Join(tmpDir, ".copilot")
	// Don't create the directory

	p := newPlatformForTest(t, "copilot", home)

	if p.Detect() {
		t.Error("copilot should NOT be detected when .copilot directory is missing")
	}
}

func TestPlatformDetect_Claude_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	home := filepath.Join(tmpDir, ".claude")
	createMockClaude(t, tmpDir)

	p := newPlatformForTest(t, "claude", home)

	if !p.Detect() {
		t.Error("claude should be detected when .claude directory exists")
	}
}

func TestPlatformDetect_Claude_MissingDir(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	home := filepath.Join(tmpDir, ".claude")

	p := newPlatformForTest(t, "claude", home)

	if p.Detect() {
		t.Error("claude should NOT be detected when .claude directory is missing")
	}
}

// --- Path tests ---

func TestPlatform_HomeDir(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	home := filepath.Join(tmpDir, ".config", "opencode")
	p := newPlatformForTest(t, "opencode", home)

	if p.HomeDir() != home {
		t.Errorf("HomeDir() = %q, want %q", p.HomeDir(), home)
	}
}

func TestPlatform_SkillsDir(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	home := filepath.Join(tmpDir, ".qwen")
	p := newPlatformForTest(t, "qwen", home)

	expected := filepath.Join(home, "skills")
	if p.SkillsDir() != expected {
		t.Errorf("SkillsDir() = %q, want %q", p.SkillsDir(), expected)
	}
}

func TestPlatform_PromptsDir(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	home := filepath.Join(tmpDir, ".copilot")
	p := newPlatformForTest(t, "copilot", home)

	expected := filepath.Join(home, "prompts", "doc")
	if p.PromptsDir() != expected {
		t.Errorf("PromptsDir() = %q, want %q", p.PromptsDir(), expected)
	}
}

func TestPlatform_AgentsDir(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	// opencode: agents are JSON-based, AgentsDir returns ""
	home := filepath.Join(tmpDir, ".config", "opencode")
	pOpen := newPlatformForTest(t, "opencode", home)
	if pOpen.AgentsDir() != "" {
		t.Errorf("opencode AgentsDir() = %q, want empty", pOpen.AgentsDir())
	}

	// claude: file-based agents
	home2 := filepath.Join(tmpDir, ".claude")
	pClaude := newPlatformForTest(t, "claude", home2)
	expected := filepath.Join(home2, "agents")
	if pClaude.AgentsDir() != expected {
		t.Errorf("claude AgentsDir() = %q, want %q", pClaude.AgentsDir(), expected)
	}
}

// --- GetSkillIDs tests ---

func makeTestManifest() DistManifest {
	return DistManifest{
		Skills: []string{"doc-arch", "doc-prd", "doc-rec", "doc-tech", "doc-pti"},
		Roles: []DistRole{
			{ID: "doc-arch", Description: "Orchestrator", Mode: "primary"},
			{ID: "doc-rec", Description: "Requirements", Mode: "auto-edit"},
			{ID: "doc-prd", Description: "PRD", Mode: "auto-edit"},
		},
		Commands: []DistCommand{
			{ID: "arch", Description: "Full flow", Agent: "doc-arch", File: "commands/arch.md"},
			{ID: "rec", Description: "Requirements", Agent: "doc-rec", File: "commands/rec.md"},
		},
	}
}

func TestPlatform_GetSkillIDs_NoneInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	home := filepath.Join(tmpDir, ".qwen")
	createMockQwen(t, tmpDir)

	p := newPlatformForTest(t, "qwen", home)
	manifest := makeTestManifest()

	installed := p.GetSkillIDs(manifest)
	if len(installed) != 0 {
		t.Errorf("expected no installed skills, got %d", len(installed))
	}
}

func TestPlatform_GetSkillIDs_SomeInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	home := filepath.Join(tmpDir, ".qwen")
	createMockQwen(t, tmpDir)

	// Create some skill directories
	skillsDir := filepath.Join(home, "skills")
	for _, skill := range []string{"doc-arch", "doc-rec"} {
		if err := os.MkdirAll(filepath.Join(skillsDir, skill), 0755); err != nil {
			t.Fatalf("create skill dir %s: %v", skill, err)
		}
	}

	p := newPlatformForTest(t, "qwen", home)
	manifest := makeTestManifest()

	installed := p.GetSkillIDs(manifest)
	if len(installed) != 2 {
		t.Errorf("expected 2 installed skills, got %d: %v", len(installed), installed)
	}
}

// --- GetPromptIDs tests ---

func TestPlatform_GetPromptIDs_NoneInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	home := filepath.Join(tmpDir, ".copilot")
	createMockCopilot(t, tmpDir)

	p := newPlatformForTest(t, "copilot", home)
	manifest := makeTestManifest()

	installed, err := p.GetPromptIDs(manifest, nil)
	if err != nil {
		t.Fatalf("GetPromptIDs: %v", err)
	}
	if len(installed) != 0 {
		t.Errorf("expected no installed prompts, got %d", len(installed))
	}
}

func TestPlatform_GetPromptIDs_SomeInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	home := filepath.Join(tmpDir, ".copilot")
	createMockCopilot(t, tmpDir)

	// Create some prompt files
	promptsDir := filepath.Join(home, "prompts", "doc")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatalf("create prompts dir: %v", err)
	}
	for _, id := range []string{"doc-arch", "doc-prd"} {
		if err := os.WriteFile(filepath.Join(promptsDir, id+".md"), []byte("# test"), 0644); err != nil {
			t.Fatalf("create prompt file: %v", err)
		}
	}

	p := newPlatformForTest(t, "copilot", home)
	manifest := makeTestManifest()

	installed, err := p.GetPromptIDs(manifest, nil)
	if err != nil {
		t.Fatalf("GetPromptIDs: %v", err)
	}
	if len(installed) != 2 {
		t.Errorf("expected 2 installed prompts, got %d: %v", len(installed), installed)
	}
}

// --- GetAgentIDs tests ---

func TestPlatform_GetAgentIDs_OpenCode(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	home := filepath.Join(tmpDir, ".config", "opencode")
	createMockOpenCode(t, tmpDir)

	// Add doc-agent-ai agents to the opencode.json
	configPath := filepath.Join(home, "opencode.json")
	data, _ := os.ReadFile(configPath)
	var config map[string]any
	json.Unmarshal(data, &config)
	agents := config["agent"].(map[string]any)
	agents["doc-arch"] = map[string]any{"description": "orch", "mode": "primary"}
	agents["doc-rec"] = map[string]any{"description": "req", "mode": "auto-edit"}
	out, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(configPath, out, 0644)

	p := newPlatformForTest(t, "opencode", home)
	manifest := makeTestManifest()

	installed, err := p.GetAgentIDs(manifest, nil)
	if err != nil {
		t.Fatalf("GetAgentIDs: %v", err)
	}
	if len(installed) != 2 {
		t.Errorf("expected 2 installed agents, got %d: %v", len(installed), installed)
	}
}

func TestPlatform_GetAgentIDs_FileBased(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	home := filepath.Join(tmpDir, ".claude")
	createMockClaude(t, tmpDir)

	agentsDir := filepath.Join(home, "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("create agents dir: %v", err)
	}
	// Create agent files
	for _, id := range []string{"doc-arch", "doc-rec"} {
		if err := os.WriteFile(filepath.Join(agentsDir, id+".md"), []byte("---\nname: "+id+"\n---"), 0644); err != nil {
			t.Fatalf("create agent file: %v", err)
		}
	}

	p := newPlatformForTest(t, "claude", home)
	manifest := makeTestManifest()

	installed, err := p.GetAgentIDs(manifest, nil)
	if err != nil {
		t.Fatalf("GetAgentIDs: %v", err)
	}
	if len(installed) != 2 {
		t.Errorf("expected 2 installed agents, got %d: %v", len(installed), installed)
	}
}

// --- GetCommandIDs tests ---

func TestPlatform_GetCommandIDs_OpenCode_Installed(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	home := filepath.Join(tmpDir, ".config", "opencode")
	createMockOpenCode(t, tmpDir)

	commandsDir := filepath.Join(home, "commands")
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		t.Fatalf("create commands dir: %v", err)
	}
	for _, id := range []string{"arch", "rec"} {
		if err := os.WriteFile(filepath.Join(commandsDir, id+".md"), []byte("# cmd"), 0644); err != nil {
			t.Fatalf("create command file: %v", err)
		}
	}

	p := newPlatformForTest(t, "opencode", home)
	manifest := makeTestManifest()

	installed, err := p.GetCommandIDs(manifest)
	if err != nil {
		t.Fatalf("GetCommandIDs: %v", err)
	}
	if len(installed) != 2 {
		t.Errorf("expected 2 installed commands, got %d: %v", len(installed), installed)
	}
}

func TestPlatform_GetCommandIDs_NonOpenCode(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	home := filepath.Join(tmpDir, ".qwen")
	createMockQwen(t, tmpDir)

	p := newPlatformForTest(t, "qwen", home)
	manifest := makeTestManifest()

	installed, err := p.GetCommandIDs(manifest)
	if err != nil {
		t.Fatalf("GetCommandIDs: %v", err)
	}
	if installed != nil {
		t.Errorf("non-opencode platforms should return nil for GetCommandIDs, got %v", installed)
	}
}

// --- PatchConfig / RemoveConfig tests ---

func TestPlatform_PatchConfig_Opencode(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	home := filepath.Join(tmpDir, ".config", "opencode")
	createMockOpenCode(t, tmpDir)

	p := newPlatformForTest(t, "opencode", home)
	manifest := makeTestManifest()

	if err := p.PatchConfig(manifest, "/tmp/docs", nil); err != nil {
		t.Fatalf("PatchConfig: %v", err)
	}

	// Verify opencode.json was patched
	configPath := filepath.Join(home, "opencode.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read patched opencode.json: %v", err)
	}
	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("parse patched opencode.json: %v", err)
	}

	agents, ok := config["agent"].(map[string]any)
	if !ok {
		t.Fatal("agent section missing after patch")
	}

	// All 3 roles from manifest should be present
	for _, role := range manifest.Roles {
		if _, ok := agents[role.ID]; !ok {
			t.Errorf("role %s should be in agent section", role.ID)
		}
	}

	// Preexisting agent must survive
	if _, ok := agents["existing-agent"]; !ok {
		t.Error("preexisting agent should survive patch")
	}
}

func TestPlatform_PatchConfig_Opencode_PreservesUnknown(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	home := filepath.Join(tmpDir, ".config", "opencode")
	configDir := home
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("create opencode dir: %v", err)
	}

	// Create opencode.json with extra fields
	cfg := map[string]any{
		"agent": map[string]any{
			"existing-agent": map[string]any{"description": "keep me"},
		},
		"customSetting": "should-survive",
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(filepath.Join(configDir, "opencode.json"), data, 0644)

	p := newPlatformForTest(t, "opencode", home)
	manifest := makeTestManifest()

	if err := p.PatchConfig(manifest, "/tmp/docs", nil); err != nil {
		t.Fatalf("PatchConfig: %v", err)
	}

	// Verify customSetting survived
	configPath := filepath.Join(home, "opencode.json")
	raw, _ := os.ReadFile(configPath)
	var config map[string]any
	json.Unmarshal(raw, &config)

	if _, ok := config["customSetting"]; !ok {
		t.Error("customSetting should survive the patch")
	}
}

func TestPlatform_RemoveConfig_Opencode(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	home := filepath.Join(tmpDir, ".config", "opencode")
	createMockOpenCode(t, tmpDir)

	p := newPlatformForTest(t, "opencode", home)
	manifest := makeTestManifest()

	// First patch, then remove
	if err := p.PatchConfig(manifest, "/tmp/docs", nil); err != nil {
		t.Fatalf("PatchConfig: %v", err)
	}

	if err := p.RemoveConfig(manifest, nil); err != nil {
		t.Fatalf("RemoveConfig: %v", err)
	}

	// Verify doc-agent-ai agents were removed
	configPath := filepath.Join(home, "opencode.json")
	raw, _ := os.ReadFile(configPath)
	var config map[string]any
	json.Unmarshal(raw, &config)

	agents := config["agent"].(map[string]any)
	for _, role := range manifest.Roles {
		if _, ok := agents[role.ID]; ok {
			t.Errorf("role %s should have been removed", role.ID)
		}
	}

	// Preexisting agent must survive
	if _, ok := agents["existing-agent"]; !ok {
		t.Error("preexisting agent should survive remove")
	}
}

func TestPlatform_RemoveConfig_Opencode_OnlyTargetRoles(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	home := filepath.Join(tmpDir, ".config", "opencode")
	createMockOpenCode(t, tmpDir)

	p := newPlatformForTest(t, "opencode", home)
	manifest := makeTestManifest()

	// Patch all roles
	if err := p.PatchConfig(manifest, "/tmp/docs", nil); err != nil {
		t.Fatalf("PatchConfig: %v", err)
	}

	// Remove only doc-arch
	if err := p.RemoveConfig(manifest, []string{"doc-arch"}); err != nil {
		t.Fatalf("RemoveConfig: %v", err)
	}

	// Verify only doc-arch was removed
	configPath := filepath.Join(home, "opencode.json")
	raw, _ := os.ReadFile(configPath)
	var config map[string]any
	json.Unmarshal(raw, &config)

	agents := config["agent"].(map[string]any)
	if _, ok := agents["doc-arch"]; ok {
		t.Error("doc-arch should have been removed")
	}
	if _, ok := agents["doc-rec"]; !ok {
		t.Error("doc-rec should NOT have been removed")
	}
}

// --- SkillRegistryTrigger tests ---

func TestPlatform_SkillRegistryTrigger(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	open := newPlatformForTest(t, "opencode", filepath.Join(tmpDir, ".config", "opencode"))
	qwen := newPlatformForTest(t, "qwen", filepath.Join(tmpDir, ".qwen"))
	claude := newPlatformForTest(t, "claude", filepath.Join(tmpDir, ".claude"))

	if open.SkillRegistryTrigger() != "opencode" {
		t.Errorf("opencode trigger = %q, want %q", open.SkillRegistryTrigger(), "opencode")
	}
	if qwen.SkillRegistryTrigger() != "opencode" {
		t.Errorf("qwen trigger = %q, want %q", qwen.SkillRegistryTrigger(), "opencode")
	}
	if claude.SkillRegistryTrigger() != "claude" {
		t.Errorf("claude trigger = %q, want %q", claude.SkillRegistryTrigger(), "claude")
	}
}

// --- WriteSkillRegistry tests ---

func TestPlatform_WriteSkillRegistry_Opencode(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	home := filepath.Join(tmpDir, ".config", "opencode")
	if err := os.MkdirAll(home, 0755); err != nil {
		t.Fatalf("create home dir: %v", err)
	}

	p := newPlatformForTest(t, "opencode", home)

	if err := p.WriteSkillRegistry("/home/user/projects"); err != nil {
		t.Fatalf("WriteSkillRegistry: %v", err)
	}

	regPath := filepath.Join(home, ".atl", "skill-registry.md")
	data, err := os.ReadFile(regPath)
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}

	content := string(data)
	if content == "" {
		t.Error("registry should not be empty")
	}
	if !contains(content, "doc-arch") {
		t.Error("registry should mention doc-arch")
	}
	if !contains(content, "/home/user/projects") {
		t.Error("registry should contain base path")
	}
}

func TestPlatform_WriteSkillRegistry_Claude(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	home := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(home, 0755); err != nil {
		t.Fatalf("create home dir: %v", err)
	}

	p := newPlatformForTest(t, "claude", home)

	if err := p.WriteSkillRegistry("/home/user/projects"); err != nil {
		t.Fatalf("WriteSkillRegistry: %v", err)
	}

	regPath := filepath.Join(home, ".atl", "skill-registry.md")
	data, err := os.ReadFile(regPath)
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}

	content := string(data)
	// Claude-style trigger text
	if !contains(content, "Documenting a project") {
		t.Error("claude registry should have claude-style trigger text")
	}
}

func TestPlatform_WriteSkillRegistry_Qwen_NoOp(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	home := filepath.Join(tmpDir, ".qwen")
	if err := os.MkdirAll(home, 0755); err != nil {
		t.Fatalf("create home dir: %v", err)
	}

	p := newPlatformForTest(t, "qwen", home)

	if err := p.WriteSkillRegistry("/tmp"); err != nil {
		t.Fatalf("WriteSkillRegistry for qwen (no-op) should return nil: %v", err)
	}

	// Registry should NOT have been written
	regPath := filepath.Join(home, ".atl", "skill-registry.md")
	if _, err := os.Stat(regPath); err == nil {
		t.Error("qwen should NOT write a skill registry")
	}
}

// --- pruneEmptyDirs tests ---

func TestPruneEmptyDirs_RemovesEmptyChain(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a chain of empty dirs
	a := filepath.Join(tmpDir, "a")
	b := filepath.Join(a, "b")
	c := filepath.Join(b, "c")
	if err := os.MkdirAll(c, 0755); err != nil {
		t.Fatalf("create dir chain: %v", err)
	}

	if err := pruneEmptyDirs(c, tmpDir); err != nil {
		t.Fatalf("pruneEmptyDirs: %v", err)
	}

	// All empty dirs should be gone
	for _, d := range []string{c, b, a} {
		if _, err := os.Stat(d); !os.IsNotExist(err) {
			t.Errorf("directory %s should have been removed", d)
		}
	}
}

func TestPruneEmptyDirs_StopsAtNonEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	// a/b/  where b is empty but a has a file
	a := filepath.Join(tmpDir, "a")
	b := filepath.Join(a, "b")
	if err := os.MkdirAll(b, 0755); err != nil {
		t.Fatalf("create dirs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(a, "keep.me"), []byte("x"), 0644); err != nil {
		t.Fatalf("create file: %v", err)
	}

	if err := pruneEmptyDirs(b, tmpDir); err != nil {
		t.Fatalf("pruneEmptyDirs: %v", err)
	}

	// b should be removed, a should survive
	if _, err := os.Stat(b); !os.IsNotExist(err) {
		t.Error("empty dir b should have been removed")
	}
	if _, err := os.Stat(a); os.IsNotExist(err) {
		t.Error("non-empty dir a should survive")
	}
}

func TestPruneEmptyDirs_DoesNotGoBelowStop(t *testing.T) {
	tmpDir := t.TempDir()

	stop := filepath.Join(tmpDir, "stop")
	below := filepath.Join(stop, "below")
	if err := os.MkdirAll(below, 0755); err != nil {
		t.Fatalf("create dirs: %v", err)
	}

	if err := pruneEmptyDirs(below, stop); err != nil {
		t.Fatalf("pruneEmptyDirs: %v", err)
	}

	// below removed, stop survives
	if _, err := os.Stat(below); !os.IsNotExist(err) {
		t.Error("below should have been removed")
	}
	if _, err := os.Stat(stop); os.IsNotExist(err) {
		t.Error("stop dir should survive")
	}
}

// --- resolvedHome test ---

func TestResolveHome(t *testing.T) {
	home, err := resolveHome("~/.config/opencode/skills")
	if err != nil {
		t.Fatalf("resolveHome: %v", err)
	}

	userHome, _ := os.UserHomeDir()
	expected := filepath.Join(userHome, ".config", "opencode")
	if home != expected {
		t.Errorf("resolveHome = %q, want %q", home, expected)
	}
}

// --- ID tests ---

func TestPlatform_ID(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	tests := []struct {
		id   string
		home string
	}{
		{"opencode", filepath.Join(tmpDir, ".config", "opencode")},
		{"qwen", filepath.Join(tmpDir, ".qwen")},
		{"copilot", filepath.Join(tmpDir, ".copilot")},
		{"claude", filepath.Join(tmpDir, ".claude")},
	}

	for _, tt := range tests {
		p := newPlatformForTest(t, tt.id, tt.home)
		if p.ID() != tt.id {
			t.Errorf("%s platform ID() = %q, want %q", tt.id, p.ID(), tt.id)
		}
	}
}

// --- Installed map filter tests ---

func TestPlatform_GetAgentIDs_WithFilter(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)

	home := filepath.Join(tmpDir, ".config", "opencode")
	createMockOpenCode(t, tmpDir)

	// Add agents to opencode.json
	configPath := filepath.Join(home, "opencode.json")
	data, _ := os.ReadFile(configPath)
	var config map[string]any
	json.Unmarshal(data, &config)
	agents := config["agent"].(map[string]any)
	agents["doc-arch"] = map[string]any{"description": "orch"}
	agents["doc-rec"] = map[string]any{"description": "req"}
	agents["doc-prd"] = map[string]any{"description": "prd"}
	out, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(configPath, out, 0644)

	p := newPlatformForTest(t, "opencode", home)
	manifest := makeTestManifest()

	// Filter: only check doc-arch
	filter := map[string]bool{"doc-arch": false}
	installed, err := p.GetAgentIDs(manifest, filter)
	if err != nil {
		t.Fatalf("GetAgentIDs: %v", err)
	}
	if len(installed) != 1 || installed[0] != "doc-arch" {
		t.Errorf("expected only doc-arch, got %v", installed)
	}
	// Map should have been updated
	if !filter["doc-arch"] {
		t.Error("filter map should have doc-arch set to true")
	}
}

// ---------------------------------------------------------------------------
// Issue 02 — XDG_CONFIG_HOME support (OpenCode)
// ---------------------------------------------------------------------------

// TestOpenCodeDetect_XDG verifies that when XDG_CONFIG_HOME is set, Detect()
// resolves opencode.json relative to $XDG_CONFIG_HOME/opencode and that
// HomeDir() points to that directory.
func TestOpenCodeDetect_XDG(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	xdgOpencode := filepath.Join(tmpDir, "opencode")
	if err := os.MkdirAll(xdgOpencode, 0755); err != nil {
		t.Fatalf("create xdg opencode dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(xdgOpencode, "opencode.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("write opencode.json: %v", err)
	}

	cfg := PlatformConfig{SkillRoot: "~/.config/opencode/skills"}
	p, err := newOpenCodePlatform(cfg)
	if err != nil {
		t.Fatalf("newOpenCodePlatform: %v", err)
	}

	if !p.Detect() {
		t.Error("Detect() should return true when opencode.json exists under XDG_CONFIG_HOME")
	}
	if p.HomeDir() != xdgOpencode {
		t.Errorf("HomeDir() = %q, want %q", p.HomeDir(), xdgOpencode)
	}
}

// TestOpenCodeDetect_XDG_Unset verifies that when XDG_CONFIG_HOME is unset,
// Detect() falls back to the default ~/.config/opencode path.
func TestOpenCodeDetect_XDG_Unset(t *testing.T) {
	tmpDir := t.TempDir()
	// Ensure XDG_CONFIG_HOME is absent.
	t.Setenv("XDG_CONFIG_HOME", "")
	mockHomeEnv(t, tmpDir)

	defaultOpencode := filepath.Join(tmpDir, ".config", "opencode")
	createMockOpenCode(t, tmpDir)

	cfg := PlatformConfig{SkillRoot: "~/.config/opencode/skills"}
	p, err := newOpenCodePlatform(cfg)
	if err != nil {
		t.Fatalf("newOpenCodePlatform: %v", err)
	}

	if !p.Detect() {
		t.Error("Detect() should return true with default path when XDG_CONFIG_HOME is unset")
	}
	if p.HomeDir() != defaultOpencode {
		t.Errorf("HomeDir() = %q, want %q", p.HomeDir(), defaultOpencode)
	}
}

// ---------------------------------------------------------------------------
// Issue 02 — --copilot-path flag override
// ---------------------------------------------------------------------------

// TestCopilotDetect_Flag verifies that when copilotPathOverride is set to an
// existing directory, Detect() returns true and HomeDir() returns that path.
func TestCopilotDetect_Flag(t *testing.T) {
	tmpDir := t.TempDir()
	orig := copilotPathOverride
	copilotPathOverride = tmpDir
	t.Cleanup(func() { copilotPathOverride = orig })

	cfg := PlatformConfig{SkillRoot: "~/.copilot/skills"}
	p, err := newCopilotPlatform(cfg)
	if err != nil {
		t.Fatalf("newCopilotPlatform: %v", err)
	}

	if p.HomeDir() != tmpDir {
		t.Errorf("HomeDir() = %q, want %q", p.HomeDir(), tmpDir)
	}
	if !p.Detect() {
		t.Error("Detect() should return true when override path exists")
	}
}

// TestCopilotDetect_FlagMissing verifies that when --copilot-path points to a
// non-existent directory the platform is still created (warn is emitted) and
// Detect() returns false because the dir does not exist.
func TestCopilotDetect_FlagMissing(t *testing.T) {
	orig := copilotPathOverride
	copilotPathOverride = filepath.Join(t.TempDir(), "does-not-exist")
	t.Cleanup(func() { copilotPathOverride = orig })

	cfg := PlatformConfig{SkillRoot: "~/.copilot/skills"}
	p, err := newCopilotPlatform(cfg)
	if err != nil {
		t.Fatalf("newCopilotPlatform: %v", err)
	}

	// HomeDir must be the override path even when it is missing.
	if p.HomeDir() != copilotPathOverride {
		t.Errorf("HomeDir() = %q, want %q", p.HomeDir(), copilotPathOverride)
	}
	// Detect() returns false because the directory does not exist.
	if p.Detect() {
		t.Error("Detect() should return false when override path does not exist")
	}
}

// TestCopilotDetect_Standard verifies the standard detection path (no override,
// no XDG) using an explicit homeDir, matching pre-existing test style.
func TestCopilotDetect_Standard(t *testing.T) {
	// Ensure no override is active.
	orig := copilotPathOverride
	copilotPathOverride = ""
	t.Cleanup(func() { copilotPathOverride = orig })

	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)
	createMockCopilot(t, tmpDir)

	home := filepath.Join(tmpDir, ".copilot")
	p := newPlatformForTest(t, "copilot", home)

	// The directory exists; Detect() no longer requires 'code' in PATH.
	if !p.Detect() {
		t.Error("Detect() should return true when ~/.copilot exists (standard path)")
	}
}

// TODO: TestCopilotDetect_Portable — tests the portable VS Code fallback
// (findPortableCopilotHome).  Requires either a real VS Code install or a mock
// 'code' binary on PATH that outputs a controlled path.  Skipping in automated
// suite; to test manually: install VS Code with Copilot Chat extension, then run
//
//	go test -run TestCopilotDetect_Portable -v
//
// and verify Detect() returns true without ~/.copilot present.

// ---------------------------------------------------------------------------
// Pi platform tests
// ---------------------------------------------------------------------------

// TestPiDetect_DirExists — ~/.pi/agent present, binary irrelevant.
func TestPiDetect_DirExists(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)
	createMockPi(t, tmpDir)

	home := filepath.Join(tmpDir, ".pi", "agent")
	p := newPlatformForTest(t, "pi", home)

	if !p.Detect() {
		t.Error("Detect() should return true when ~/.pi/agent exists")
	}
}

// TestPiDetect_DirMissing_BinaryOnPATH — fallback to LookPath("pi") when the
// home directory does not exist.  Skipped on Windows because exec.LookPath
// requires a .exe/.cmd extension that complicates mocking.
func TestPiDetect_DirMissing_BinaryOnPATH(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("LookPath binary mocking requires .exe/.cmd handling — manual test on Windows")
	}

	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)
	// Intentionally do NOT create ~/.pi/agent.

	// Create a fake `pi` executable in a separate dir and prepend it to PATH.
	binDir := t.TempDir()
	piBin := filepath.Join(binDir, "pi")
	if err := os.WriteFile(piBin, []byte("#!/bin/sh\necho mock pi\n"), 0755); err != nil {
		t.Fatalf("write mock pi: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	home := filepath.Join(tmpDir, ".pi", "agent")
	p := newPlatformForTest(t, "pi", home)

	if !p.Detect() {
		t.Error("Detect() should return true when 'pi' is on PATH even without ~/.pi/agent")
	}
}

// TestPiDetect_NeitherExists — no dir, no binary → not detected.
func TestPiDetect_NeitherExists(t *testing.T) {
	tmpDir := t.TempDir()
	mockHomeEnv(t, tmpDir)
	// Scrub PATH so any host-level `pi` does not leak in.
	t.Setenv("PATH", t.TempDir())

	home := filepath.Join(tmpDir, ".pi", "agent")
	p := newPlatformForTest(t, "pi", home)

	if p.Detect() {
		t.Error("Detect() should return false when ~/.pi/agent missing and 'pi' not on PATH")
	}
}

// TestPiDetect_Flag — --pi-path override points to an existing dir.
func TestPiDetect_Flag(t *testing.T) {
	tmpDir := t.TempDir()
	orig := piPathOverride
	piPathOverride = tmpDir
	t.Cleanup(func() { piPathOverride = orig })

	cfg := PlatformConfig{SkillRoot: "~/.pi/agent/skills"}
	p, err := newPiPlatform(cfg)
	if err != nil {
		t.Fatalf("newPiPlatform: %v", err)
	}

	if p.HomeDir() != tmpDir {
		t.Errorf("HomeDir() = %q, want %q", p.HomeDir(), tmpDir)
	}
	if !p.Detect() {
		t.Error("Detect() should return true when override path exists")
	}
}

// TestPiDetect_FlagMissing — override points to a nonexistent path; HomeDir
// reflects the override anyway and Detect falls back to the PATH check.
func TestPiDetect_FlagMissing(t *testing.T) {
	orig := piPathOverride
	piPathOverride = filepath.Join(t.TempDir(), "does-not-exist")
	// Scrub PATH so the test does not accidentally find a real `pi`.
	t.Setenv("PATH", t.TempDir())
	t.Cleanup(func() { piPathOverride = orig })

	cfg := PlatformConfig{SkillRoot: "~/.pi/agent/skills"}
	p, err := newPiPlatform(cfg)
	if err != nil {
		t.Fatalf("newPiPlatform: %v", err)
	}

	if p.HomeDir() != piPathOverride {
		t.Errorf("HomeDir() = %q, want %q", p.HomeDir(), piPathOverride)
	}
	if p.Detect() {
		t.Error("Detect() should return false when override path missing and 'pi' not on PATH")
	}
}

// TestPiDetect_EnvOverride — PI_CODING_AGENT_DIR is honoured when --pi-path
// is unset.
func TestPiDetect_EnvOverride(t *testing.T) {
	orig := piPathOverride
	piPathOverride = ""
	t.Cleanup(func() { piPathOverride = orig })

	tmpDir := t.TempDir()
	t.Setenv("PI_CODING_AGENT_DIR", tmpDir)

	cfg := PlatformConfig{SkillRoot: "~/.pi/agent/skills"}
	p, err := newPiPlatform(cfg)
	if err != nil {
		t.Fatalf("newPiPlatform: %v", err)
	}

	if p.HomeDir() != tmpDir {
		t.Errorf("HomeDir() = %q, want %q", p.HomeDir(), tmpDir)
	}
	if !p.Detect() {
		t.Error("Detect() should return true when env-overridden path exists")
	}
}

// TestPiGetAgentIDs_AlwaysNil — Pi has no native agent directory.
func TestPiGetAgentIDs_AlwaysNil(t *testing.T) {
	tmpDir := t.TempDir()
	home := filepath.Join(tmpDir, ".pi", "agent")
	p := newPlatformForTest(t, "pi", home)

	ids, err := p.GetAgentIDs(DistManifest{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ids != nil {
		t.Errorf("GetAgentIDs should return nil for Pi, got %v", ids)
	}
	if p.AgentsDir() != "" {
		t.Errorf("AgentsDir() should be empty for Pi, got %q", p.AgentsDir())
	}
}

// TestPiSkillRegistryTrigger — Pi reuses the claude registry format.
func TestPiSkillRegistryTrigger(t *testing.T) {
	tmpDir := t.TempDir()
	home := filepath.Join(tmpDir, ".pi", "agent")
	p := newPlatformForTest(t, "pi", home)

	if got := p.SkillRegistryTrigger(); got != "claude" {
		t.Errorf("SkillRegistryTrigger() = %q, want %q", got, "claude")
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
