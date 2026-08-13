package pipeline

import "testing"

func TestEmbeddedQuestionBankLoads(t *testing.T) {
	bank, err := LoadQuestionBank()
	if err != nil {
		t.Fatalf("LoadQuestionBank() returned error: %v", err)
	}
	if bank.SchemaName != QuestionBankSchema {
		t.Errorf("schemaName = %q, want %q", bank.SchemaName, QuestionBankSchema)
	}
}

func TestQuestionBankCoversEveryCanonicalPhaseInOrder(t *testing.T) {
	bank, err := LoadQuestionBank()
	if err != nil {
		t.Fatalf("LoadQuestionBank() returned error: %v", err)
	}

	want := CanonicalPhaseOrder()
	if len(bank.Phases) != len(want) {
		t.Fatalf("bank declares %d phases, want %d", len(bank.Phases), len(want))
	}
	for i, spec := range bank.Phases {
		if spec.Phase != want[i] {
			t.Errorf("bank phase[%d] = %q, want %q", i, spec.Phase, want[i])
		}
	}
}

func TestQuestionBankPhaseShapes(t *testing.T) {
	bank, err := LoadQuestionBank()
	if err != nil {
		t.Fatalf("LoadQuestionBank() returned error: %v", err)
	}

	tests := []struct {
		name         string
		phase        PhaseID
		wantKind     Kind
		wantOptional bool
		wantAudit    bool
	}{
		{
			name:     "idea is an interview",
			phase:    PhaseIdea,
			wantKind: KindInterview,
		},
		{
			// skills/doc-arch/SKILL.md:167 — refine audits INVEST, it elicits nothing.
			name:      "refine is an audit with an audit rule and no topics",
			phase:     PhaseRefine,
			wantKind:  KindAudit,
			wantAudit: true,
		},
		{
			// skills/doc-arch/SKILL.md:127-129 — ddd is asked, never forced.
			name:         "ddd is an optional interview",
			phase:        PhaseDDD,
			wantKind:     KindInterview,
			wantOptional: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, ok := bank.Phase(tt.phase)
			if !ok {
				t.Fatalf("bank has no phase %q", tt.phase)
			}
			if spec.Kind != tt.wantKind {
				t.Errorf("kind = %q, want %q", spec.Kind, tt.wantKind)
			}
			if spec.Optional != tt.wantOptional {
				t.Errorf("optional = %v, want %v", spec.Optional, tt.wantOptional)
			}
			if (spec.AuditRule != nil) != tt.wantAudit {
				t.Errorf("has audit rule = %v, want %v", spec.AuditRule != nil, tt.wantAudit)
			}
			if tt.wantKind == KindAudit && len(spec.RequiredTopics) != 0 {
				t.Errorf("audit phase declares %d topics, want 0", len(spec.RequiredTopics))
			}
		})
	}
}

func TestRequiredTopicsFilterByNodeType(t *testing.T) {
	bank, err := LoadQuestionBank()
	if err != nil {
		t.Fatalf("LoadQuestionBank() returned error: %v", err)
	}

	// skills/doc-arch/SKILL.md:149 — only modules and submodules must declare
	// whether they inherit the parent architecture or diverge from it.
	tests := []struct {
		name        string
		nodeType    NodeType
		wantPresent bool
	}{
		{name: "system does not owe an inheritance decision", nodeType: NodeSystem, wantPresent: false},
		{name: "module owes an inheritance decision", nodeType: NodeModule, wantPresent: true},
		{name: "submodule owes an inheritance decision", nodeType: NodeSubmodule, wantPresent: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			topics := bank.RequiredTopics(PhaseTech, tt.nodeType)

			found := false
			for _, id := range topics {
				if id == "inheritance-mode" {
					found = true
				}
			}
			if found != tt.wantPresent {
				t.Errorf("inheritance-mode present = %v, want %v (topics: %v)", found, tt.wantPresent, topics)
			}
		})
	}
}

func TestRequiredTopicsAreUniqueWithinEveryPhase(t *testing.T) {
	bank, err := LoadQuestionBank()
	if err != nil {
		t.Fatalf("LoadQuestionBank() returned error: %v", err)
	}

	for _, spec := range bank.Phases {
		t.Run(string(spec.Phase), func(t *testing.T) {
			seen := map[string]bool{}
			for _, topic := range spec.RequiredTopics {
				if topic.ID == "" {
					t.Errorf("phase %q declares a topic with an empty id", spec.Phase)
					continue
				}
				if seen[topic.ID] {
					t.Errorf("phase %q declares topic %q more than once", spec.Phase, topic.ID)
				}
				seen[topic.ID] = true
			}
		})
	}
}

func TestEveryInterviewPhaseDeclaresAtLeastOneTopic(t *testing.T) {
	bank, err := LoadQuestionBank()
	if err != nil {
		t.Fatalf("LoadQuestionBank() returned error: %v", err)
	}

	for _, spec := range bank.Phases {
		if spec.Kind != KindInterview {
			continue
		}
		if len(spec.RequiredTopics) == 0 {
			t.Errorf("interview phase %q declares no required topics, so it can never block", spec.Phase)
		}
	}
}

func TestEveryPhaseDeclaresAnArtifactTemplate(t *testing.T) {
	bank, err := LoadQuestionBank()
	if err != nil {
		t.Fatalf("LoadQuestionBank() returned error: %v", err)
	}

	for _, spec := range bank.Phases {
		if spec.Artifact == "" {
			t.Errorf("phase %q declares no artifact template", spec.Phase)
		}
	}
}

func TestArtifactNameSubstitutesTheNodeShortName(t *testing.T) {
	bank, err := LoadQuestionBank()
	if err != nil {
		t.Fatalf("LoadQuestionBank() returned error: %v", err)
	}

	spec, ok := bank.Phase(PhasePRD)
	if !ok {
		t.Fatal("bank has no prd phase")
	}

	// skills/doc-arch/SKILL.md:31 — artifacts always use the node SHORT name,
	// never the full path.
	if got, want := spec.ArtifactName("payroll"), "payroll_prd.md"; got != want {
		t.Errorf("ArtifactName = %q, want %q", got, want)
	}
}
