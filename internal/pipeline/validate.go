package pipeline

import (
	"fmt"
	"strings"
)

// ValidationSchema is the versioned name of the validation contract.
const ValidationSchema = "docagent.validation/v1"

// Check ids. Stable identifiers so callers can branch on a specific failure.
const (
	CheckAnswerRecordPresent   = "answer-record-present"
	CheckRequiredTopicsCovered = "required-topics-covered"
	CheckVerbatimProvenance    = "verbatim-provenance"
	CheckSectionContentPresent = "section-content-present"
	CheckSectionsKnown         = "sections-known"
	CheckAuditRecordPresent    = "audit-record-present"
	CheckAuditSubjectsComplete = "audit-subjects-complete"
)

// Outcomes for a single check and for the submission overall.
const (
	CheckPass         = "pass"
	CheckFail         = "fail"
	CheckUndetermined = "undetermined"

	ResultAccepted     = "accepted"
	ResultRejected     = "rejected"
	ResultUndetermined = "undetermined"
)

// Check is one validator outcome.
type Check struct {
	ID       string   `json:"id"`
	Result   string   `json:"result"`
	Detail   string   `json:"detail,omitempty"`
	TopicIDs []string `json:"topicIds,omitempty"`
}

// ValidationResult is the typed verdict on a submission.
type ValidationResult struct {
	SchemaName      string   `json:"schemaName"`
	Node            string   `json:"node"`
	Phase           PhaseID  `json:"phase"`
	Result          string   `json:"result"`
	Checks          []Check  `json:"checks"`
	RejectedBecause []string `json:"rejectedBecause,omitempty"`
}

// Accepted reports whether the submission may be written.
func (v ValidationResult) Accepted() bool { return v.Result == ResultAccepted }

// Submission is what a caller offers for one phase: the artifact content plus
// the record that proves its coverage. The program validates this and only then
// writes, so a rejected submission leaves nothing behind.
type Submission struct {
	Node  Node
	Phase PhaseID
	// Content is the per-topic prose the model authored. The program renders the
	// canonical headings around it, so there is no way to submit a finished
	// document whose structure disagrees with its coverage record.
	Content SectionInput
	// Answers is required for interview phases.
	Answers *AnswerRecord
	// Audit is required for audit phases.
	Audit *AuditRecord
}

// Validate checks a submission against the question bank.
//
// Structure is verified without depending on the documentation language: the
// headings are rendered by the program from canonical English titles in the
// bank, so this validator checks that every answered topic carries prose rather
// than trying to match heading text written in Spanish or English.
func Validate(sub Submission, bank QuestionBank) ValidationResult {
	result := ValidationResult{
		SchemaName: ValidationSchema,
		Node:       sub.Node.Raw,
		Phase:      sub.Phase,
	}

	spec, ok := bank.Phase(sub.Phase)
	if !ok {
		result.Result = ResultUndetermined
		result.Checks = append(result.Checks, Check{
			ID:     CheckAnswerRecordPresent,
			Result: CheckUndetermined,
			Detail: fmt.Sprintf("phase %q is not declared in the question bank", sub.Phase),
		})
		return result
	}

	if spec.Kind == KindAudit {
		result.Checks = append(result.Checks, auditChecks(sub, bank)...)
	} else {
		result.Checks = append(result.Checks, interviewChecks(sub, bank, spec)...)
	}
	result.Checks = append(result.Checks, sectionChecks(sub, bank, spec)...)

	for _, check := range result.Checks {
		switch check.Result {
		case CheckFail:
			result.RejectedBecause = append(result.RejectedBecause, check.ID)
		case CheckUndetermined:
			// An undetermined check never becomes an acceptance.
			result.RejectedBecause = append(result.RejectedBecause, check.ID)
		}
	}

	if len(result.RejectedBecause) == 0 {
		result.Result = ResultAccepted
	} else {
		result.Result = ResultRejected
	}
	return result
}

