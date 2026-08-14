package docagent

import (
	"os"
	"path/filepath"
	"regexp"
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

func TestNoRoleReadsTheArchetypeOutOfProse(t *testing.T) {
	// Real vaults phrase the same fact at least three ways — "Tipo de sistema",
	// "Arquetipo", "Archetype" — in two languages, with and without bold markers.
	// Any literal a role matched would fail on some real index, so the archetype
	// must come from the program.
	forbidden := []string{
		"verify `Arquetipo: Producto evolutivo`",
		"verify `Archetype: Evolving product`",
		"and verify `Arquetipo",
		"and verify `Archetype",
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
					t.Errorf("%s still matches archetype prose (%q); it must read target.archetype", role, phrase)
				}
			}
			// A role that gates on the archetype must gate on the recorded value.
			if strings.Contains(content, "supports modules") && !strings.Contains(content, "target.archetype") {
				t.Errorf("%s gates on the archetype without reading target.archetype", role)
			}
		})
	}
}

func TestDoctorIsDocumentedInTheHelp(t *testing.T) {
	// A command nobody can discover is a command that does not exist.
	raw, err := os.ReadFile(filepath.Join("cmd", "doc-agent-ai", "main.go"))
	if err != nil {
		t.Fatalf("cannot read main.go: %v", err)
	}
	content := string(raw)

	for _, phrase := range []string{"doctor", "--apply", "unverified"} {
		if !strings.Contains(content, phrase) {
			t.Errorf("the CLI help never mentions %q", phrase)
		}
	}
}

func TestProtocolTeachesClosedValueSets(t *testing.T) {
	// A topic the bank closed to a fixed set is refused without its `value`, so a
	// protocol that never mentions the field sends every new system straight into
	// a rejection on its first rec submission.
	raw, err := embedded.ReadFile("src/templates/pipeline-protocol.md.tmpl")
	if err != nil {
		t.Fatalf("cannot read protocol template: %v", err)
	}
	content := string(raw)

	for _, phrase := range []string{"`values`", "`value`", "closed to that set", "refused"} {
		if !strings.Contains(content, phrase) {
			t.Errorf("protocol template never explains closed value sets: missing %q", phrase)
		}
	}
	// The example record must actually show the field, not only describe it.
	if !strings.Contains(content, `"value": "<one of that topic's declared values>"`) {
		t.Error("the answer-record example does not show the value field")
	}
}

func TestProtocolRequiresTheQuestionAndOffersAnAlternative(t *testing.T) {
	// A rule that demands a prompt without naming the unasked-words alternative
	// pushes the agent to invent a question rather than reclassify the source.
	raw, err := embedded.ReadFile("src/templates/pipeline-protocol.md.tmpl")
	if err != nil {
		t.Fatalf("cannot read protocol template: %v", err)
	}
	content := string(raw)

	for _, phrase := range []string{
		"`prompt`",
		"`volunteered`",
		"Never invent a quote or a question",
		"unreadable on its own",
	} {
		if !strings.Contains(content, phrase) {
			t.Errorf("protocol template is missing %q", phrase)
		}
	}
	if !strings.Contains(content, `"prompt": "<the question you asked, in your own words>"`) {
		t.Error("the answer-record example does not show the prompt field")
	}
}

func TestProtocolTellsTheModelToReadTopicNotes(t *testing.T) {
	// A note nobody reads is decoration. The observed failure was two phases
	// asking the same question, so the protocol has to say what a note is for.
	raw, err := embedded.ReadFile("src/templates/pipeline-protocol.md.tmpl")
	if err != nil {
		t.Fatalf("cannot read protocol template: %v", err)
	}
	content := string(raw)

	for _, phrase := range []string{"`note`", "what it is **not**", "repetitive"} {
		if !strings.Contains(content, phrase) {
			t.Errorf("protocol template never explains topic notes: missing %q", phrase)
		}
	}
}

func TestRefineRoleKnowsCorrectingInvalidatesTheAudit(t *testing.T) {
	// Observed live: the PRD correction was committed and the audit that judged the
	// pre-correction stories stayed "complete". The role has to say that rewriting
	// the stories invalidates the verdicts, and that the anchor is not its to set.
	raw, err := embedded.ReadFile("src/content/roles/doc-refinement.md")
	if err != nil {
		t.Fatalf("cannot read refine role: %v", err)
	}
	content := string(raw)

	for _, phrase := range []string{
		"auditedRevision",
		"audit-stale",
		"Re-run the audit after correcting",
	} {
		if !strings.Contains(content, phrase) {
			t.Errorf("refine role is missing %q", phrase)
		}
	}
}

func TestRefineRoleDoesNotWriteItsOwnReport(t *testing.T) {
	// Real vaults hold refinement reports of several hundred lines, and the first
	// hardened run produced none. The report came back as a rendered view — so the
	// role must supply summary and notes, and must not author the file.
	raw, err := embedded.ReadFile("src/content/roles/doc-refinement.md")
	if err != nil {
		t.Fatalf("cannot read refine role: %v", err)
	}
	content := string(raw)

	for _, phrase := range []string{"`summary`", "`notes`", "_refinement.md", "Do **not** write that file yourself"} {
		if !strings.Contains(content, phrase) {
			t.Errorf("refine role is missing %q", phrase)
		}
	}
}

func TestEverySubcommandIsDocumentedInTheReadme(t *testing.T) {
	// A README naming a command that does not exist, or omitting one that does, is
	// the same failure the pipeline refuses elsewhere: an instruction the reader
	// cannot run, or a capability they never learn about.
	main, err := os.ReadFile(filepath.Join("cmd", "doc-agent-ai", "main.go"))
	if err != nil {
		t.Fatalf("cannot read main.go: %v", err)
	}
	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("cannot read README.md: %v", err)
	}

	dispatched := regexp.MustCompile(`case "([a-z][a-z-]+)":`).FindAllStringSubmatch(string(main), -1)
	if len(dispatched) == 0 {
		t.Fatal("found no subcommands in the dispatch; if the switch changed shape, update this guard")
	}

	for _, match := range dispatched {
		command := match[1]
		t.Run(command, func(t *testing.T) {
			if !strings.Contains(string(readme), "`"+command) {
				t.Errorf("subcommand %q is dispatched but never appears in README.md", command)
			}
		})
	}
}
