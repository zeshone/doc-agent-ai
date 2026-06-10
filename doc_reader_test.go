package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// doc-reader skill content validation (T3-1 RED tests)
// ---------------------------------------------------------------------------

// TestDocReaderSkillMD_Exists verifies the SKILL.md file exists in the embedded FS.
func TestDocReaderSkillMD_Exists(t *testing.T) {
	_, err := embedded.ReadFile("skills/doc-reader/SKILL.md")
	if err != nil {
		t.Fatalf("skills/doc-reader/SKILL.md not found in embedded FS: %v", err)
	}
}

// TestDocReaderSkillMD_CompactedContextOnly verifies the skill directs agents to
// use ONLY the compacted SDD context output directory.
func TestDocReaderSkillMD_CompactedContextOnly(t *testing.T) {
	data, err := embedded.ReadFile("skills/doc-reader/SKILL.md")
	if err != nil {
		t.Fatalf("cannot read skills/doc-reader/SKILL.md: %v", err)
	}
	content := string(data)

	required := []string{
		"agent_sdd_context_project",
		"_sdd-context.md",
		"_sdd-tech-context.md",
	}
	for _, r := range required {
		if !strings.Contains(content, r) {
			t.Errorf("doc-reader SKILL.md missing required reference: %q", r)
		}
	}
}

// TestDocReaderSkillMD_ExcludesNormalFlowArtifacts verifies the skill explicitly
// excludes normal-flow artifacts from agent context (spec F3 content contract).
func TestDocReaderSkillMD_ExcludesNormalFlowArtifacts(t *testing.T) {
	data, err := embedded.ReadFile("skills/doc-reader/SKILL.md")
	if err != nil {
		t.Fatalf("cannot read skills/doc-reader/SKILL.md: %v", err)
	}
	content := string(data)

	// The skill must mention these exclusions.
	excluded := []string{
		"_prd.md",
		"_tech-spec.md",
	}
	for _, ex := range excluded {
		if !strings.Contains(content, ex) {
			t.Errorf("doc-reader SKILL.md must reference excluded artifact %q", ex)
		}
	}
}

// TestDocReaderSkillMD_FallbackInstruction verifies the skill instructs the
// agent to suggest /doc-to-sdd when the context files do not exist yet.
func TestDocReaderSkillMD_FallbackInstruction(t *testing.T) {
	data, err := embedded.ReadFile("skills/doc-reader/SKILL.md")
	if err != nil {
		t.Fatalf("cannot read skills/doc-reader/SKILL.md: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "doc-to-sdd") {
		t.Error("doc-reader SKILL.md must reference /doc-to-sdd for fallback when context files are absent")
	}
}

// TestDocReaderSkillMD_ValidFrontmatter verifies the skill has required frontmatter
// fields per skill-creator conventions.
func TestDocReaderSkillMD_ValidFrontmatter(t *testing.T) {
	data, err := embedded.ReadFile("skills/doc-reader/SKILL.md")
	if err != nil {
		t.Fatalf("cannot read skills/doc-reader/SKILL.md: %v", err)
	}
	content := string(data)

	required := []string{
		"name: doc-reader",
		"description:",
		"license: Apache-2.0",
		"metadata:",
		"author:",
		"version:",
	}
	for _, r := range required {
		if !strings.Contains(content, r) {
			t.Errorf("doc-reader SKILL.md missing frontmatter field: %q", r)
		}
	}
}

// TestDocReaderSkillMD_IsEnglish verifies the skill content is in English
// (spot-check: no Spanish keywords that would fail the lang gate).
func TestDocReaderSkillMD_IsEnglish(t *testing.T) {
	data, err := embedded.ReadFile("skills/doc-reader/SKILL.md")
	if err != nil {
		t.Fatalf("cannot read skills/doc-reader/SKILL.md: %v", err)
	}
	content := string(data)

	// Must have English headings characteristic of LLM-first skills.
	englishMarkers := []string{
		"## Activation Contract",
		"## Hard Rules",
	}
	for _, m := range englishMarkers {
		if !strings.Contains(content, m) {
			t.Errorf("doc-reader SKILL.md missing English section heading: %q", m)
		}
	}
}

// TestConditionalSkillsSubsetOfSkills guards the manifest invariant: every ID
// in conditionalSkills must also exist in skills. The install loop iterates
// manifest.Skills and consults ConditionalSkills only to decide skipping — a
// conditional entry missing from skills would be silently never installed.
func TestConditionalSkillsSubsetOfSkills(t *testing.T) {
	data, err := embedded.ReadFile("src/manifests/content.json")
	if err != nil {
		t.Fatalf("read content.json: %v", err)
	}
	var manifest struct {
		Skills            []string `json:"skills"`
		ConditionalSkills []string `json:"conditionalSkills"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse content.json: %v", err)
	}

	skillSet := make(map[string]bool, len(manifest.Skills))
	for _, s := range manifest.Skills {
		skillSet[s] = true
	}
	for _, c := range manifest.ConditionalSkills {
		if !skillSet[c] {
			t.Errorf("conditionalSkills entry %q is not present in skills — it would never be installed", c)
		}
	}
}
