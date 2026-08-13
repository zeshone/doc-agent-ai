package docagent

import (
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Prompt / program wiring guard
// ---------------------------------------------------------------------------
//
// The program only holds authority while the prompts actually call it. Before
// this wiring existed the skills had zero references to any subcommand and the
// orchestrator still told the model to mark `[x]` by hand, so every contract in
// internal/pipeline was inert. These guards fail if that regresses.

// phaseRoles maps each interview/audit phase to its role content file.
var phaseRoles = map[string]string{
	"idea":   "doc-idea",
	"rec":    "doc-rec",
	"prd":    "doc-prd",
	"refine": "doc-refinement",
	"tech":   "doc-tech",
	"ddd":    "doc-ddd",
	"pti":    "doc-pti",
}

func TestEveryPhaseRoleCarriesThePipelineProtocol(t *testing.T) {
	for phase, role := range phaseRoles {
		t.Run(phase, func(t *testing.T) {
			raw, err := embedded.ReadFile(filepath.ToSlash(filepath.Join("src/content/roles", role+".md")))
			if err != nil {
				t.Fatalf("cannot read role: %v", err)
			}
			content := string(raw)

			if !strings.Contains(content, "{{PIPELINE_PROTOCOL}}") {
				t.Errorf("%s does not reference {{PIPELINE_PROTOCOL}}, so it never learns to call the program", role)
			}
			if !strings.Contains(content, "commit-phase") {
				t.Errorf("%s never invokes commit-phase, so it would write the artifact itself", role)
			}
		})
	}
}

func TestNoRoleInstructsWritingStateByHand(t *testing.T) {
	// These are the exact instructions that made completion a model claim.
	forbidden := []string{
		"mark `[x]`",
		"Mark `[x]`",
		"Update the index file",
		"Update the master index",
		"Create or update the index file",
		"set status → `documented`",
		"Set status → `in progress`",
	}

	for phase, role := range phaseRoles {
		t.Run(phase, func(t *testing.T) {
			raw, err := embedded.ReadFile(filepath.ToSlash(filepath.Join("src/content/roles", role+".md")))
			if err != nil {
				t.Fatalf("cannot read role: %v", err)
			}
			content := string(raw)

			for _, phrase := range forbidden {
				if strings.Contains(content, phrase) {
					t.Errorf("%s still instructs %q — the program computes index state", role, phrase)
				}
			}
		})
	}
}

func TestOrchestratorRoutesOnTheProgram(t *testing.T) {
	role, err := embedded.ReadFile("src/content/roles/doc-arch.md")
	if err != nil {
		t.Fatalf("cannot read orchestrator role: %v", err)
	}
	if !strings.Contains(string(role), "{{PIPELINE_ROUTING}}") {
		t.Error("orchestrator role does not reference {{PIPELINE_ROUTING}}")
	}

	skill, err := embedded.ReadFile("skills/doc-arch/SKILL.md")
	if err != nil {
		t.Fatalf("cannot read orchestrator skill: %v", err)
	}
	content := string(skill)

	for _, phrase := range []string{
		"Every phase completion updates the corresponding checkbox",
		"checkboxes marked → status recalculated",
		"Recalculate automatically after each phase completion",
	} {
		if strings.Contains(content, phrase) {
			t.Errorf("orchestrator skill still claims %q — the program computes that now", phrase)
		}
	}
	for _, phrase := range []string{"nextRecommended", "blockedReasons"} {
		if !strings.Contains(content, phrase) {
			t.Errorf("orchestrator skill never mentions %q, so it has no routing rule to follow", phrase)
		}
	}
}

func TestPipelineTemplatesTeachProvenanceAndForbidHandWrittenHeadings(t *testing.T) {
	raw, err := embedded.ReadFile("src/templates/pipeline-protocol.md.tmpl")
	if err != nil {
		t.Fatalf("cannot read protocol template: %v", err)
	}
	content := string(raw)

	// The provenance rule is the load-bearing instruction: without it the model
	// has no reason to record the user's words rather than its own summary.
	for _, phrase := range []string{
		"Never invent a quote",
		"Do not paraphrase",
		"Do not write\nheadings",
		"docagent.answers/v1",
		"docagent.sections/v1",
	} {
		if !strings.Contains(content, phrase) {
			t.Errorf("protocol template is missing %q", phrase)
		}
	}
}

func TestGeneratedBundleExpandsThePipelinePlaceholders(t *testing.T) {
	bundle, err := BuildBundle()
	if err != nil {
		t.Fatalf("BuildBundle: %v", err)
	}

	var sawProtocol, sawRouting bool
	for path, data := range bundle.Files {
		content := string(data)
		for _, token := range []string{"{{PIPELINE_PROTOCOL}}", "{{PIPELINE_ROUTING}}"} {
			if strings.Contains(content, token) {
				t.Errorf("%s contains %s as an unexpanded literal token", path, token)
			}
		}
		// A distinctive line from each injected block proves the expansion landed.
		if strings.Contains(content, "you interview, the program decides") {
			sawProtocol = true
		}
		if strings.Contains(content, "read state from the program, never from the conversation") {
			sawRouting = true
		}
	}

	if !sawProtocol {
		t.Error("no generated file carries the expanded pipeline protocol")
	}
	if !sawRouting {
		t.Error("no generated file carries the expanded routing block")
	}
}
