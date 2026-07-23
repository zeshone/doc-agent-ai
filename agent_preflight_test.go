package docagent

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// agent-preflight-and-naming — Phase 1 (#3): doc-to-sdd mode-aware naming
//
// VAULT mode must prefix output filenames with `<system>_` (system-level) or
// `<system>_<module>_` (feature/module-level). IN-PROJECT mode must keep the
// bare `_sdd-context.md` / `_sdd-tech-context.md` names UNCHANGED. All three
// content trees (skill, role, command) must document both forms.
// ---------------------------------------------------------------------------

// TestDocToSddSkillMD_ModeAwareNaming verifies skills/doc-to-sdd/SKILL.md
// documents vault-prefixed filenames (system-level and feature/module-level)
// and keeps the in-project bare form.
func TestDocToSddSkillMD_ModeAwareNaming(t *testing.T) {
	data, err := embedded.ReadFile("skills/doc-to-sdd/SKILL.md")
	if err != nil {
		t.Fatalf("cannot read SKILL.md: %v", err)
	}
	content := string(data)

	required := []string{
		"<system>_sdd-context.md",
		"<system>_sdd-tech-context.md",
		"<system>_<module>_sdd-context.md",
		"<system>_<module>_sdd-tech-context.md",
		"_sdd-context.md` (unchanged, bare)",
		"_sdd-tech-context.md` (unchanged, bare)",
	}
	for _, r := range required {
		if !strings.Contains(content, r) {
			t.Errorf("SKILL.md missing mode-aware naming reference: %q", r)
		}
	}
}

// TestDocToSddRoleMD_ModeAwareWritePaths verifies
// src/content/roles/doc-to-sdd.md Steps 2-4 branch write paths and the index
// block on resolved mode (vault prefixed vs in-project bare).
func TestDocToSddRoleMD_ModeAwareWritePaths(t *testing.T) {
	data, err := embedded.ReadFile("src/content/roles/doc-to-sdd.md")
	if err != nil {
		t.Fatalf("cannot read role file: %v", err)
	}
	content := string(data)

	required := []string{
		"mode-aware",
		"<system>_sdd-context.md",
		"<system>_sdd-tech-context.md",
		"<system>_<module>_sdd-context.md",
		"<system>_<module>_sdd-tech-context.md",
		"agent_sdd_context_project/_sdd-context.md` (bare, unchanged)",
		"agent_sdd_context_project/_sdd-tech-context.md` (bare, unchanged)",
	}
	for _, r := range required {
		if !strings.Contains(content, r) {
			t.Errorf("role file missing mode-aware write-path reference: %q", r)
		}
	}
}

// TestDocToSddCommandMD_ModeAwareOutput verifies
// src/content/commands/doc-to-sdd.md Output block documents both the vault
// prefixed and the in-project bare naming forms.
func TestDocToSddCommandMD_ModeAwareOutput(t *testing.T) {
	data, err := embedded.ReadFile("src/content/commands/doc-to-sdd.md")
	if err != nil {
		t.Fatalf("cannot read command file: %v", err)
	}
	content := string(data)

	required := []string{
		"Vault mode",
		"In-project mode",
		"<system>_sdd-context.md",
		"<system>_sdd-tech-context.md",
		"<system>_<module>_sdd-context.md",
		"<system>_<module>_sdd-tech-context.md",
		"_sdd-context.md` / `_sdd-tech-context.md` (bare, unchanged)",
	}
	for _, r := range required {
		if !strings.Contains(content, r) {
			t.Errorf("command file missing mode-aware output reference: %q", r)
		}
	}
}
