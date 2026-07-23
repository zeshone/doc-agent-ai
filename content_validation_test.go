package docagent

import (
	"encoding/json"
	"io/fs"
	"strings"
	"testing"
)

// TestDocToSddSkillMD_Exists verifies the SKILL.md file exists in the embedded FS.
func TestDocToSddSkillMD_Exists(t *testing.T) {
	_, err := embedded.ReadFile("skills/doc-to-sdd/SKILL.md")
	if err != nil {
		t.Fatalf("skills/doc-to-sdd/SKILL.md not found in embedded FS: %v", err)
	}
}

// TestDocToSddSkillMD_RequiredSections verifies all 8 required sections from
// the design are present in SKILL.md.
func TestDocToSddSkillMD_RequiredSections(t *testing.T) {
	data, err := embedded.ReadFile("skills/doc-to-sdd/SKILL.md")
	if err != nil {
		t.Fatalf("cannot read SKILL.md: %v", err)
	}
	content := string(data)

	requiredSections := []string{
		"## Trigger / Positioning",
		"## Activation Contract",
		"## Exit Contract",
		"## Compaction Procedure",
		"## Output Format Specification",
		"## Partial Availability Rules",
		"## Error Table",
		"## Anti-Patterns",
	}

	for _, section := range requiredSections {
		if !strings.Contains(content, section) {
			t.Errorf("SKILL.md missing required section: %q", section)
		}
	}
}

// TestDocToSddSkillMD_CompactionRules verifies the NON-NEGOTIABLE compaction
// rules from the spec are embedded in the skill file.
func TestDocToSddSkillMD_CompactionRules(t *testing.T) {
	data, err := embedded.ReadFile("skills/doc-to-sdd/SKILL.md")
	if err != nil {
		t.Fatalf("cannot read SKILL.md: %v", err)
	}
	content := string(data)

	rules := []string{
		"MAXIMUM TOKEN EFFICIENCY",
		"CLARITY > TOKEN DENSITY",
	}
	for _, rule := range rules {
		if !strings.Contains(content, rule) {
			t.Errorf("SKILL.md missing compaction rule: %q", rule)
		}
	}
}

// TestDocToSddSkillMD_OutputSchemas verifies both output file schemas are
// present in the Output Format Specification section.
func TestDocToSddSkillMD_OutputSchemas(t *testing.T) {
	data, err := embedded.ReadFile("skills/doc-to-sdd/SKILL.md")
	if err != nil {
		t.Fatalf("cannot read SKILL.md: %v", err)
	}
	content := string(data)

	schemas := []string{
		"_sdd-context.md",
		"_sdd-tech-context.md",
		"agent_sdd_context_project",
	}
	for _, schema := range schemas {
		if !strings.Contains(content, schema) {
			t.Errorf("SKILL.md missing output schema reference: %q", schema)
		}
	}
}

