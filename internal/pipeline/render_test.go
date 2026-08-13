package pipeline

import (
	"strings"
	"testing"
)

// spanishSections builds Spanish prose for every topic a phase requires.
func spanishSections(t *testing.T, bank QuestionBank, phase PhaseID, nodeType NodeType) SectionInput {
	t.Helper()

	sections := map[string]string{}
	for _, topicID := range bank.RequiredTopics(phase, nodeType) {
		sections[topicID] = "Contenido redactado en español sobre " + topicID + "."
	}
	return SectionInput{SchemaName: SectionsSchema, Sections: sections}
}

func fullRecord(bank QuestionBank, phase PhaseID, nodeType NodeType, node Node) AnswerRecord {
	var answers []Answer
	for _, topicID := range bank.RequiredTopics(phase, nodeType) {
		answers = append(answers, Answer{
			TopicID:    topicID,
			Status:     AnswerAnswered,
			Source:     SourceUserAnswer,
			Verbatim:   "lo que dijo el usuario sobre " + topicID,
			CapturedAt: "2026-08-12T18:40:12Z",
		})
	}
	return AnswerRecord{
		SchemaName: AnswerRecordSchema,
		Node:       node.Raw,
		Phase:      phase,
		Answers:    answers,
	}
}

func TestRenderKeepsHeadingsEnglishAndProseInTheUsersLanguage(t *testing.T) {
	bank := mustLoadBank(t)
	node, err := ParseNode("acme-hr")
	if err != nil {
		t.Fatalf("ParseNode: %v", err)
	}
	spec, _ := bank.Phase(PhasePRD)

	rendered, err := Render(spec, node, bank,
		spanishSections(t, bank, PhasePRD, NodeSystem),
		fullRecord(bank, PhasePRD, NodeSystem, node))
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	text := string(rendered)

	// Structure is machine vocabulary: canonical English, always.
	for _, topic := range bank.TopicsFor(PhasePRD, NodeSystem) {
		if !strings.Contains(text, "## "+topic.Title+"\n") {
			t.Errorf("rendered artifact is missing heading %q", topic.Title)
		}
	}
	if !strings.Contains(text, "# acme-hr — Product Requirements Document") {
		t.Error("document title is missing or not canonical")
	}
	// Prose is the user's language, untouched.
	if !strings.Contains(text, "Contenido redactado en español sobre success-metrics.") {
		t.Error("Spanish prose was altered or dropped")
	}
}

func TestRenderFollowsBankDeclarationOrder(t *testing.T) {
	bank := mustLoadBank(t)
	node, err := ParseNode("acme-hr")
	if err != nil {
		t.Fatalf("ParseNode: %v", err)
	}
	spec, _ := bank.Phase(PhaseRec)

	rendered, err := Render(spec, node, bank,
		spanishSections(t, bank, PhaseRec, NodeSystem),
		fullRecord(bank, PhaseRec, NodeSystem, node))
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	text := string(rendered)

	previous := -1
	for _, topic := range bank.TopicsFor(PhaseRec, NodeSystem) {
		at := strings.Index(text, "## "+topic.Title+"\n")
		if at < 0 {
			t.Fatalf("heading %q is missing", topic.Title)
		}
		if at <= previous {
			t.Errorf("heading %q appears out of declaration order", topic.Title)
		}
		previous = at
	}
}

func TestRenderMarksADeferredTopicAsTBDInTheDocument(t *testing.T) {
	bank := mustLoadBank(t)
	node, err := ParseNode("acme-hr")
	if err != nil {
		t.Fatalf("ParseNode: %v", err)
	}
	spec, _ := bank.Phase(PhasePRD)

	record := fullRecord(bank, PhasePRD, NodeSystem, node)
	for i := range record.Answers {
		if record.Answers[i].TopicID == "risks-roadmap" {
			record.Answers[i].Status = AnswerDeferred
		}
	}

	input := spanishSections(t, bank, PhasePRD, NodeSystem)
	// A deferred topic owes no prose; the marker is the content.
	delete(input.Sections, "risks-roadmap")

	rendered, err := Render(spec, node, bank, input, record)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	text := string(rendered)

	// An acknowledged gap must be visible in the deliverable, not only in status.
	deferredAt := strings.Index(text, "## Risks and Roadmap")
	if deferredAt < 0 {
		t.Fatal("the deferred topic lost its section")
	}
	if !strings.Contains(text[deferredAt:], tbdMarker) {
		t.Error("deferred section carries no TBD marker")
	}
}

