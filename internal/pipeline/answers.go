package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// AnswerRecordSchema is the versioned name of the answer-record contract.
const AnswerRecordSchema = "docagent.answers/v1"

// AnswerStatus is what the record asserts about a topic.
type AnswerStatus string

const (
	// AnswerAnswered means the user supplied substance for the topic.
	AnswerAnswered AnswerStatus = "answered"
	// AnswerDeferred means the user explicitly said they do not know yet. It is
	// an acknowledged gap, permitted by global agent rule 8, and it counts as
	// coverage so the pipeline does not deadlock on genuine uncertainty — but it
	// stays visible in every status report.
	AnswerDeferred AnswerStatus = "deferred"
)

// AnswerSource is where the answer came from.
type AnswerSource string

const (
	// SourceBrainDump is the unstructured intake block the orchestrator collects
	// before the structured flow (skills/doc-arch/SKILL.md:109).
	SourceBrainDump AnswerSource = "brain-dump"
	// SourceUserAnswer is a direct reply to a question the agent asked. It carries
	// that question, because an answer without it can be unreadable: "yes, those
	// are the right ones" says nothing on its own about which ones.
	SourceUserAnswer AnswerSource = "user-answer"
	// SourceVolunteered is the user offering something nobody asked for, outside
	// the brain-dump. It exists so the agent never has to invent a question to
	// satisfy a required field — a fabricated prompt would be the same class of
	// defect this package exists to stop.
	SourceVolunteered AnswerSource = "volunteered"
	// SourceInheritedParent is context a module takes from its parent system
	// (skills/doc-arch/SKILL.md:147-149). It is the only source with no user
	// utterance to quote, and therefore the weakest one.
	SourceInheritedParent AnswerSource = "inherited-parent"
)

// Answer is one topic's recorded answer.
type Answer struct {
	TopicID string       `json:"topicId"`
	Status  AnswerStatus `json:"status"`
	Source  AnswerSource `json:"source"`
	// Verbatim is the user's own words. Required for every source except
	// inherited-parent: without it the entry is a claim rather than a record.
	Verbatim string `json:"verbatim,omitempty"`
	// Prompt is the question the agent asked, in its own words. Required for
	// user-answer and forbidden elsewhere. Without it a reader cannot judge
	// whether the answer fits the topic, which is exactly the audit the verbatim
	// is supposed to make possible.
	Prompt string `json:"prompt,omitempty"`
	// InheritedFrom points at the parent artifact section. Required for
	// inherited-parent and forbidden otherwise.
	InheritedFrom string `json:"inheritedFrom,omitempty"`
	// Value carries the machine-readable choice for a topic the bank closed to a
	// fixed set. It exists so a fact like the system archetype is known rather
	// than parsed out of prose that varies by author and language.
	Value      string `json:"value,omitempty"`
	CapturedAt string `json:"capturedAt"`
}

// AnswerRecord is the on-disk set of recorded answers for one node and phase.
type AnswerRecord struct {
	SchemaName string   `json:"schemaName"`
	Node       string   `json:"node"`
	Phase      PhaseID  `json:"phase"`
	Answers    []Answer `json:"answers"`
}

// Validate enforces the contract. It rejects rather than repairs: a record the
// program cannot trust must not silently become partial coverage.
func (r AnswerRecord) Validate(bank QuestionBank) error {
	if r.SchemaName != AnswerRecordSchema {
		return fmt.Errorf("answer record declares schema %q, expected %q", r.SchemaName, AnswerRecordSchema)
	}
	if strings.TrimSpace(r.Node) == "" {
		return fmt.Errorf("answer record has an empty node")
	}
	if _, err := ParsePhase(string(r.Phase)); err != nil {
		return fmt.Errorf("answer record phase is invalid: %w", err)
	}

	seen := map[string]bool{}
	for i, answer := range r.Answers {
		if err := answer.validate(bank, r.Phase); err != nil {
			return fmt.Errorf("answer %d (%s): %w", i+1, answer.TopicID, err)
		}
		if seen[answer.TopicID] {
			return fmt.Errorf("answer record contains topic %q more than once", answer.TopicID)
		}
		seen[answer.TopicID] = true
	}
	return nil
}

