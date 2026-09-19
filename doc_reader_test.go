package docagent

import (
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
// use ONLY the paths the program returns for the compacted business and
// technical layers — the strict rule, restated for a program-routed skill
// rather than for one that names a directory itself.
func TestDocReaderSkillMD_CompactedContextOnly(t *testing.T) {
	data, err := embedded.ReadFile("skills/doc-reader/SKILL.md")
	if err != nil {
		t.Fatalf("cannot read skills/doc-reader/SKILL.md: %v", err)
	}
	content := string(data)

	required := []string{
		"sddContext.outputs",
		"business-layer",
		"technical-layer",
		"ONLY",
	}
	for _, r := range required {
		if !strings.Contains(content, r) {
			t.Errorf("doc-reader SKILL.md missing required reference: %q", r)
		}
	}
}

// TestDocReaderSkillMD_NoHardcodedDocsPath verifies the skill never bakes a
// filesystem location for the docs tree into its prose. Only the program
// knows where a node's documentation lives: in-project and vault modes
// resolve to different directories, and vault mode prefixes filenames with
// the node's short name, so any literal path or bare filename the skill
// hardcodes is wrong for at least one mode. This is the stronger invariant
// that replaces the old test pinning those exact hardcoded fragments.
func TestDocReaderSkillMD_NoHardcodedDocsPath(t *testing.T) {
	data, err := embedded.ReadFile("skills/doc-reader/SKILL.md")
	if err != nil {
		t.Fatalf("cannot read skills/doc-reader/SKILL.md: %v", err)
	}
	content := string(data)

	forbidden := []string{
		"docs/doc-agent",
		"agent_sdd_context_project",
		"_sdd-context.md",
		"_sdd-tech-context.md",
	}
	for _, f := range forbidden {
		if strings.Contains(content, f) {
			t.Errorf("doc-reader SKILL.md hardcodes docs-tree reference %q — only the program may resolve this path", f)
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
// agent to suggest /doc-to-sdd when the context is absent, and that it
// explicitly forbids falling back to the full docs tree instead.
func TestDocReaderSkillMD_FallbackInstruction(t *testing.T) {
	data, err := embedded.ReadFile("skills/doc-reader/SKILL.md")
	if err != nil {
		t.Fatalf("cannot read skills/doc-reader/SKILL.md: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "doc-to-sdd") {
		t.Error("doc-reader SKILL.md must reference /doc-to-sdd for fallback when context is absent")
	}
	if !strings.Contains(strings.ToLower(content), "fall back to") {
		t.Error("doc-reader SKILL.md must explicitly prohibit falling back to the full docs tree")
	}
}

// TestDocReaderSkillMD_UnreachableBinaryStop verifies the skill instructs the
// agent to stop rather than guess when it cannot run doc-agent-ai at all — no
// shell tool, or the binary missing from PATH. A guessed path in that
// situation reads somebody else's documentation, or nothing, with no way to
// tell which.
func TestDocReaderSkillMD_UnreachableBinaryStop(t *testing.T) {
	data, err := embedded.ReadFile("skills/doc-reader/SKILL.md")
	if err != nil {
		t.Fatalf("cannot read skills/doc-reader/SKILL.md: %v", err)
	}
	content := string(data)

	required := []string{
		"PATH",
		"shell tool",
		"stop",
	}
	for _, r := range required {
		if !strings.Contains(content, r) {
			t.Errorf("doc-reader SKILL.md missing unreachable-binary guidance: %q", r)
		}
	}
}

// TestDocReaderSkillMD_NodeScopedContext verifies the skill makes explicit
// that compacted context is scoped per node — a system and each of its
// modules/submodules carry independent context — and that the agent must ask
// a human rather than guess a node it does not already know. The program
// cannot enumerate a node's children, so a guessed node is the only
// alternative and it silently returns the wrong documentation.
func TestDocReaderSkillMD_NodeScopedContext(t *testing.T) {
	data, err := embedded.ReadFile("skills/doc-reader/SKILL.md")
	if err != nil {
		t.Fatalf("cannot read skills/doc-reader/SKILL.md: %v", err)
	}
	content := string(data)

	required := []string{
		"--node",
		"<system>/<module>",
		"own compacted context",
		"ask the human",
		"Never guess a node",
	}
	for _, r := range required {
		if !strings.Contains(content, r) {
			t.Errorf("doc-reader SKILL.md missing node-scoping guidance: %q", r)
		}
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