func interviewChecks(sub Submission, bank QuestionBank, spec PhaseSpec) []Check {
	if sub.Answers == nil {
		return []Check{{
			ID:     CheckAnswerRecordPresent,
			Result: CheckFail,
			Detail: fmt.Sprintf("phase %q is an interview and needs an answer record", spec.Phase),
		}}
	}

	if err := sub.Answers.Validate(bank); err != nil {
		// A malformed record cannot be counted, so provenance is unknown rather
		// than failed — the distinction matters when reporting to an operator.
		return []Check{
			{ID: CheckAnswerRecordPresent, Result: CheckFail, Detail: err.Error()},
			{ID: CheckVerbatimProvenance, Result: CheckUndetermined,
				Detail: "provenance cannot be counted while the record is invalid"},
		}
	}
	if sub.Answers.Phase != sub.Phase {
		return []Check{{
			ID:     CheckAnswerRecordPresent,
			Result: CheckFail,
			Detail: fmt.Sprintf("answer record is for phase %q but the submission is for %q",
				sub.Answers.Phase, sub.Phase),
		}}
	}
	if sub.Answers.Node != sub.Node.Raw {
		return []Check{{
			ID:     CheckAnswerRecordPresent,
			Result: CheckFail,
			Detail: fmt.Sprintf("answer record is for node %q but the submission is for %q",
				sub.Answers.Node, sub.Node.Raw),
		}}
	}

	checks := []Check{{ID: CheckAnswerRecordPresent, Result: CheckPass}}

	coverage := Coverage(bank, sub.Phase, sub.Node.Type, *sub.Answers)
	if coverage.Complete() {
		checks = append(checks, Check{
			ID:     CheckRequiredTopicsCovered,
			Result: CheckPass,
			Detail: fmt.Sprintf("%d answered, %d explicitly deferred, of %d required",
				len(coverage.Answered), len(coverage.Deferred), len(coverage.Required)),
		})
	} else {
		checks = append(checks, Check{
			ID:     CheckRequiredTopicsCovered,
			Result: CheckFail,
			Detail: fmt.Sprintf("%d of %d required topics have no recorded answer",
				len(coverage.Unanswered), len(coverage.Required)),
			TopicIDs: coverage.Unanswered,
		})
	}

	// Record-level validation already rejects an entry with no provenance, so
	// reaching here means every entry carries one. Reporting the count makes the
	// guarantee visible to whoever reads the result.
	attributed := 0
	for _, answer := range sub.Answers.Answers {
		if strings.TrimSpace(answer.Verbatim) != "" || strings.TrimSpace(answer.InheritedFrom) != "" {
			attributed++
		}
	}
	checks = append(checks, Check{
		ID:     CheckVerbatimProvenance,
		Result: CheckPass,
		Detail: fmt.Sprintf("%d of %d recorded answers carry a source; 0 unattributed",
			attributed, len(sub.Answers.Answers)),
	})

	return checks
}

func auditChecks(sub Submission, bank QuestionBank) []Check {
	if sub.Audit == nil {
		return []Check{{
			ID:     CheckAuditRecordPresent,
			Result: CheckFail,
			Detail: fmt.Sprintf("phase %q is an audit and needs an audit record", sub.Phase),
		}}
	}
	if err := sub.Audit.Validate(bank); err != nil {
		return []Check{
			{ID: CheckAuditRecordPresent, Result: CheckFail, Detail: err.Error()},
			{ID: CheckAuditSubjectsComplete, Result: CheckUndetermined,
				Detail: "subjects cannot be counted while the record is invalid"},
		}
	}
	if sub.Audit.Phase != sub.Phase {
		return []Check{{
			ID:     CheckAuditRecordPresent,
			Result: CheckFail,
			Detail: fmt.Sprintf("audit record is for phase %q but the submission is for %q",
				sub.Audit.Phase, sub.Phase),
		}}
	}

	return []Check{
		{ID: CheckAuditRecordPresent, Result: CheckPass},
		{ID: CheckAuditSubjectsComplete, Result: CheckPass,
			Detail: fmt.Sprintf("%d subjects carry a verdict for every criterion", len(sub.Audit.Subjects))},
	}
}

// sectionChecks verifies the artifact content: every answered required topic
// carries prose, and no section references a topic the bank does not declare.
func sectionChecks(sub Submission, bank QuestionBank, spec PhaseSpec) []Check {
	// An audit phase writes no artifact of its own: it evaluates one that already
	// exists, so it owes no sections.
	if spec.Kind == KindAudit {
		return []Check{{
			ID:     CheckSectionContentPresent,
			Result: CheckPass,
			Detail: "audit phase writes no artifact",
		}}
	}

	var checks []Check

	if unknown := unknownSections(spec, sub.Content); len(unknown) > 0 {
		checks = append(checks, Check{
			ID:       CheckSectionsKnown,
			Result:   CheckFail,
			Detail:   fmt.Sprintf("%d section(s) reference topics the bank does not declare", len(unknown)),
			TopicIDs: unknown,
		})
	} else {
		checks = append(checks, Check{ID: CheckSectionsKnown, Result: CheckPass})
	}

	// Coverage cannot be crossed with content without a record to read the
	// deferred topics from.
	if sub.Answers == nil {
		return append(checks, Check{
			ID:     CheckSectionContentPresent,
			Result: CheckUndetermined,
			Detail: "section content cannot be checked without an answer record",
		})
	}

	missing := missingSections(spec, sub.Node.Type, bank, sub.Content, *sub.Answers)
	if len(missing) > 0 {
		return append(checks, Check{
			ID:     CheckSectionContentPresent,
			Result: CheckFail,
			Detail: fmt.Sprintf(
				"%d answered topic(s) have no prose in the artifact", len(missing)),
			TopicIDs: missing,
		})
	}

	return append(checks, Check{
		ID:     CheckSectionContentPresent,
		Result: CheckPass,
		Detail: fmt.Sprintf("every required topic for a %s node carries prose or an explicit TBD",
			sub.Node.Type),
	})
}
