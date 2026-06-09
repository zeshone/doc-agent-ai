package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// 7.1 Template golden-file tests
// ---------------------------------------------------------------------------

func TestRenderTemplate_Basic(t *testing.T) {
	raw := "Hello {{NAME}}!"
	vars := map[string]string{"NAME": "World"}

	result, err := renderTemplate(raw, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Hello World!"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRenderTemplate_MultipleVars(t *testing.T) {
	raw := "{{GREETING}}, {{NAME}}! Your id is {{ID}}."
	vars := map[string]string{
		"GREETING": "Hello",
		"NAME":     "Carlo",
		"ID":       "42",
	}

	result, err := renderTemplate(raw, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Hello, Carlo! Your id is 42."
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRenderTemplate_BoolText(t *testing.T) {
	// boolText is a plain function; test it directly.
	if bt := boolText(true); bt != "true" {
		t.Errorf("boolText(true) = %q, want %q", bt, "true")
	}
	if bt := boolText(false); bt != "false" {
		t.Errorf("boolText(false) = %q, want %q", bt, "false")
	}
}

func TestRenderTemplate_YamlList(t *testing.T) {
	items := []string{"Read", "Write", "Edit"}
	got := yamlList(items)
	want := "  - Read\n  - Write\n  - Edit"
	if got != want {
		t.Errorf("yamlList = %q, want %q", got, want)
	}
}

func TestRenderTemplate_YamlListEmpty(t *testing.T) {
	got := yamlList(nil)
	if got != "" {
		t.Errorf("yamlList(nil) = %q, want empty", got)
	}
	got = yamlList([]string{})
	if got != "" {
		t.Errorf("yamlList([]) = %q, want empty", got)
	}
}

func TestRenderTemplate_MissingVariable(t *testing.T) {
	raw := "Hello {{MISSING_VAR}}!"
	vars := map[string]string{"NAME": "World"}

	_, err := renderTemplate(raw, vars)
	if err == nil {
		t.Fatal("expected error for missing variable, got nil")
	}
	if !strings.Contains(err.Error(), "MISSING_VAR") {
		t.Errorf("error should mention the missing variable, got: %v", err)
	}
}

func TestRenderTemplate_YamlFrontmatter(t *testing.T) {
	// Simulate the agent template rendering pattern
	raw := "---\nname: {{NAME}}\ndescription: \"{{DESCRIPTION}}\"\n---\n\n{{BODY}}"
	vars := map[string]string{
		"NAME":        "doc-arch",
		"DESCRIPTION": "Orchestrator",
		"BODY":        "You are the orchestrator.",
	}

	result, err := renderTemplate(raw, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "name: doc-arch") {
		t.Errorf("expected 'name: doc-arch' in output, got: %s", result)
	}
	if !strings.Contains(result, "You are the orchestrator.") {
		t.Errorf("expected body in output, got: %s", result)
	}
}

// ---------------------------------------------------------------------------
// 7.2 Manifest struct unmarshal
// ---------------------------------------------------------------------------

func TestContentManifest_Unmarshal(t *testing.T) {
	data, err := embedded.ReadFile("src/manifests/content.json")
	if err != nil {
		t.Fatalf("cannot read embedded content.json: %v", err)
	}

	var cm ContentManifest
	if err := json.Unmarshal(data, &cm); err != nil {
		t.Fatalf("failed to unmarshal content.json: %v", err)
	}

	if cm.PlaceholderBasePath != "__DOC_AGENT_BASE_PATH__/" {
		t.Errorf("PlaceholderBasePath = %q", cm.PlaceholderBasePath)
	}

	if len(cm.Skills) != 13 {
		t.Errorf("expected 13 skills, got %d", len(cm.Skills))
	}

	if len(cm.Roles) != 9 {
		t.Errorf("expected 9 roles, got %d", len(cm.Roles))
	}

	if cm.Roles[0].ID != "doc-arch" {
		t.Errorf("first role should be doc-arch, got %s", cm.Roles[0].ID)
	}

	if cm.Roles[0].Mode != "primary" {
		t.Errorf("doc-arch mode = %q, want 'primary'", cm.Roles[0].Mode)
	}

	if len(cm.Roles[0].CopilotChildren) != 12 {
		t.Errorf("doc-arch copilotChildren count = %d, want 12", len(cm.Roles[0].CopilotChildren))
	}

	if len(cm.Commands) != 11 {
		t.Errorf("expected 11 commands, got %d", len(cm.Commands))
	}

	if cm.Commands[0].ID != "arch" {
		t.Errorf("first command should be arch, got %s", cm.Commands[0].ID)
	}

	// Verify opencodeTools are deserialized as map[string]bool
	if cm.Roles[0].OpenCodeTools == nil {
		t.Error("doc-arch should have opencodeTools")
	}
	if !cm.Roles[0].OpenCodeTools["bash"] {
		t.Error("doc-arch opencodeTools should include bash")
	}
}

// ---------------------------------------------------------------------------
// 7.3 Skill frontmatter lint
// ---------------------------------------------------------------------------

func TestLintSkillFrontmatter_QuotedValueWithColonIsValid(t *testing.T) {
	data := []byte("---\nname: foo\ndescription: 'Hello. Trigger: bar, baz.'\n---\n\nbody\n")
	if err := lintSkillFrontmatter("test.md", data); err != nil {
		t.Errorf("expected no error for quoted value, got: %v", err)
	}
}

func TestLintSkillFrontmatter_PlainValueIsValid(t *testing.T) {
	data := []byte("---\nname: foo\ndescription: Plain description without colons\n---\n\nbody\n")
	if err := lintSkillFrontmatter("test.md", data); err != nil {
		t.Errorf("expected no error for plain value, got: %v", err)
	}
}

func TestLintSkillFrontmatter_UnquotedColonSpaceIsRejected(t *testing.T) {
	data := []byte("---\nname: foo\ndescription: Hello. Trigger: bar\n---\n\nbody\n")
	err := lintSkillFrontmatter("test.md", data)
	if err == nil {
		t.Fatal("expected error for unquoted description containing ': ', got nil")
	}
	if !strings.Contains(err.Error(), "colon+space") {
		t.Errorf("error should mention 'colon+space', got: %v", err)
	}
	if !strings.Contains(err.Error(), "test.md:3") {
		t.Errorf("error should reference test.md:3, got: %v", err)
	}
}

func TestLintSkillFrontmatter_NoFrontmatterIsIgnored(t *testing.T) {
	data := []byte("# Just markdown\n\nWith content: that has colons but no frontmatter.\n")
	if err := lintSkillFrontmatter("test.md", data); err != nil {
		t.Errorf("expected no error for file without frontmatter, got: %v", err)
	}
}

func TestLintEmbeddedSkills_RepoStateIsClean(t *testing.T) {
	if err := lintEmbeddedSkills(); err != nil {
		t.Errorf("embedded skills/ should pass lint: %v", err)
	}
}

// TestLegacyCommandIds_FlowsToDistManifest verifies that ContentManifest carries
// legacyCommandIds and that generate copies the field into DistManifest.
func TestLegacyCommandIds_FlowsToDistManifest(t *testing.T) {
	// --- Part 1: ContentManifest carries the field after unmarshal ---
	data, err := embedded.ReadFile("src/manifests/content.json")
	if err != nil {
		t.Fatalf("cannot read embedded content.json: %v", err)
	}

	var cm ContentManifest
	if err := json.Unmarshal(data, &cm); err != nil {
		t.Fatalf("failed to unmarshal content.json: %v", err)
	}

	wantLegacy := []string{"arch", "idea", "rec", "prd", "refine", "tech", "pti", "mod", "feat", "ddd", "to-sdd"}

	if cm.LegacyCommandIds == nil {
		t.Fatal("ContentManifest.LegacyCommandIds is nil — field not defined or content.json missing legacyCommandIds")
	}
	if len(cm.LegacyCommandIds) != len(wantLegacy) {
		t.Fatalf("LegacyCommandIds length = %d, want %d; got %v", len(cm.LegacyCommandIds), len(wantLegacy), cm.LegacyCommandIds)
	}
	for i, id := range wantLegacy {
		if cm.LegacyCommandIds[i] != id {
			t.Errorf("LegacyCommandIds[%d] = %q, want %q", i, cm.LegacyCommandIds[i], id)
		}
	}

	// --- Part 2: generate copies the field into DistManifest ---
	distDir := t.TempDir()
	if err := generate(distDir); err != nil {
		t.Fatalf("generate: %v", err)
	}

	distData, err := os.ReadFile(distDir + "/manifest.json")
	if err != nil {
		t.Fatalf("cannot read generated manifest.json: %v", err)
	}

	var dm DistManifest
	if err := json.Unmarshal(distData, &dm); err != nil {
		t.Fatalf("cannot unmarshal generated manifest.json: %v", err)
	}

	if dm.LegacyCommandIds == nil {
		t.Fatal("DistManifest.LegacyCommandIds is nil — generate does not copy field")
	}
	if len(dm.LegacyCommandIds) != len(wantLegacy) {
		t.Fatalf("DistManifest.LegacyCommandIds length = %d, want %d", len(dm.LegacyCommandIds), len(wantLegacy))
	}
	for i, id := range wantLegacy {
		if dm.LegacyCommandIds[i] != id {
			t.Errorf("DistManifest.LegacyCommandIds[%d] = %q, want %q", i, dm.LegacyCommandIds[i], id)
		}
	}
}

func TestPlatformManifest_Unmarshal(t *testing.T) {
	data, err := embedded.ReadFile("src/manifests/platforms.json")
	if err != nil {
		t.Fatalf("cannot read embedded platforms.json: %v", err)
	}

	var pm PlatformManifest
	if err := json.Unmarshal(data, &pm); err != nil {
		t.Fatalf("failed to unmarshal platforms.json: %v", err)
	}

	// opencode: command-only platform, no agent support
	if pm.OpenCode.PromptDir != "prompts" {
		t.Errorf("opencode PromptDir = %q", pm.OpenCode.PromptDir)
	}
	if pm.OpenCode.CommandDir != "commands" {
		t.Errorf("opencode CommandDir = %q", pm.OpenCode.CommandDir)
	}
	if pm.OpenCode.AgentDir != "" {
		t.Error("opencode should have no AgentDir")
	}

	// copilot: has agent support
	if pm.Copilot.AgentDir != "agents-copilot" {
		t.Errorf("copilot AgentDir = %q", pm.Copilot.AgentDir)
	}
	if pm.Copilot.AgentExtension != ".agent.md" {
		t.Errorf("copilot AgentExtension = %q", pm.Copilot.AgentExtension)
	}
	if len(pm.Copilot.OrchestratorTools) == 0 {
		t.Error("copilot should have orchestratorTools")
	}

	// claude
	if pm.Claude.AgentExtension != ".md" {
		t.Errorf("claude AgentExtension = %q", pm.Claude.AgentExtension)
	}

	// qwen: has approval modes
	if pm.Qwen.ApprovalMode != "auto-edit" {
		t.Errorf("qwen ApprovalMode = %q", pm.Qwen.ApprovalMode)
	}
	if pm.Qwen.OrchestratorApprovalMode != "suggest" {
		t.Errorf("qwen OrchestratorApprovalMode = %q", pm.Qwen.OrchestratorApprovalMode)
	}
}