// TestDocToSddSkillMD_KeepDropCriteria verifies KEEP and DROP criteria are defined.
func TestDocToSddSkillMD_KeepDropCriteria(t *testing.T) {
	data, err := embedded.ReadFile("skills/doc-to-sdd/SKILL.md")
	if err != nil {
		t.Fatalf("cannot read SKILL.md: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "KEEP") {
		t.Error("SKILL.md missing KEEP criteria")
	}
	if !strings.Contains(content, "DROP") {
		t.Error("SKILL.md missing DROP criteria")
	}
}

// TestDocToSddRoleMD_Exists verifies the role file exists.
func TestDocToSddRoleMD_Exists(t *testing.T) {
	_, err := embedded.ReadFile("src/content/roles/doc-to-sdd.md")
	if err != nil {
		t.Fatalf("src/content/roles/doc-to-sdd.md not found in embedded FS: %v", err)
	}
}

// TestDocToSddRoleMD_RequiredPlaceholders verifies the 3 required template
// placeholders are present in the role file.
func TestDocToSddRoleMD_RequiredPlaceholders(t *testing.T) {
	data, err := embedded.ReadFile("src/content/roles/doc-to-sdd.md")
	if err != nil {
		t.Fatalf("cannot read role file: %v", err)
	}
	content := string(data)

	requiredPlaceholders := []string{
		"{{SKILL_PATH}}",
		"{{RULES_SKILL_PATH}}",
		"{{BASE_PATH}}",
	}

	for _, ph := range requiredPlaceholders {
		if !strings.Contains(content, ph) {
			t.Errorf("role file missing placeholder: %q", ph)
		}
	}
}

// TestDocToSddRoleMD_WorkflowSteps verifies the role file contains the
// 5-step main workflow from the design.
func TestDocToSddRoleMD_WorkflowSteps(t *testing.T) {
	data, err := embedded.ReadFile("src/content/roles/doc-to-sdd.md")
	if err != nil {
		t.Fatalf("cannot read role file: %v", err)
	}
	content := string(data)

	workflowElements := []string{
		"Pre-flight",
		"_sdd-context.md",
		"_sdd-tech-context.md",
		"agent_sdd_context_project",
	}
	for _, elem := range workflowElements {
		if !strings.Contains(content, elem) {
			t.Errorf("role file missing workflow element: %q", elem)
		}
	}
}

// TestDocToSddRoleMD_ExecutorHeader verifies the standard executor header.
func TestDocToSddRoleMD_ExecutorHeader(t *testing.T) {
	data, err := embedded.ReadFile("src/content/roles/doc-to-sdd.md")
	if err != nil {
		t.Fatalf("cannot read role file: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "SDD Context Compactor executor") {
		t.Error("role file missing standard executor header")
	}
}

// TestDocToSddCommandMD_Exists verifies the command file exists.
func TestDocToSddCommandMD_Exists(t *testing.T) {
	_, err := embedded.ReadFile("src/content/commands/doc-to-sdd.md")
	if err != nil {
		t.Fatalf("src/content/commands/doc-to-sdd.md not found in embedded FS: %v", err)
	}
}

// TestDocToSddCommandMD_DelegatesToCorrectAgent verifies the command delegates
// directly to doc-to-sdd (NOT doc-arch).
func TestDocToSddCommandMD_DelegatesToCorrectAgent(t *testing.T) {
	data, err := embedded.ReadFile("src/content/commands/doc-to-sdd.md")
	if err != nil {
		t.Fatalf("cannot read command file: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "doc-to-sdd") {
		t.Error("command file does not reference doc-to-sdd agent")
	}
}

// TestDocToSddCommandMD_NotRoutedThroughOrchestrator verifies the command
// does NOT delegate through doc-arch.
func TestDocToSddCommandMD_NotRoutedThroughOrchestrator(t *testing.T) {
	data, err := embedded.ReadFile("src/content/commands/doc-to-sdd.md")
	if err != nil {
		t.Fatalf("cannot read command file: %v", err)
	}
	content := string(data)

	// The command file should reference doc-to-sdd directly, not doc-arch
	// as the delegate target. "doc-arch" may appear in context but must NOT
	// be the delegation target.
	if strings.Contains(content, "Delegate to the `doc-arch`") {
		t.Error("command file should delegate to doc-to-sdd directly, NOT doc-arch")
	}
}

// TestDocToSddCommandMD_StandalonePositioning verifies the command is marked
// as standalone and not part of the arch flow.
func TestDocToSddCommandMD_StandalonePositioning(t *testing.T) {
	data, err := embedded.ReadFile("src/content/commands/doc-to-sdd.md")
	if err != nil {
		t.Fatalf("cannot read command file: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "$ARGUMENTS") {
		t.Error("command file missing $ARGUMENTS placeholder")
	}

	if !strings.Contains(content, "agent_sdd_context_project") {
		t.Error("command file missing output folder path")
	}
}

// --- Phase 3: Registration tests ---

// loadContentManifest is a test helper that reads and unmarshals content.json.
func loadContentManifest(t *testing.T) ContentManifest {
	t.Helper()
	data, err := embedded.ReadFile("src/manifests/content.json")
	if err != nil {
		t.Fatalf("cannot read content.json: %v", err)
	}
	var cm ContentManifest
	if err := json.Unmarshal(data, &cm); err != nil {
		t.Fatalf("cannot unmarshal content.json: %v", err)
	}
	return cm
}

// TestDocToSddRegistration_SkillListed verifies doc-to-sdd is in the skills array.
func TestDocToSddRegistration_SkillListed(t *testing.T) {
	cm := loadContentManifest(t)
	found := false
	for _, s := range cm.Skills {
		if s == "doc-to-sdd" {
			found = true
			break
		}
	}
	if !found {
		t.Error("doc-to-sdd not found in content.json skills array")
	}
}

// TestDocToSddRegistration_RoleEntry verifies the doc-to-sdd role is registered
// with correct properties: hidden subagent with direct tool access.
func TestDocToSddRegistration_RoleEntry(t *testing.T) {
	cm := loadContentManifest(t)

	var role *RoleConfig
	for i := range cm.Roles {
		if cm.Roles[i].ID == "doc-to-sdd" {
			role = &cm.Roles[i]
			break
		}
	}
	if role == nil {
		t.Fatal("doc-to-sdd role not found in content.json roles array")
	}

	if role.Content != "roles/doc-to-sdd.md" {
		t.Errorf("role content = %q, want %q", role.Content, "roles/doc-to-sdd.md")
	}
	if role.Skill != "doc-to-sdd" {
		t.Errorf("role skill = %q, want %q", role.Skill, "doc-to-sdd")
	}
	if role.RulesSkill != "doc-arch" {
		t.Errorf("role rulesSkill = %q, want %q", role.RulesSkill, "doc-arch")
	}
	if !role.Hidden {
		t.Error("role should be hidden")
	}
	if role.UserInvocable {
		t.Error("role should NOT be userInvocable")
	}
	if role.Mode != "subagent" {
		t.Errorf("role mode = %q, want %q", role.Mode, "subagent")
	}
	if !role.OpenCodeTools["edit"] || !role.OpenCodeTools["read"] || !role.OpenCodeTools["write"] {
		t.Error("role opencodeTools must include edit, read, write")
	}
}

// TestDocToSddRegistration_CommandEntry verifies the doc-to-sdd command is registered
// with direct routing to doc-to-sdd (NOT doc-arch).
func TestDocToSddRegistration_CommandEntry(t *testing.T) {
	cm := loadContentManifest(t)

	var cmd *CommandConfig
	for i := range cm.Commands {
		if cm.Commands[i].ID == "doc-to-sdd" {
			cmd = &cm.Commands[i]
			break
		}
	}
	if cmd == nil {
		t.Fatal("doc-to-sdd command not found in content.json commands array")
	}

	if cmd.Agent != "doc-to-sdd" {
		t.Errorf("command agent = %q, want %q (direct routing, NOT doc-arch)", cmd.Agent, "doc-to-sdd")
	}
	if cmd.Content != "commands/doc-to-sdd.md" {
		t.Errorf("command content = %q, want %q", cmd.Content, "commands/doc-to-sdd.md")
	}
	if !strings.Contains(cmd.Description, "Standalone") {
		t.Error("command description should indicate standalone positioning")
	}
}

// TestContentManifest_LegacyCommandIds verifies the legacyCommandIds field in content.json
// contains all 11 bare command names that were renamed to doc-* prefix.
func TestContentManifest_LegacyCommandIds(t *testing.T) {
	cm := loadContentManifest(t)

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
}

// TestDocArchOpenCodeTools_HasQuestion guards that the doc-arch orchestrator
// can invoke OpenCode's interactive-question tool. doc-arch is the only
// mode:primary role — it runs the Phase 0 preflight and asks the closed-ended
// questions (language, destination, feature offer, ddd) that Global Agent
// Rule #13 requires be rendered via the native option-selection tool. OpenCode
// treats an explicit opencodeTools map as an allowlist, so without "question"
// the agent cannot call the tool and falls back to plain-text options.
func TestDocArchOpenCodeTools_HasQuestion(t *testing.T) {
	cm := loadContentManifest(t)

	var docArch *RoleConfig
	for i := range cm.Roles {
		if cm.Roles[i].ID == "doc-arch" {
			docArch = &cm.Roles[i]
			break
		}
	}
	if docArch == nil {
		t.Fatal("doc-arch role not found in content.json roles array")
	}
	if !docArch.OpenCodeTools["question"] {
		t.Errorf("doc-arch opencodeTools must include \"question\" so the orchestrator can use the interactive-question tool per Rule #13; got %v", docArch.OpenCodeTools)
	}
}

// TestDocArchCopilotChildren_CoversDocArchSubagents guards Copilot routing:
// every doc-arch-managed subagent role must be declared in doc-arch.copilotChildren.
func TestDocArchCopilotChildren_CoversDocArchSubagents(t *testing.T) {
	cm := loadContentManifest(t)

	var docArch *RoleConfig
	for i := range cm.Roles {
		if cm.Roles[i].ID == "doc-arch" {
			docArch = &cm.Roles[i]
			break
		}
	}
	if docArch == nil {
		t.Fatal("doc-arch role not found in content.json roles array")
	}

	children := make(map[string]bool, len(docArch.CopilotChildren))
	for _, id := range docArch.CopilotChildren {
		children[id] = true
	}

	for _, role := range cm.Roles {
		if role.Mode == "subagent" && role.RulesSkill == "doc-arch" {
			if !children[role.ID] {
				t.Errorf("doc-arch copilotChildren missing subagent %q", role.ID)
			}
		}
	}
}

// TestDocToSddRegistration_CompactRules verifies the compact rules block for
// doc-to-sdd exists in registryTemplate output with required rules.
func TestDocToSddRegistration_CompactRules(t *testing.T) {
	output := registryTemplate("/base", "/skills", "opencode")

	if !strings.Contains(output, "### doc-to-sdd") {
		t.Fatal("registryTemplate missing ### doc-to-sdd compact rules section")
	}

	requiredRules := []string{
		"Standalone command",
		"NOT part of the arch flow",
		"_sdd-context.md",
		"_sdd-tech-context.md",
		"agent_sdd_context_project",
		"English",
		"token efficiency",
	}
	for _, rule := range requiredRules {
		if !strings.Contains(output, rule) {
			t.Errorf("compact rules missing required text: %q", rule)
		}
	}
}

// TestDocToSddRegistration_RegistryRow verifies the User Skills table includes
// a doc-to-sdd row with correct skill path.
func TestDocToSddRegistration_RegistryRow(t *testing.T) {
	output := registryTemplate("/base", "/skills", "opencode")

	if !strings.Contains(output, "doc-to-sdd") {
		t.Fatal("registryTemplate missing doc-to-sdd in User Skills table")
	}
	if !strings.Contains(output, "/doc-to-sdd") {
		t.Error("registryTemplate User Skills row missing /doc-to-sdd trigger")
	}
}

// TestContentNoBareCommandReferences verifies that no agent-facing content file
// (src/content and skills) references the pre-v4 bare command names (/rec,
// /prd, ...) in prose or error messages. Commands were renamed to doc-* in
// v4.0.0; user-facing instructions must only mention the doc-* forms.
func TestContentNoBareCommandReferences(t *testing.T) {
	bareNames := []string{
		"arch", "idea", "rec", "prd", "refine",
		"tech", "pti", "mod", "feat", "ddd", "to-sdd",
	}

	for _, root := range []string{"src/content", "skills"} {
		err := fs.WalkDir(embedded, root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".md") {
				return nil
			}
			data, err := embedded.ReadFile(path)
			if err != nil {
				return err
			}
			content := string(data)
			for _, name := range bareNames {
				// Backtick-anchored command tokens: `/rec`, `/rec <args>`, `/rec<...
				for _, suffix := range []string{"`", " ", "<"} {
					token := "`/" + name + suffix
					if strings.Contains(content, token) {
						t.Errorf("%s contains bare command token %q — use `/doc-%s instead", path, token, name)
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
}
