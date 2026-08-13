package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// AuditRecordSchema is the versioned name of the audit-record contract. Audit
// phases evaluate subjects that already exist rather than eliciting answers, so
// they record verdicts per subject instead of topic coverage.
const AuditRecordSchema = "docagent.audit/v1"

// Verdict is the outcome of one audit criterion for one subject.
type Verdict string

const (
	VerdictPass Verdict = "pass"
	VerdictFail Verdict = "fail"
)

// AuditSubject is one audited item, such as a single user story.
type AuditSubject struct {
	// ID identifies the subject inside the source artifact.
	ID string `json:"id"`
	// Verdicts maps each criterion from the phase's audit rule to its outcome.
	Verdicts map[string]Verdict `json:"verdicts"`
	// Notes is optional free text explaining a failing verdict.
	Notes string `json:"notes,omitempty"`
}

// AuditRecord is the on-disk audit outcome for one node and audit phase.
type AuditRecord struct {
	SchemaName string         `json:"schemaName"`
	Node       string         `json:"node"`
	Phase      PhaseID        `json:"phase"`
	Subjects   []AuditSubject `json:"subjects"`
}

// Validate enforces the audit contract against the phase's declared rule.
//
// An audit with zero subjects is rejected: "I audited nothing" is exactly the
// fabricated-completeness shape this package exists to stop.
func (r AuditRecord) Validate(bank QuestionBank) error {
	if r.SchemaName != AuditRecordSchema {
		return fmt.Errorf("audit record declares schema %q, expected %q", r.SchemaName, AuditRecordSchema)
	}
	if strings.TrimSpace(r.Node) == "" {
		return fmt.Errorf("audit record has an empty node")
	}
	phase, err := ParsePhase(string(r.Phase))
	if err != nil {
		return fmt.Errorf("audit record phase is invalid: %w", err)
	}

	spec, ok := bank.Phase(phase)
	if !ok {
		return fmt.Errorf("question bank has no phase %q", phase)
	}
	if spec.Kind != KindAudit {
		return fmt.Errorf("phase %q is not an audit phase", phase)
	}
	if spec.AuditRule == nil {
		return fmt.Errorf("phase %q declares no audit rule", phase)
	}
	if len(r.Subjects) == 0 {
		return fmt.Errorf("audit record for phase %q contains no subjects", phase)
	}

	seen := map[string]bool{}
	for i, subject := range r.Subjects {
		if strings.TrimSpace(subject.ID) == "" {
			return fmt.Errorf("audit subject %d has an empty id", i+1)
		}
		if seen[subject.ID] {
			return fmt.Errorf("audit record contains subject %q more than once", subject.ID)
		}
		seen[subject.ID] = true

		for _, criterion := range spec.AuditRule.PerSubjectVerdict {
			verdict, present := subject.Verdicts[criterion]
			if !present {
				return fmt.Errorf("subject %q has no verdict for criterion %q", subject.ID, criterion)
			}
			if verdict != VerdictPass && verdict != VerdictFail {
				return fmt.Errorf(
					"subject %q has verdict %q for criterion %q: expected %q or %q",
					subject.ID, verdict, criterion, VerdictPass, VerdictFail)
			}
		}
		for criterion := range subject.Verdicts {
			if !containsString(spec.AuditRule.PerSubjectVerdict, criterion) {
				return fmt.Errorf(
					"subject %q records verdict for unknown criterion %q", subject.ID, criterion)
			}
		}
	}
	return nil
}

// LoadAuditRecord reads and validates an audit record. A missing file is a
// normal state reported through the found flag.
func LoadAuditRecord(path string, bank QuestionBank) (record AuditRecord, found bool, err error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return AuditRecord{}, false, nil
	}
	if err != nil {
		return AuditRecord{}, false, fmt.Errorf("reading audit record %s: %w", path, err)
	}
	if err := json.Unmarshal(raw, &record); err != nil {
		return AuditRecord{}, true, fmt.Errorf("parsing audit record %s: %w", path, err)
	}
	if err := record.Validate(bank); err != nil {
		return AuditRecord{}, true, fmt.Errorf("audit record %s is invalid: %w", path, err)
	}
	return record, true, nil
}

// DecisionsSchema is the versioned name of the optional-phase decision record.
const DecisionsSchema = "docagent.decisions/v1"

// OptionalPhaseDecision is what the user chose about an optional phase.
type OptionalPhaseDecision string

const (
	// DecisionAccepted means the user wants the optional phase.
	DecisionAccepted OptionalPhaseDecision = "accepted"
	// DecisionDeclined means the user does not want it. Persisting this is what
	// keeps the pipeline from re-asking across sessions.
	DecisionDeclined OptionalPhaseDecision = "declined"
)

// Decisions records choices about optional phases for one node.
type Decisions struct {
	SchemaName string                            `json:"schemaName"`
	Node       string                            `json:"node"`
	Optional   map[PhaseID]OptionalPhaseDecision `json:"optionalPhases"`
}

// Validate enforces the decisions contract.
func (d Decisions) Validate(bank QuestionBank) error {
	if d.SchemaName != DecisionsSchema {
		return fmt.Errorf("decisions record declares schema %q, expected %q", d.SchemaName, DecisionsSchema)
	}
	for phase, decision := range d.Optional {
		spec, ok := bank.Phase(phase)
		if !ok {
			return fmt.Errorf("decisions record references unknown phase %q", phase)
		}
		if !spec.Optional {
			return fmt.Errorf("phase %q is not optional, so it carries no decision", phase)
		}
		if decision != DecisionAccepted && decision != DecisionDeclined {
			return fmt.Errorf(
				"phase %q has decision %q: expected %q or %q",
				phase, decision, DecisionAccepted, DecisionDeclined)
		}
	}
	return nil
}

// LoadDecisions reads and validates the decisions record. Absence means no
// optional phase has been decided yet.
func LoadDecisions(path string, bank QuestionBank) (Decisions, bool, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Decisions{}, false, nil
	}
	if err != nil {
		return Decisions{}, false, fmt.Errorf("reading decisions %s: %w", path, err)
	}
	var decisions Decisions
	if err := json.Unmarshal(raw, &decisions); err != nil {
		return Decisions{}, true, fmt.Errorf("parsing decisions %s: %w", path, err)
	}
	if err := decisions.Validate(bank); err != nil {
		return Decisions{}, true, fmt.Errorf("decisions %s are invalid: %w", path, err)
	}
	return decisions, true, nil
}

func containsString(haystack []string, needle string) bool {
	for _, candidate := range haystack {
		if candidate == needle {
			return true
		}
	}
	return false
}
