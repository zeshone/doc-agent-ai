package pipeline

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// AdoptionSchema is the versioned name of the adoption contract.
//
// Adoption exists for documentation written before answer records did. Such a
// document may be complete and good, but its coverage cannot be verified: the
// user's words from an interview months ago exist nowhere. Adoption records that
// honestly — it never claims coverage, it claims inheritance.
const AdoptionSchema = "docagent.adoption/v1"

// Archetype values. Closed on purpose: the same fact appears in real vaults as
// "Tipo de sistema", "Arquetipo" and "Archetype", in two languages, with and
// without bold markers. Storing a value ends the guessing.
const (
	ArchetypeBounded  = "bounded"
	ArchetypeEvolving = "evolving"
)

// AdoptedPhase is one inherited phase and the evidence behind adopting it.
type AdoptedPhase struct {
	// Artifact is the file that made this phase adoptable.
	Artifact string `json:"artifact"`
	// Evidence states why it was adopted rather than counted.
	Evidence string `json:"evidence"`
}

// Adoption is the per-node record of inherited documentation.
type Adoption struct {
	SchemaName string                   `json:"schemaName"`
	Node       string                   `json:"node"`
	AdoptedAt  string                   `json:"adoptedAt"`
	Archetype  string                   `json:"archetype,omitempty"`
	Phases     map[PhaseID]AdoptedPhase `json:"phases"`
}

// Validate enforces the contract.
func (a Adoption) Validate(bank QuestionBank) error {
	if a.SchemaName != AdoptionSchema {
		return fmt.Errorf("adoption record declares schema %q, expected %q", a.SchemaName, AdoptionSchema)
	}
	if strings.TrimSpace(a.Node) == "" {
		return fmt.Errorf("adoption record has an empty node")
	}
	if a.Archetype != "" && !validArchetype(a.Archetype) {
		return fmt.Errorf("adoption record declares archetype %q, expected %q or %q",
			a.Archetype, ArchetypeBounded, ArchetypeEvolving)
	}
	for phase, adopted := range a.Phases {
		if _, ok := bank.Phase(phase); !ok {
			return fmt.Errorf("adoption record references unknown phase %q", phase)
		}
		if strings.TrimSpace(adopted.Artifact) == "" {
			return fmt.Errorf("adopted phase %q names no artifact", phase)
		}
		if strings.TrimSpace(adopted.Evidence) == "" {
			return fmt.Errorf("adopted phase %q records no evidence", phase)
		}
	}
	return nil
}

func validArchetype(value string) bool {
	return value == ArchetypeBounded || value == ArchetypeEvolving
}

// LoadAdoption reads the adoption record. Absence is the normal case: a node
// documented through the pipeline has nothing to adopt.
func LoadAdoption(path string, bank QuestionBank) (Adoption, bool, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Adoption{}, false, nil
	}
	if err != nil {
		return Adoption{}, false, fmt.Errorf("reading adoption record %s: %w", path, err)
	}
	var adoption Adoption
	if err := json.Unmarshal(raw, &adoption); err != nil {
		return Adoption{}, true, fmt.Errorf("parsing adoption record %s: %w", path, err)
	}
	if err := adoption.Validate(bank); err != nil {
		return Adoption{}, true, fmt.Errorf("adoption record %s is invalid: %w", path, err)
	}
	return adoption, true, nil
}

// inheritedArchetype walks up to the nearest ancestor that recorded an archetype.
// A module does not decide the shape of its system; it inherits it.
func inheritedArchetype(node Node, env Environment, bank QuestionBank) string {
	current := node
	for {
		parent, ok := current.Parent()
		if !ok {
			return ""
		}
		res, err := Resolve(parent, env)
		if err != nil {
			return ""
		}
		if adoption, _, err := LoadAdoption(res.AdoptionPath(), bank); err == nil && adoption.Archetype != "" {
			return adoption.Archetype
		}
		if recorded := recordedArchetype(res, bank); recorded != "" {
			return recorded
		}
		current = parent
	}
}

// sectionRevision fingerprints the authored prose of a phase.
//
// The stored section input is the anchor rather than the rendered artifact: the
// artifact also changes when a heading is renamed in the question bank, and a
// retitled section is not a rewritten story. An absent input yields an empty
// revision, which callers must read as "cannot compute" and never as a mismatch.
func sectionRevision(res Resolution, phase PhaseID) string {
	raw, err := os.ReadFile(res.SectionInputPath(phase))
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// recordedArchetype returns the archetype a node's own answers established, if
// any. This is the path for nodes documented through the pipeline: the archetype
// topic declares a closed value set, so the answer carries the value directly.
func recordedArchetype(res Resolution, bank QuestionBank) string {
	record, found, err := LoadAnswerRecord(res.AnswerRecordPath(PhaseRec), bank)
	if err != nil || !found {
		return ""
	}
	for _, answer := range record.Answers {
		if answer.TopicID == "archetype" && answer.Status == AnswerAnswered {
			return answer.Value
		}
	}
	return ""
}
