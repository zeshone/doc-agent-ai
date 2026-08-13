// Package pipeline holds the deterministic authority over the documentation
// workflow: which phase a node is in, whether a phase is complete, whether
// coverage exists, and what happens next.
//
// The split this package exists to enforce: the model routes (it decides what
// to ask, how to phrase it, when to dig deeper) and this package decides state.
// Nothing here consults conversation text, and nothing here trusts a claim the
// model makes about its own work. Coverage is counted from answer records on
// disk, never read from a checkbox the model wrote.
package pipeline

import (
	"fmt"
	"strings"
)

// PhaseID identifies one phase of the documentation workflow.
type PhaseID string

const (
	PhaseIdea   PhaseID = "idea"
	PhaseRec    PhaseID = "rec"
	PhasePRD    PhaseID = "prd"
	PhaseRefine PhaseID = "refine"
	PhaseTech   PhaseID = "tech"
	PhaseDDD    PhaseID = "ddd"
	PhasePTI    PhaseID = "pti"
)

// canonicalOrder is the workflow sequence from skills/doc-arch/SKILL.md:125:
// idea → rec → prd → refine → tech → [ddd] → pti.
//
// Off-chain commands (`feat`, `to-sdd`, standalone `refine`) are deliberately
// absent: they are not phases of a node's progression and carry no coverage.
var canonicalOrder = []PhaseID{
	PhaseIdea,
	PhaseRec,
	PhasePRD,
	PhaseRefine,
	PhaseTech,
	PhaseDDD,
	PhasePTI,
}

// CanonicalPhaseOrder returns the workflow order. The result is a copy, so a
// caller cannot reorder the pipeline for everyone else.
func CanonicalPhaseOrder() []PhaseID {
	out := make([]PhaseID, len(canonicalOrder))
	copy(out, canonicalOrder)
	return out
}

// ParsePhase accepts only an exact canonical phase id. It is deliberately
// strict: a near miss like "tech-specs" is an error rather than a guess,
// because guessing here is how a caller silently lands on the wrong phase.
func ParsePhase(raw string) (PhaseID, error) {
	for _, id := range canonicalOrder {
		if string(id) == raw {
			return id, nil
		}
	}
	return "", fmt.Errorf("unknown phase %q: expected one of %s", raw, joinPhases(canonicalOrder))
}

func joinPhases(ids []PhaseID) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = string(id)
	}
	return strings.Join(parts, ", ")
}

// PhaseState is the computed condition of one phase for one node.
type PhaseState string

const (
	// StateComplete means the artifact exists, the validator passed, and every
	// required topic has an answer on record.
	StateComplete PhaseState = "complete"
	// StateIncomplete means work is recorded but coverage or validation falls short.
	StateIncomplete PhaseState = "incomplete"
	// StatePending means not started, with prerequisites satisfied.
	StatePending PhaseState = "pending"
	// StateBlocked means not started, with an earlier phase not yet complete.
	StateBlocked PhaseState = "blocked"
	// StateAdopted means the phase carries inherited documentation that predates
	// answer records. Coverage is UNVERIFIED and says so. It does not block
	// downstream phases, because requiring a re-interview of an already
	// documented system before any new module could start would be absurd.
	StateAdopted PhaseState = "adopted"
	// StateNotApplicable means the phase does not apply — an optional phase the
	// user declined, for example.
	StateNotApplicable PhaseState = "not-applicable"
	// StateUndetermined means the state could not be computed. It never degrades
	// to a more optimistic value: an unreadable docs root is not "pending".
	StateUndetermined PhaseState = "undetermined"
)

// NodeType is the depth of a documentation node.
type NodeType string

const (
	NodeSystem    NodeType = "system"
	NodeModule    NodeType = "module"
	NodeSubmodule NodeType = "submodule"
)

// Node is a parsed documentation target such as "acme-hr/payroll".
type Node struct {
	// Raw is the caller-supplied identifier, trimmed.
	Raw string
	// Type is the depth classification.
	Type NodeType
	// Segments holds the path parts, system first.
	Segments []string
	// ShortName is the last segment. Artifact filenames use this and never the
	// full path — see skills/doc-arch/SKILL.md:31.
	ShortName string
}

// maxNodeDepth caps the hierarchy at system → module → sub-module, per
// skills/doc-arch/SKILL.md:27 ("max 2 levels: module → sub-module").
const maxNodeDepth = 3

// ParseNode classifies a node identifier by depth.
func ParseNode(raw string) (Node, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return Node{}, fmt.Errorf("node identifier is empty")
	}

	segments := strings.Split(trimmed, "/")
	if len(segments) > maxNodeDepth {
		return Node{}, fmt.Errorf(
			"node %q is %d levels deep: the hierarchy allows at most system/module/sub-module",
			trimmed, len(segments))
	}
	for i, segment := range segments {
		if strings.TrimSpace(segment) == "" {
			return Node{}, fmt.Errorf("node %q has an empty segment at position %d", trimmed, i+1)
		}
		segments[i] = strings.TrimSpace(segment)
	}

	types := map[int]NodeType{1: NodeSystem, 2: NodeModule, 3: NodeSubmodule}

	return Node{
		Raw:       trimmed,
		Type:      types[len(segments)],
		Segments:  segments,
		ShortName: segments[len(segments)-1],
	}, nil
}

// System returns the root system segment.
func (n Node) System() string {
	if len(n.Segments) == 0 {
		return ""
	}
	return n.Segments[0]
}

// Parent returns the node one level up and whether one exists. A system node
// has no parent.
func (n Node) Parent() (Node, bool) {
	if len(n.Segments) < 2 {
		return Node{}, false
	}
	parent, err := ParseNode(strings.Join(n.Segments[:len(n.Segments)-1], "/"))
	if err != nil {
		return Node{}, false
	}
	return parent, true
}