func (a Answer) validate(bank QuestionBank, phase PhaseID) error {
	if strings.TrimSpace(a.TopicID) == "" {
		return fmt.Errorf("topicId is empty")
	}
	if !bank.KnownTopic(phase, a.TopicID) {
		return fmt.Errorf(
			"topic %q is not declared for phase %q in the question bank", a.TopicID, phase)
	}

	switch a.Status {
	case AnswerAnswered, AnswerDeferred:
	default:
		return fmt.Errorf("status %q is not one of %q or %q", a.Status, AnswerAnswered, AnswerDeferred)
	}

	switch a.Source {
	case SourceBrainDump, SourceUserAnswer, SourceVolunteered:
		// Provenance rule: the user's own words are the record. This is what
		// makes a fabricated answer require a forged quote rather than a bare
		// assertion — the difference the whole design rests on.
		if strings.TrimSpace(a.Verbatim) == "" {
			return fmt.Errorf("source %q requires a non-empty verbatim span", a.Source)
		}
		if a.InheritedFrom != "" {
			return fmt.Errorf("source %q must not set inheritedFrom", a.Source)
		}
		// The question is half the exchange. An answer recorded without it can be
		// impossible to evaluate later, however genuine the words are.
		if a.Source == SourceUserAnswer {
			if strings.TrimSpace(a.Prompt) == "" {
				return fmt.Errorf(
					"source %q requires the prompt that was asked; use %q for words the user offered unasked",
					SourceUserAnswer, SourceVolunteered)
			}
		} else if a.Prompt != "" {
			return fmt.Errorf("source %q must not set prompt: nobody asked", a.Source)
		}
	case SourceInheritedParent:
		if strings.TrimSpace(a.InheritedFrom) == "" {
			return fmt.Errorf("source %q requires a non-empty inheritedFrom pointer", a.Source)
		}
		if a.Verbatim != "" {
			return fmt.Errorf(
				"source %q must not set verbatim: there is no user utterance to quote", a.Source)
		}
		if a.Prompt != "" {
			return fmt.Errorf("source %q must not set prompt: nobody was asked", a.Source)
		}
	default:
		return fmt.Errorf("source %q is not a recognised answer source", a.Source)
	}

	topic, _ := bank.Topic(phase, a.TopicID)
	switch {
	case len(topic.Values) == 0:
		if a.Value != "" {
			return fmt.Errorf("topic %q declares no closed value set, so value must be empty", a.TopicID)
		}
	case a.Status == AnswerDeferred:
		// A deferred topic is one the user does not know yet, so it owes no choice.
		if a.Value != "" {
			return fmt.Errorf("a deferred answer must not carry a value")
		}
	default:
		if a.Value == "" {
			return fmt.Errorf("topic %q requires one of %v in value", a.TopicID, topic.Values)
		}
		if !containsString(topic.Values, a.Value) {
			return fmt.Errorf("value %q for topic %q is not one of %v", a.Value, a.TopicID, topic.Values)
		}
	}

	if strings.TrimSpace(a.CapturedAt) == "" {
		return fmt.Errorf("capturedAt is empty")
	}
	if _, err := time.Parse(time.RFC3339, a.CapturedAt); err != nil {
		return fmt.Errorf("capturedAt %q is not an RFC3339 timestamp", a.CapturedAt)
	}
	return nil
}

// TopicCoverage is the counted answer state for one phase and node type.
type TopicCoverage struct {
	Required   []string `json:"required"`
	Answered   []string `json:"answered"`
	Deferred   []string `json:"deferred"`
	Unanswered []string `json:"unanswered"`
}

// Complete reports whether every required topic is accounted for, either
// answered or explicitly deferred.
func (c TopicCoverage) Complete() bool {
	return len(c.Unanswered) == 0
}

// Coverage counts a record against what the phase requires for a node type.
// It reads only the record: no artifact prose, no checkbox, no model claim.
//
// Topics present in the record but not required for this node type are ignored
// rather than rejected — extra information is not a defect.
func Coverage(bank QuestionBank, phase PhaseID, nodeType NodeType, record AnswerRecord) TopicCoverage {
	required := bank.RequiredTopics(phase, nodeType)

	status := make(map[string]AnswerStatus, len(record.Answers))
	for _, answer := range record.Answers {
		status[answer.TopicID] = answer.Status
	}

	coverage := TopicCoverage{
		Required:   required,
		Answered:   []string{},
		Deferred:   []string{},
		Unanswered: []string{},
	}
	for _, topicID := range required {
		switch status[topicID] {
		case AnswerAnswered:
			coverage.Answered = append(coverage.Answered, topicID)
		case AnswerDeferred:
			coverage.Deferred = append(coverage.Deferred, topicID)
		default:
			coverage.Unanswered = append(coverage.Unanswered, topicID)
		}
	}
	return coverage
}

// LoadAnswerRecord reads and validates a record from disk. A missing file is
// reported via the found flag rather than as an error: "no record yet" is a
// normal pipeline state, while "a record exists but is unreadable" is not.
func LoadAnswerRecord(path string, bank QuestionBank) (record AnswerRecord, found bool, err error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return AnswerRecord{}, false, nil
	}
	if err != nil {
		return AnswerRecord{}, false, fmt.Errorf("reading answer record %s: %w", path, err)
	}
	if err := json.Unmarshal(raw, &record); err != nil {
		return AnswerRecord{}, true, fmt.Errorf("parsing answer record %s: %w", path, err)
	}
	if err := record.Validate(bank); err != nil {
		return AnswerRecord{}, true, fmt.Errorf("answer record %s is invalid: %w", path, err)
	}
	return record, true, nil
}
