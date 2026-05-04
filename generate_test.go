package main

import (
	"encoding/json"
	"os"
	"path/filepath"
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

	if len(cm.Skills) != 5 {
		t.Errorf("expected 5 skills, got %d", len(cm.Skills))
	}

	if len(cm.Roles) != 5 {
		t.Errorf("expected 5 roles, got %d", len(cm.Roles))
	}

	if cm.Roles[0].ID != "doc-arch" {
		t.Errorf("first role should be doc-arch, got %s", cm.Roles[0].ID)
	}

	if cm.Roles[0].Mode != "primary" {
		t.Errorf("doc-arch mode = %q, want 'primary'", cm.Roles[0].Mode)
	}

	if len(cm.Roles[0].CopilotChildren) != 4 {
		t.Errorf("doc-arch copilotChildren count = %d, want 4", len(cm.Roles[0].CopilotChildren))
	}

	if len(cm.Commands) != 6 {
		t.Errorf("expected 6 commands, got %d", len(cm.Commands))
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

// ---------------------------------------------------------------------------
// 7.3 Generate integration test
// ---------------------------------------------------------------------------

func TestGenerate_Integration(t *testing.T) {
	// Prerequisite: v2.0.0 dist/ must exist (generated by npm run generate).
	// Save the original working directory before changing it.
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot get working dir: %v", err)
	}

	canonicalDist := filepath.Join(origWd, "dist")
	if _, err := os.Stat(canonicalDist); os.IsNotExist(err) {
		t.Skip("canonical dist/ does not exist — run 'npm run generate' first")
	}

	// Generate into a temp dir
	tmpDir := t.TempDir()
	goDist := filepath.Join(tmpDir, "dist")

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir to temp dir: %v", err)
	}
	defer os.Chdir(origWd)

	if err := generate("dist"); err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	// Verify manifest.json structure
	compareManifestJSON(t, canonicalDist, goDist)

	// Verify all prompt files match
	compareDirFiles(t, canonicalDist, goDist, "prompts")
	compareDirFiles(t, canonicalDist, goDist, "prompts-claude")
	compareDirFiles(t, canonicalDist, goDist, "prompts-copilot")
	compareDirFiles(t, canonicalDist, goDist, "prompts-qwen")

	// Verify agent files match
	compareDirFiles(t, canonicalDist, goDist, "agents-claude")
	compareDirFiles(t, canonicalDist, goDist, "agents-copilot")
	compareDirFiles(t, canonicalDist, goDist, "agents-qwen")

	// Verify commands match
	compareDirFiles(t, canonicalDist, goDist, "commands")

	// Verify skills match (recursive)
	compareDirTrees(t, canonicalDist, goDist, "skills")
}

func compareManifestJSON(t *testing.T, canonicalDist, goDist string) {
	t.Helper()

	canonicalPath := filepath.Join(canonicalDist, "manifest.json")
	goPath := filepath.Join(goDist, "manifest.json")

	canonicalData, err := os.ReadFile(canonicalPath)
	if err != nil {
		t.Fatalf("read canonical manifest: %v", err)
	}
	goData, err := os.ReadFile(goPath)
	if err != nil {
		t.Fatalf("read go manifest: %v", err)
	}

	var canMap, goMap map[string]any
	if err := json.Unmarshal(canonicalData, &canMap); err != nil {
		t.Fatalf("parse canonical manifest: %v", err)
	}
	if err := json.Unmarshal(goData, &goMap); err != nil {
		t.Fatalf("parse go manifest: %v", err)
	}

	// Normalize generatedAt (timestamp will differ)
	canMap["generatedAt"] = "NORMALIZED"
	goMap["generatedAt"] = "NORMALIZED"

	canRe, _ := json.MarshalIndent(canMap, "", "  ")
	goRe, _ := json.MarshalIndent(goMap, "", "  ")

	if string(canRe) != string(goRe) {
		t.Errorf("manifest.json mismatch after normalizing generatedAt:\ncanonical:\n%s\n\ngo:\n%s",
			string(canRe), string(goRe))
	}
}

func compareDirFiles(t *testing.T, canonicalDist, goDist, dirName string) {
	t.Helper()

	canDir := filepath.Join(canonicalDist, dirName)
	goDir := filepath.Join(goDist, dirName)

	entries, err := os.ReadDir(canDir)
	if err != nil {
		t.Fatalf("read canonical %s: %v", dirName, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		canPath := filepath.Join(canDir, entry.Name())
		goPath := filepath.Join(goDir, entry.Name())

		canData, err := os.ReadFile(canPath)
		if err != nil {
			t.Errorf("read canonical %s: %v", canPath, err)
			continue
		}
		goData, err := os.ReadFile(goPath)
		if err != nil {
			t.Errorf("read go %s: %v", goPath, err)
			continue
		}

		if string(canData) != string(goData) {
			t.Errorf("%s mismatch:\ncanonical: %s\n\ngo: %s", entry.Name(), canPath, goPath)
		}
	}
}

func compareDirTrees(t *testing.T, canonicalDist, goDist, dirName string) {
	t.Helper()

	canDir := filepath.Join(canonicalDist, dirName)
	goDir := filepath.Join(goDist, dirName)

	err := filepath.WalkDir(canDir, func(canPath string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(canDir, canPath)
		goPath := filepath.Join(goDir, relPath)

		canData, err := os.ReadFile(canPath)
		if err != nil {
			t.Errorf("read canonical %s: %v", relPath, err)
			return nil
		}
		goData, err := os.ReadFile(goPath)
		if err != nil {
			t.Errorf("read go %s: %v", relPath, err)
			return nil
		}

		if string(canData) != string(goData) {
			t.Errorf("%s/%s mismatch", dirName, relPath)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk canonical %s: %v", dirName, err)
	}
}