func TestRenderIncludesConditionalSectionsOnlyForTheRightNodeType(t *testing.T) {
	bank := mustLoadBank(t)
	spec, _ := bank.Phase(PhaseTech)

	tests := []struct {
		name        string
		raw         string
		wantPresent bool
	}{
		{name: "system owes no inheritance section", raw: "acme-hr", wantPresent: false},
		{name: "module owes an inheritance section", raw: "acme-hr/payroll", wantPresent: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := ParseNode(tt.raw)
			if err != nil {
				t.Fatalf("ParseNode: %v", err)
			}
			rendered, err := Render(spec, node, bank,
				spanishSections(t, bank, PhaseTech, node.Type),
				fullRecord(bank, PhaseTech, node.Type, node))
			if err != nil {
				t.Fatalf("Render: %v", err)
			}

			got := strings.Contains(string(rendered), "## Architecture Inheritance")
			if got != tt.wantPresent {
				t.Errorf("inheritance heading present = %v, want %v", got, tt.wantPresent)
			}
		})
	}
}

func TestRenderPlacesThePreambleBeforeTheFirstSection(t *testing.T) {
	bank := mustLoadBank(t)
	node, err := ParseNode("acme-hr")
	if err != nil {
		t.Fatalf("ParseNode: %v", err)
	}
	spec, _ := bank.Phase(PhaseIdea)

	input := spanishSections(t, bank, PhaseIdea, NodeSystem)
	input.Preamble = "Resumen ejecutivo del sistema."

	rendered, err := Render(spec, node, bank, input,
		fullRecord(bank, PhaseIdea, NodeSystem, node))
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	text := string(rendered)

	preambleAt := strings.Index(text, "Resumen ejecutivo del sistema.")
	firstHeadingAt := strings.Index(text, "## Target Users")
	if preambleAt < 0 {
		t.Fatal("preamble is missing")
	}
	if preambleAt > firstHeadingAt {
		t.Error("preamble was rendered after the first section")
	}
}

func TestRenderRefusesAnAnsweredTopicWithNoProse(t *testing.T) {
	// Defence in depth: Validate already rejects this, so reaching Render means a
	// caller bypassed validation. Emitting a hollow heading would look like
	// coverage, which is the failure mode this package exists to prevent.
	bank := mustLoadBank(t)
	node, err := ParseNode("acme-hr")
	if err != nil {
		t.Fatalf("ParseNode: %v", err)
	}
	spec, _ := bank.Phase(PhaseIdea)

	input := spanishSections(t, bank, PhaseIdea, NodeSystem)
	delete(input.Sections, "why-now")

	if _, err := Render(spec, node, bank, input,
		fullRecord(bank, PhaseIdea, NodeSystem, node)); err == nil {
		t.Fatal("Render emitted an artifact with a hollow section")
	}
}

func TestRenderRefusesAuditPhases(t *testing.T) {
	bank := mustLoadBank(t)
	node, err := ParseNode("acme-hr")
	if err != nil {
		t.Fatalf("ParseNode: %v", err)
	}
	spec, _ := bank.Phase(PhaseRefine)

	if _, err := Render(spec, node, bank, SectionInput{}, AnswerRecord{}); err == nil {
		t.Fatal("Render produced an artifact for an audit phase")
	}
}

func TestRenderedArtifactHasOneHeadingPerRequiredTopic(t *testing.T) {
	// Coverage and structure are the same truth measured on two surfaces: every
	// required topic has a place in the document where its answer lives.
	bank := mustLoadBank(t)
	node, err := ParseNode("acme-hr")
	if err != nil {
		t.Fatalf("ParseNode: %v", err)
	}

	for _, spec := range bank.Phases {
		if spec.Kind != KindInterview {
			continue
		}
		t.Run(string(spec.Phase), func(t *testing.T) {
			rendered, err := Render(spec, node, bank,
				spanishSections(t, bank, spec.Phase, NodeSystem),
				fullRecord(bank, spec.Phase, NodeSystem, node))
			if err != nil {
				t.Fatalf("Render: %v", err)
			}

			headings := strings.Count(string(rendered), "\n## ")
			required := len(bank.RequiredTopics(spec.Phase, NodeSystem))
			if headings != required {
				t.Errorf("rendered %d sections for %d required topics", headings, required)
			}
		})
	}
}
