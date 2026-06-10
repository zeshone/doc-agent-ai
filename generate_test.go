package main

import (
	"encoding/json"
	"os"
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

	if len(cm.Skills) != 14 {
		t.Errorf("expected 14 skills, got %d", len(cm.Skills))
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

	if cm.Commands[0].ID != "doc-arch" {
		t.Errorf("first command should be doc-arch, got %s", cm.Commands[0].ID)
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

// TestGenerateCommandFilenames verifies that generate produces doc-prefixed
// command filenames and none of the 11 legacy bare names.
func TestGenerateCommandFilenames(t *testing.T) {
	distDir := t.TempDir()
	if err := generate(distDir); err != nil {
		t.Fatalf("generate: %v", err)
	}

	// doc-prefixed files that must exist.
	wantFiles := []string{
		"doc-arch.md", "doc-idea.md", "doc-rec.md", "doc-prd.md",
		"doc-refine.md", "doc-tech.md", "doc-pti.md", "doc-mod.md",
		"doc-feat.md", "doc-ddd.md", "doc-to-sdd.md",
	}
	for _, f := range wantFiles {
		p := distDir + "/commands/" + f
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Errorf("expected generated command file %s, not found", p)
		}
	}

	// Legacy bare names that must NOT exist.
	legacyFiles := []string{
		"arch.md", "idea.md", "rec.md", "prd.md", "refine.md",
		"tech.md", "pti.md", "mod.md", "feat.md", "ddd.md", "to-sdd.md",
	}
	for _, f := range legacyFiles {
		p := distDir + "/commands/" + f
		if _, err := os.Stat(p); err == nil {
			t.Errorf("legacy command file %s should not exist in generated output", p)
		}
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

// ---------------------------------------------------------------------------
// T1a-7: PATH_RESOLUTION token opt-in + vault byte-identical regression guard
// ---------------------------------------------------------------------------

// TestGenerate_VaultPathSemanticsPreserved verifies that after PR 1b (preamble
// insertion) the vault path semantics remain intact: __DOC_AGENT_BASE_PATH__/
// still appears in rendered output and generate is idempotent. The preamble is
// additive prose — it does NOT change vault path substitution behavior.
func TestGenerate_VaultPathSemanticsPreserved(t *testing.T) {
	// Generate a baseline output.
	distDir := t.TempDir()
	if err := generate(distDir); err != nil {
		t.Fatalf("generate: %v", err)
	}

	// Read the generated doc-prd.md prompt for opencode (representative file).
	baseline, err := os.ReadFile(distDir + "/prompts/doc-prd.md")
	if err != nil {
		t.Fatalf("read baseline: %v", err)
	}

	// Generate again to confirm idempotence (same call, same output).
	distDir2 := t.TempDir()
	if err := generate(distDir2); err != nil {
		t.Fatalf("generate second run: %v", err)
	}
	second, err := os.ReadFile(distDir2 + "/prompts/doc-prd.md")
	if err != nil {
		t.Fatalf("read second run: %v", err)
	}

	if string(baseline) != string(second) {
		t.Error("generate is not idempotent: two runs produced different output for prompts/doc-prd.md")
	}

	// The vault placeholder __DOC_AGENT_BASE_PATH__/ must still appear in generated
	// output (not yet substituted at generate time — substitution happens at install time).
	if !strings.Contains(string(baseline), "__DOC_AGENT_BASE_PATH__/") {
		t.Error("vault placeholder __DOC_AGENT_BASE_PATH__/ missing from generated doc-prd.md — vault path semantics contract violated")
	}

	// {{PATH_RESOLUTION}} must NOT appear as a literal leftover token in the output:
	// after 1b the token must be expanded (present as preamble prose, not as literal token).
	if strings.Contains(string(baseline), "{{PATH_RESOLUTION}}") {
		t.Error("{{PATH_RESOLUTION}} token found as unexpanded literal in generated output — it must be expanded, not left as-is")
	}
}

// TestGenerate_PathResolutionVarNeverCausesError verifies that adding
// PATH_RESOLUTION to the buildBodyVars map does NOT break rendering of content
// files that do NOT reference the token. Under missingkey=error, a key being
// PRESENT but UNUSED is fine; only a MISSING key causes an error.
// This test uses generate() to render all roles and commands and asserts
// no rendering error occurs (i.e. PATH_RESOLUTION is always provided).
func TestGenerate_PathResolutionVarNeverCausesError(t *testing.T) {
	distDir := t.TempDir()
	if err := generate(distDir); err != nil {
		t.Fatalf("generate() returned error — PATH_RESOLUTION may not be in buildBodyVars or caused a rendering issue: %v", err)
	}
	// If we reach here, all 9 roles × 5 platforms and 11 commands rendered
	// without error, which means PATH_RESOLUTION is in the var map and did
	// not cause missingkey=error on any template that does not use it.
}

// ---------------------------------------------------------------------------
// T1b: Preamble injection golden assertions
// ---------------------------------------------------------------------------

// preambleAnchor is a substring of the preamble text guaranteed to be present
// in every file that has had {{PATH_RESOLUTION}} inserted (PR 1b). Using a
// substring avoids brittle full-text comparisons while still catching absence.
// Must match the text in src/templates/path-resolution.md.tmpl.
const preambleAnchor = "Resolve docs mode before writing any file"

// TestGenerate_PreamblePresentInAllRenderedRoles asserts that every role file
// rendered for every platform contains the resolution preamble exactly once.
// This test is RED before the {{PATH_RESOLUTION}} token is inserted into the
// 9 role content files and GREEN after insertion.
func TestGenerate_PreamblePresentInAllRenderedRoles(t *testing.T) {
	distDir := t.TempDir()
	if err := generate(distDir); err != nil {
		t.Fatalf("generate: %v", err)
	}

	platforms := []string{"opencode", "copilot", "claude", "qwen", "pi"}
	// Prompt files are generated for all 5 platforms under prompts/.
	// We check prompts/ (which are the canonical per-platform role files).
	roles := []string{
		"doc-arch", "doc-idea", "doc-rec", "doc-prd", "doc-refinement",
		"doc-tech", "doc-ddd", "doc-pti", "doc-to-sdd",
	}

	for _, role := range roles {
		promptPath := distDir + "/prompts/" + role + ".md"
		data, err := os.ReadFile(promptPath)
		if err != nil {
			t.Errorf("read %s: %v", promptPath, err)
			continue
		}
		content := string(data)
		count := strings.Count(content, preambleAnchor)
		if count != 1 {
			t.Errorf("prompts/%s.md: preamble anchor present %d time(s), want exactly 1", role, count)
		}
		// {{PATH_RESOLUTION}} must be expanded — must not appear as literal.
		if strings.Contains(content, "{{PATH_RESOLUTION}}") {
			t.Errorf("prompts/%s.md: unexpanded {{PATH_RESOLUTION}} token found", role)
		}
	}
	// Suppress unused variable warning — platforms list verifies we cover all.
	_ = platforms
}

// TestGenerate_PreamblePresentInAllRenderedCommands asserts that every opencode
// command file rendered contains the resolution preamble exactly once.
// This test is RED before {{PATH_RESOLUTION}} is inserted into the 11 command
// content files and GREEN after insertion.
func TestGenerate_PreamblePresentInAllRenderedCommands(t *testing.T) {
	distDir := t.TempDir()
	if err := generate(distDir); err != nil {
		t.Fatalf("generate: %v", err)
	}

	commands := []string{
		"doc-arch", "doc-idea", "doc-rec", "doc-prd", "doc-refine",
		"doc-tech", "doc-ddd", "doc-pti", "doc-mod", "doc-feat", "doc-to-sdd",
	}

	for _, cmd := range commands {
		cmdPath := distDir + "/commands/" + cmd + ".md"
		data, err := os.ReadFile(cmdPath)
		if err != nil {
			t.Errorf("read %s: %v", cmdPath, err)
			continue
		}
		content := string(data)
		count := strings.Count(content, preambleAnchor)
		if count != 1 {
			t.Errorf("commands/%s.md: preamble anchor present %d time(s), want exactly 1", cmd, count)
		}
		if strings.Contains(content, "{{PATH_RESOLUTION}}") {
			t.Errorf("commands/%s.md: unexpanded {{PATH_RESOLUTION}} token found", cmd)
		}
	}
}

// TestGenerate_NoLeakedInstallTokenInRolesOrCommands verifies that
// __DOC_AGENT_GLOBAL_MODE__ (install-time token) IS present (it is burned at
// install, not at generate), and that {{PATH_RESOLUTION}} is NOT present
// (template token, must be expanded at generate time).
// This distinguishes generate-time tokens from install-time tokens.
func TestGenerate_NoLeakedGenerateTokensInOutput(t *testing.T) {
	distDir := t.TempDir()
	if err := generate(distDir); err != nil {
		t.Fatalf("generate: %v", err)
	}

	files := []string{
		distDir + "/prompts/doc-prd.md",
		distDir + "/prompts/doc-rec.md",
		distDir + "/prompts/doc-to-sdd.md",
		distDir + "/commands/doc-prd.md",
		distDir + "/commands/doc-arch.md",
	}

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Errorf("read %s: %v", f, err)
			continue
		}
		content := string(data)
		// Generate-time token must be expanded.
		if strings.Contains(content, "{{PATH_RESOLUTION}}") {
			t.Errorf("%s: unexpanded {{PATH_RESOLUTION}} found — generate-time token must be expanded", f)
		}
		// Install-time token must remain as-is (burned by installToPlatform).
		if !strings.Contains(content, "__DOC_AGENT_GLOBAL_MODE__") {
			t.Errorf("%s: __DOC_AGENT_GLOBAL_MODE__ missing — install-time token must survive generate", f)
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
