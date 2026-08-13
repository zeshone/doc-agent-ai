package pipeline

import "testing"

func TestParseNodeClassifiesDepth(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantType  NodeType
		wantShort string
	}{
		{
			name:      "bare system name is a system node",
			raw:       "acme-hr",
			wantType:  NodeSystem,
			wantShort: "acme-hr",
		},
		{
			name:      "one segment below the system is a module",
			raw:       "acme-hr/payroll",
			wantType:  NodeModule,
			wantShort: "payroll",
		},
		{
			name:      "two segments below the system is a submodule",
			raw:       "acme-hr/payroll/tax",
			wantType:  NodeSubmodule,
			wantShort: "tax",
		},
		{
			name:      "surrounding whitespace is trimmed",
			raw:       "  acme-hr/payroll  ",
			wantType:  NodeModule,
			wantShort: "payroll",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := ParseNode(tt.raw)
			if err != nil {
				t.Fatalf("ParseNode(%q) returned error: %v", tt.raw, err)
			}
			if node.Type != tt.wantType {
				t.Errorf("node type = %q, want %q", node.Type, tt.wantType)
			}
			if node.ShortName != tt.wantShort {
				t.Errorf("short name = %q, want %q", node.ShortName, tt.wantShort)
			}
		})
	}
}

func TestParseNodeRejectsUnusableInput(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{name: "empty string", raw: ""},
		{name: "whitespace only", raw: "   "},
		{name: "deeper than submodule", raw: "acme-hr/payroll/tax/detail"},
		{name: "empty middle segment", raw: "acme-hr//tax"},
		{name: "trailing separator leaves an empty segment", raw: "acme-hr/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseNode(tt.raw); err == nil {
				t.Fatalf("ParseNode(%q) succeeded, want error", tt.raw)
			}
		})
	}
}

func TestCanonicalPhaseOrderMatchesTheDocumentedWorkflow(t *testing.T) {
	// skills/doc-arch/SKILL.md:125 — "Always: idea → rec → prd → refine → tech → [ddd] → pti"
	want := []PhaseID{PhaseIdea, PhaseRec, PhasePRD, PhaseRefine, PhaseTech, PhaseDDD, PhasePTI}

	got := CanonicalPhaseOrder()
	if len(got) != len(want) {
		t.Fatalf("phase count = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("phase[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestCanonicalPhaseOrderIsNotAliasedToCallers(t *testing.T) {
	// A caller mutating the returned slice must not corrupt the package order.
	first := CanonicalPhaseOrder()
	first[0] = "tampered"

	if second := CanonicalPhaseOrder(); second[0] != PhaseIdea {
		t.Fatalf("phase order was mutated by a caller: got %q, want %q", second[0], PhaseIdea)
	}
}

func TestParsePhaseAcceptsOnlyKnownPhases(t *testing.T) {
	for _, id := range CanonicalPhaseOrder() {
		t.Run("accepts "+string(id), func(t *testing.T) {
			got, err := ParsePhase(string(id))
			if err != nil {
				t.Fatalf("ParsePhase(%q) returned error: %v", id, err)
			}
			if got != id {
				t.Errorf("ParsePhase(%q) = %q, want %q", id, got, id)
			}
		})
	}

	for _, raw := range []string{"", "tech-specs", "idea ", "IDEA", "to-sdd", "feat"} {
		t.Run("rejects "+raw, func(t *testing.T) {
			if _, err := ParsePhase(raw); err == nil {
				t.Fatalf("ParsePhase(%q) succeeded, want error", raw)
			}
		})
	}
}
