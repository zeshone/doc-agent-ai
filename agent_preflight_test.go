package docagent

import (
	pipelinepkg "github.com/zeshone/doc-agent-ai/internal/pipeline"
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
func TestSddOutputNamingIsModeAwareInTheProgram(t *testing.T) {
	// Naming used to be the role's job, spelled out as mode-aware write paths.
	// The program writes those files now, so the guard follows the responsibility:
	// asserting the prose still describes paths nobody follows would test nothing.
	tests := []struct {
		name          string
		env           pipelinepkg.Environment
		node          string
		wantBusiness  string
		wantTechnical string
	}{
		{
			name:          "vault prefixes with the node short name",
			env:           pipelinepkg.Environment{GlobalMode: pipelinepkg.ModeVault, GlobalBasePath: "/vault"},
			node:          "acme-hr",
			wantBusiness:  "acme-hr_sdd-context.md",
			wantTechnical: "acme-hr_sdd-tech-context.md",
		},
		{
			name:          "vault module level carries the module short name",
			env:           pipelinepkg.Environment{GlobalMode: pipelinepkg.ModeVault, GlobalBasePath: "/vault"},
			node:          "acme-hr/payroll",
			wantBusiness:  "payroll_sdd-context.md",
			wantTechnical: "payroll_sdd-tech-context.md",
		},
		{
			name:          "in-project keeps the bare form",
			env:           pipelinepkg.Environment{GlobalMode: pipelinepkg.ModeInProject, ProjectRoot: "/repo"},
			node:          "acme-hr",
			wantBusiness:  "_sdd-context.md",
			wantTechnical: "_sdd-tech-context.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := pipelinepkg.ParseNode(tt.node)
			if err != nil {
				t.Fatalf("ParseNode: %v", err)
			}
			res, err := pipelinepkg.Resolve(node, tt.env)
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if got := res.SDDOutputName(pipelinepkg.LayerBusiness); got != tt.wantBusiness {
				t.Errorf("business = %q, want %q", got, tt.wantBusiness)
			}
			if got := res.SDDOutputName(pipelinepkg.LayerTechnical); got != tt.wantTechnical {
				t.Errorf("technical = %q, want %q", got, tt.wantTechnical)
			}
		})
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

// ---------------------------------------------------------------------------
// agent-preflight-and-naming — Phase 2 (#2): brain-dump antechamber
//
// After the language question, doc-arch must invite a free unstructured
// brain-dump before any structured questions, then carry it forward to
// doc-idea verbatim with an explicit "do NOT re-ask what this covers" note.
// doc-idea must consume that dump: map it onto the 5 PO questions and ask
// only the gaps.
// ---------------------------------------------------------------------------

// TestDocArchSkillMD_BrainDumpAntechamber verifies skills/doc-arch/SKILL.md
// documents the free-form brain-dump step and the delegation note instructing
// doc-idea not to re-ask what the dump already covers.
func TestDocArchSkillMD_BrainDumpAntechamber(t *testing.T) {
	data, err := embedded.ReadFile("skills/doc-arch/SKILL.md")
	if err != nil {
		t.Fatalf("cannot read SKILL.md: %v", err)
	}
	content := string(data)

	required := []string{
		"Brain-Dump Antechamber",
		"done-cue",
		"do NOT re-ask what this covers",
	}
	for _, r := range required {
		if !strings.Contains(content, r) {
			t.Errorf("SKILL.md missing brain-dump antechamber reference: %q", r)
		}
	}
}

// TestDocArchRoleMD_BrainDumpAntechamber verifies
// src/content/roles/doc-arch.md documents the same brain-dump step and
// carry-forward delegation note.
func TestDocArchRoleMD_BrainDumpAntechamber(t *testing.T) {
	data, err := embedded.ReadFile("src/content/roles/doc-arch.md")
	if err != nil {
		t.Fatalf("cannot read role file: %v", err)
	}
	content := string(data)

	required := []string{
		"brain-dump antechamber",
		"done-cue",
		"do NOT re-ask what this covers",
	}
	for _, r := range required {
		if !strings.Contains(content, r) {
			t.Errorf("role file missing brain-dump antechamber reference: %q", r)
		}
	}
}

// TestDocIdeaSkillMD_ConsumesUpstreamDump verifies skills/doc-idea/SKILL.md
// Step 1 documents mapping an upstream brain-dump onto the 5 PO questions and
// asking only the gaps.
func TestDocIdeaSkillMD_ConsumesUpstreamDump(t *testing.T) {
	data, err := embedded.ReadFile("skills/doc-idea/SKILL.md")
	if err != nil {
		t.Fatalf("cannot read SKILL.md: %v", err)
	}
	content := string(data)

	required := []string{
		"upstream intake notes",
		"map its content onto the 5 PO questions",
		"asking only the",
	}
	for _, r := range required {
		if !strings.Contains(content, r) {
			t.Errorf("SKILL.md missing upstream dump consumption reference: %q", r)
		}
	}
}

// TestDocIdeaRoleMD_ConsumesUpstreamDump verifies
// src/content/roles/doc-idea.md steps 3-4 document the same dump-consumption
// behavior.
func TestDocIdeaRoleMD_ConsumesUpstreamDump(t *testing.T) {
	data, err := embedded.ReadFile("src/content/roles/doc-idea.md")
	if err != nil {
		t.Fatalf("cannot read role file: %v", err)
	}
	content := string(data)

	required := []string{
		"upstream intake notes",
		"5 PO questions",
		"ask only the",
	}
	for _, r := range required {
		if !strings.Contains(content, r) {
			t.Errorf("role file missing upstream dump consumption reference: %q", r)
		}
	}
}

// ---------------------------------------------------------------------------
// agent-preflight-and-naming — Phase 3 (#4): existing-project detection
//
// On startup with a project name, doc-arch must probe engram availability
// first (tool-absence or error treated as "unavailable"), fall back to a
// filesystem check, and — if found — make an advisory (never forced) offer
// to route into mod/doc-feat instead of re-documenting the whole system.
// ---------------------------------------------------------------------------

// TestDocArchSkillMD_ExistingProjectDetection verifies
// skills/doc-arch/SKILL.md documents the engram-probe-first,
// filesystem-fallback, and advisory offer sequence.
func TestDocArchSkillMD_ExistingProjectDetection(t *testing.T) {
	data, err := embedded.ReadFile("skills/doc-arch/SKILL.md")
	if err != nil {
		t.Fatalf("cannot read SKILL.md: %v", err)
	}
	content := string(data)

	required := []string{
		"Existing-Project Detection",
		"mem_search",
		"tool-absence or any error",
		"unavailable",
		"filesystem check",
		"advisory, but never forced",
		"mod <system> <module>",
		"doc-feat",
	}
	for _, r := range required {
		if !strings.Contains(content, r) {
			t.Errorf("SKILL.md missing existing-project detection reference: %q", r)
		}
	}
}

// TestDocArchRoleMD_ExistingProjectDetection verifies
// src/content/roles/doc-arch.md mirrors the same detection sequence.
func TestDocArchRoleMD_ExistingProjectDetection(t *testing.T) {
	data, err := embedded.ReadFile("src/content/roles/doc-arch.md")
	if err != nil {
		t.Fatalf("cannot read role file: %v", err)
	}
	content := string(data)

	required := []string{
		"existing-project detection",
		"mem_search",
		"tool is absent or returns an error",
		"unavailable",
		"filesystem check",
		"advisory, but never forced",
		"mod <system> <module>",
		"doc-feat",
	}
	for _, r := range required {
		if !strings.Contains(content, r) {
			t.Errorf("role file missing existing-project detection reference: %q", r)
		}
	}
}

// ---------------------------------------------------------------------------
// agent-preflight-and-naming — Phase 4 (#1): destination confirmation
//
// At each documentation start, doc-arch must show the resolved mode/
// destination and let the user confirm or change it. On change, it must
// write ONLY the `mode` field to `.doc-agent.json` at the project root,
// preserving all other keys. This finalizes the Phase 0 order: detection ->
// language -> destination confirm -> dump -> structured flow.
// ---------------------------------------------------------------------------

// TestDocArchSkillMD_DestinationConfirmation verifies
// skills/doc-arch/SKILL.md documents the destination confirm-or-change step
// and the mode-only `.doc-agent.json` write instruction.
func TestDocArchSkillMD_DestinationConfirmation(t *testing.T) {
	data, err := embedded.ReadFile("skills/doc-arch/SKILL.md")
	if err != nil {
		t.Fatalf("cannot read SKILL.md: %v", err)
	}
	content := string(data)

	required := []string{
		"Destination Confirmation",
		"marker.mode > global.mode > default vault",
		"confirm or change",
		".doc-agent.json",
		"ONLY the `mode` field",
		"preserving every other existing key",
		"Never re-ask destination",
	}
	for _, r := range required {
		if !strings.Contains(content, r) {
			t.Errorf("SKILL.md missing destination confirmation reference: %q", r)
		}
	}
}

// TestDocArchRoleMD_DestinationConfirmation verifies
// src/content/roles/doc-arch.md mirrors the same destination confirm-or-
// change sequence and finalizes the Phase 0 preflight ordering.
func TestDocArchRoleMD_DestinationConfirmation(t *testing.T) {
	data, err := embedded.ReadFile("src/content/roles/doc-arch.md")
	if err != nil {
		t.Fatalf("cannot read role file: %v", err)
	}
	content := string(data)

	required := []string{
		"destination confirmation",
		"marker.mode > global.mode > default vault",
		"confirm or change",
		".doc-agent.json",
		"ONLY the `mode` field",
		"preserving every other existing key",
		"Never re-ask",
		"detection → language question → destination confirmation → brain-dump antechamber → structured flow",
	}
	for _, r := range required {
		if !strings.Contains(content, r) {
			t.Errorf("role file missing destination confirmation reference: %q", r)
		}
	}
}

// ---------------------------------------------------------------------------
// agent-preflight-and-naming — PR2 review correction
//
// Fix 1: src/content/roles/doc-idea.md's reformulation step must fold a
// carried-forward brain-dump into the reformulation, matching
// skills/doc-idea/SKILL.md Step 1's "Fold the dump's content into the Step 4
// reformulation together with any gap answers."
//
// Fix 2: the doc-arch Phase 0 preflight (existing-project detection and the
// preflight order) must be explicitly scoped to the full orchestrator run
// (`arch`/`mod`), not to sub-agent commands (`rec`, `prd`, `tech`, etc.)
// invoked directly. Both trees (SKILL.md and role.md) must carry this
// scoping clause.
// ---------------------------------------------------------------------------

// TestDocIdeaRoleMD_ReformulationFoldsCarriedForwardDump verifies
// src/content/roles/doc-idea.md's reformulation step (Step 6) instructs
// folding a carried-forward brain-dump into the reformulation, matching
// skills/doc-idea/SKILL.md Step 1's parity requirement.
func TestDocIdeaRoleMD_ReformulationFoldsCarriedForwardDump(t *testing.T) {
	data, err := embedded.ReadFile("src/content/roles/doc-idea.md")
	if err != nil {
		t.Fatalf("cannot read role file: %v", err)
	}
	content := string(data)

	required := []string{
		"fold its content into this reformulation",
		"carried forward from the orchestrator",
	}
	for _, r := range required {
		if !strings.Contains(content, r) {
			t.Errorf("role file missing reformulation dump-fold reference: %q", r)
		}
	}
}

// TestDocArchSkillMD_Phase0ScopedToOrchestrator verifies
// skills/doc-arch/SKILL.md's Existing-Project Detection and Phase 0
// Preflight Order sections explicitly scope the preflight to the full
// `doc-arch` orchestrator run, not to sub-agent commands invoked directly.
func TestDocArchSkillMD_Phase0ScopedToOrchestrator(t *testing.T) {
	data, err := embedded.ReadFile("skills/doc-arch/SKILL.md")
	if err != nil {
		t.Fatalf("cannot read SKILL.md: %v", err)
	}
	content := string(data)

	required := []string{
		"applies only to the full `doc-arch` orchestrator run (`arch` / `mod`)",
		"not to sub-agent commands (`rec`, `prd`, `tech`, etc.) invoked directly",
	}
	if n := strings.Count(content, required[0]); n < 2 {
		t.Errorf("SKILL.md expected orchestrator-scoping clause in both Existing-Project Detection and Phase 0 Preflight Order sections, found %d occurrence(s)", n)
	}
	for _, r := range required {
		if !strings.Contains(content, r) {
			t.Errorf("SKILL.md missing Phase 0 orchestrator-scoping reference: %q", r)
		}
	}
}

// TestDocArchRoleMD_Phase0ScopedToOrchestrator verifies
// src/content/roles/doc-arch.md mirrors the same orchestrator-scoping clause
// in both the existing-project detection and Phase 0 preflight order
// sections.
func TestDocArchRoleMD_Phase0ScopedToOrchestrator(t *testing.T) {
	data, err := embedded.ReadFile("src/content/roles/doc-arch.md")
	if err != nil {
		t.Fatalf("cannot read role file: %v", err)
	}
	content := string(data)

	required := []string{
		"applies only to the full `doc-arch` orchestrator run (`arch` / `mod`)",
		"not to sub-agent commands (`rec`, `prd`, `tech`, etc.) invoked directly",
	}
	if n := strings.Count(content, required[0]); n < 2 {
		t.Errorf("role file expected orchestrator-scoping clause in both existing-project detection and Phase 0 preflight order sections, found %d occurrence(s)", n)
	}
	for _, r := range required {
		if !strings.Contains(content, r) {
			t.Errorf("role file missing Phase 0 orchestrator-scoping reference: %q", r)
		}
	}
}

// ---------------------------------------------------------------------------
// structured-option-selection — Phase 1: Global Agent Rule #13
//
// Closed-ended questions (small, fixed answer sets the agent already knows)
// must be presented through the host platform's native option-selection UI
// (arrow-key navigation plus Enter), falling back to a clearly enumerated
// list where no such widget exists. This governs FORM ONLY — the question
// stays mandatory and non-skippable. Open-ended questions (free product
// discovery, requirements elicitation, the brain-dump) are exempt.
//
// Rule #13's text lives only in skills/doc-arch/SKILL.md's Global Agent
// Rules list (single-tree — role.md has no such numbered list and reaches
// the rules via {{RULES_SKILL_PATH}}). doc-arch's 4 closed-ended question
// sections in BOTH trees must reference "Global Agent Rule #13". Two
// pre-existing voseo strings ("querés"/"Querés") are neutralized.
// ---------------------------------------------------------------------------

// TestDocArchSkillMD_Rule13OptionSelection verifies
// skills/doc-arch/SKILL.md's Global Agent Rules list defines Rule #13:
// closed-ended questions must use native option-selection UI with an
// enumerated-list fallback, form-only, never skippable, open-ended exempt.
func TestDocArchSkillMD_Rule13OptionSelection(t *testing.T) {
	data, err := embedded.ReadFile("skills/doc-arch/SKILL.md")
	if err != nil {
		t.Fatalf("cannot read SKILL.md: %v", err)
	}
	content := string(data)

	required := []string{
		"Closed-ended questions use the interactive-question tool",
		"`question` tool",
		"AskUserQuestion",
		"never plain chat text",
		"(Recommended)",
		"STOP and wait",
		"enumerated list",
		"mandatory and non-skippable",
		"Open-ended questions",
		"exempt",
	}
	for _, r := range required {
		if !strings.Contains(content, r) {
			t.Errorf("SKILL.md missing Rule #13 reference: %q", r)
		}
	}
}

// TestDocArchSkillMD_ClosedEndedSectionsReferenceRule13 verifies
// skills/doc-arch/SKILL.md's 4 closed-ended question sections (language,
// destination confirmation, existing-project offer, ddd yes/no) each
// reference "Global Agent Rule #13".
func TestDocArchSkillMD_ClosedEndedSectionsReferenceRule13(t *testing.T) {
	data, err := embedded.ReadFile("skills/doc-arch/SKILL.md")
	if err != nil {
		t.Fatalf("cannot read SKILL.md: %v", err)
	}
	content := string(data)

	if n := strings.Count(content, "Global Agent Rule #13"); n < 4 {
		t.Errorf("SKILL.md expected at least 4 references to \"Global Agent Rule #13\" (one per closed-ended section), found %d", n)
	}
}

// TestDocArchRoleMD_ClosedEndedSectionsReferenceRule13 verifies
// src/content/roles/doc-arch.md's 4 closed-ended question sections each
// reference "Global Agent Rule #13".
func TestDocArchRoleMD_ClosedEndedSectionsReferenceRule13(t *testing.T) {
	data, err := embedded.ReadFile("src/content/roles/doc-arch.md")
	if err != nil {
		t.Fatalf("cannot read role file: %v", err)
	}
	content := string(data)

	if n := strings.Count(content, "Global Agent Rule #13"); n < 4 {
		t.Errorf("role file expected at least 4 references to \"Global Agent Rule #13\" (one per closed-ended section), found %d", n)
	}
}

// TestDocArchRoleMD_LanguageQuestionNeutralSpanish verifies
// src/content/roles/doc-arch.md's language-choice Spanish example uses
// neutral Spanish ("quieres") instead of voseo ("querés"/"Querés").
func TestDocArchRoleMD_LanguageQuestionNeutralSpanish(t *testing.T) {
	data, err := embedded.ReadFile("src/content/roles/doc-arch.md")
	if err != nil {
		t.Fatalf("cannot read role file: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "¿En qué idioma quieres") {
		t.Errorf("role file missing neutral-Spanish language question: %q", "¿En qué idioma quieres")
	}
	if strings.Contains(strings.ToLower(content), "uerés") {
		t.Errorf("role file still contains voseo form (\"querés\"/\"Querés\")")
	}
}

// TestDocArchSkillMD_DddQuestionAsksInUsersLanguage verifies
// skills/doc-arch/SKILL.md's ddd yes/no prompt no longer hardcodes a
// language. It used to quote the question in neutral Spanish verbatim
// ("quieres", not voseo — see the prior TestDocArchSkillMD_DddQuestionNeutralSpanish),
// but a hardcoded output question ignores an English-speaking user's
// language entirely. The fix names the question and instructs the agent to
// ask it in whichever language the conversation is already in.
func TestDocArchSkillMD_DddQuestionAsksInUsersLanguage(t *testing.T) {
	data, err := embedded.ReadFile("skills/doc-arch/SKILL.md")
	if err != nil {
		t.Fatalf("cannot read SKILL.md: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "ask whether the user wants to document the database design, in the language the user is using") {
		t.Error("SKILL.md missing language-neutral ddd question instruction")
	}
	if strings.Contains(content, "¿Quieres documentar el diseño de la base de datos?") {
		t.Error("SKILL.md still hardcodes the Spanish ddd question verbatim — it must ask in the user's own language instead")
	}
}

// TestDocArchCommandMD_DddQuestionAsksInUsersLanguage verifies
// src/content/commands/doc-arch.md's ddd yes/no prompt no longer hardcodes a
// language, for the same reason as the SKILL.md counterpart above. This
// command file ships into the doc-arch command bundle, so its output must be
// language-neutral too — the skill and role trees are not the only
// user-facing surface.
func TestDocArchCommandMD_DddQuestionAsksInUsersLanguage(t *testing.T) {
	data, err := embedded.ReadFile("src/content/commands/doc-arch.md")
	if err != nil {
		t.Fatalf("cannot read command file: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "ask whether the user wants to document the database design, in the language the user is using") {
		t.Error("command file missing language-neutral ddd question instruction")
	}
	if strings.Contains(content, "¿Quieres documentar el diseño de la base de datos?") {
		t.Error("command file still hardcodes the Spanish ddd question verbatim — it must ask in the user's own language instead")
	}
}

// ---------------------------------------------------------------------------
// doc-to-sdd-unreachable-binary-guard — #90
//
// src/content/roles/doc-to-sdd.md is the one phase executor that submits to
// the binary from its own protocol (`sdd-commit`, not `commit-phase`) and
// carries no {{PIPELINE_PROTOCOL}} placeholder, so it never inherits the
// shared template's stop for a wrong invocation (exit 1) or an unreachable
// binary (no shell tool / not on PATH). Its "Step 5 — Submit" section
// documents only exit 0 (written) and exit 2 (refused), leaving a model that
// cannot reach the binary with a goal, an output path, and no rule telling
// it to stop instead of hand-writing the two context files.
// ---------------------------------------------------------------------------

// TestDocToSddRoleMD_UnreachableBinaryGuard verifies
// src/content/roles/doc-to-sdd.md's Step 5 — Submit section documents both
// an exit 1 branch (wrong invocation/environment: report it, do not retry
// blindly) and the unreachable-binary stop (no shell tool, or the binary is
// not on PATH: stop and say so, never fall back to writing the two context
// files by hand).
func TestDocToSddRoleMD_UnreachableBinaryGuard(t *testing.T) {
	data, err := embedded.ReadFile("src/content/roles/doc-to-sdd.md")
	if err != nil {
		t.Fatalf("cannot read role file: %v", err)
	}
	content := string(data)

	required := []string{
		"Exit `1`",
		"the invocation or the environment is wrong",
		"do not retry blindly",
		"cannot run the command at all",
		"the binary is not on `PATH`",
		"stop and say exactly that",
		"Do not fall back to writing the two context files by hand",
	}
	for _, r := range required {
		if !strings.Contains(content, r) {
			t.Errorf("role file missing unreachable-binary guard reference: %q", r)
		}
	}
}
