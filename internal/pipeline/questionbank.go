package pipeline

import (
	"embed"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// QuestionBankSchema is the versioned name of the question-bank contract.
const QuestionBankSchema = "docagent.questionbank/v1"

// questionBankFile holds the embedded bank. It ships inside the binary so the
// authority travels with the installed tool and cannot drift per machine.
//
//go:embed questionbank.yaml
var questionBankFS embed.FS

const questionBankPath = "questionbank.yaml"

// Kind distinguishes phases that elicit information from phases that audit
// information already gathered.
type Kind string

const (
	// KindInterview phases gather answers and are measured by topic coverage.
	KindInterview Kind = "interview"
	// KindAudit phases evaluate an existing artifact and have no topics. Giving
	// them required topics would turn an audit into an interview.
	KindAudit Kind = "audit"
)

// AppliesWhen narrows a topic to a subset of node types.
type AppliesWhen struct {
	NodeType []NodeType `yaml:"nodeType,omitempty" json:"nodeType,omitempty"`
}

// Topic is one required interview topic.
type Topic struct {
	ID string `yaml:"id" json:"id"`
	// Title is the canonical English section heading for this topic. The program
	// renders it; the model never types it. That is what keeps document structure
	// verifiable regardless of the language the prose is written in.
	Title string `yaml:"title,omitempty" json:"title,omitempty"`
	// Values, when present, closes this topic to a fixed set. An answered entry
	// must carry one of them in its `value` field, which is what makes the fact
	// machine-readable instead of something to be inferred from prose.
	Values      []string     `yaml:"values,omitempty" json:"values,omitempty"`
	AppliesWhen *AppliesWhen `yaml:"appliesWhen,omitempty" json:"appliesWhen,omitempty"`
}

// appliesTo reports whether this topic is required for the given node type.
// A topic with no condition applies to every node type.
func (t Topic) appliesTo(nodeType NodeType) bool {
	if t.AppliesWhen == nil || len(t.AppliesWhen.NodeType) == 0 {
		return true
	}
	for _, candidate := range t.AppliesWhen.NodeType {
		if candidate == nodeType {
			return true
		}
	}
	return false
}

// AuditRule describes what an audit phase evaluates and the verdicts it must
// record per subject.
type AuditRule struct {
	Subject           string   `yaml:"subject" json:"subject"`
	SourcePhase       PhaseID  `yaml:"source_phase" json:"sourcePhase"`
	PerSubjectVerdict []string `yaml:"per_subject_verdict" json:"perSubjectVerdict"`
}

// PhaseSpec is the declared contract for one phase.
type PhaseSpec struct {
	Phase PhaseID `yaml:"phase" json:"phase"`
	Kind  Kind    `yaml:"kind" json:"kind"`
	// Artifact is a filename template containing the {node} placeholder.
	Artifact string `yaml:"artifact" json:"artifact"`
	// LegacyArtifacts are filenames earlier versions of the agent produced. They
	// are used only to recognise pre-existing documentation during adoption, never
	// to write, so the current naming stays the single output convention.
	LegacyArtifacts []string `yaml:"legacyArtifacts,omitempty" json:"legacyArtifacts,omitempty"`
	// DocumentTitle is the canonical English H1 of the rendered artifact. Absent
	// for audit phases, which write no artifact of their own.
	DocumentTitle string `yaml:"documentTitle,omitempty" json:"documentTitle,omitempty"`
	// ArtifactOptional marks phases whose artifact may legitimately be absent.
	ArtifactOptional bool `yaml:"artifactOptional,omitempty" json:"artifactOptional,omitempty"`
	// Optional marks a phase the user may decline outright.
	Optional       bool       `yaml:"optional,omitempty" json:"optional,omitempty"`
	RequiredTopics []Topic    `yaml:"required_topics" json:"requiredTopics"`
	AuditRule      *AuditRule `yaml:"audit_rule,omitempty" json:"auditRule,omitempty"`
}

// ArtifactName resolves the artifact filename for a node short name.
func (s PhaseSpec) ArtifactName(shortName string) string {
	return strings.ReplaceAll(s.Artifact, "{node}", shortName)
}

// LegacyArtifactNames resolves the legacy filenames for a node short name.
func (s PhaseSpec) LegacyArtifactNames(shortName string) []string {
	out := make([]string, 0, len(s.LegacyArtifacts))
	for _, template := range s.LegacyArtifacts {
		out = append(out, strings.ReplaceAll(template, "{node}", shortName))
	}
	return out
}

// QuestionBank is the declared set of phase contracts.
type QuestionBank struct {
	SchemaName string      `yaml:"schemaName" json:"schemaName"`
	Phases     []PhaseSpec `yaml:"phases" json:"phases"`
}

// LoadQuestionBank parses the embedded bank and validates its internal shape.
// A malformed bank is a hard error: this file is the authority, so a silent
// fallback would leave the program enforcing nothing while reporting success.
func LoadQuestionBank() (QuestionBank, error) {
	raw, err := questionBankFS.ReadFile(questionBankPath)
	if err != nil {
		return QuestionBank{}, fmt.Errorf("reading embedded question bank: %w", err)
	}

	var bank QuestionBank
	if err := yaml.Unmarshal(raw, &bank); err != nil {
		return QuestionBank{}, fmt.Errorf("parsing embedded question bank: %w", err)
	}
	if bank.SchemaName != QuestionBankSchema {
		return QuestionBank{}, fmt.Errorf(
			"question bank declares schema %q, expected %q", bank.SchemaName, QuestionBankSchema)
	}
	if err := bank.validate(); err != nil {
		return QuestionBank{}, err
	}
	return bank, nil
}

// validate enforces the invariants the rest of the package relies on.
func (b QuestionBank) validate() error {
	seenPhase := map[PhaseID]bool{}
	for _, spec := range b.Phases {
		if _, err := ParsePhase(string(spec.Phase)); err != nil {
			return fmt.Errorf("question bank declares an unknown phase: %w", err)
		}
		if seenPhase[spec.Phase] {
			return fmt.Errorf("question bank declares phase %q more than once", spec.Phase)
		}
		seenPhase[spec.Phase] = true

		switch spec.Kind {
		case KindInterview:
			if len(spec.RequiredTopics) == 0 {
				return fmt.Errorf("interview phase %q declares no required topics", spec.Phase)
			}
			if spec.DocumentTitle == "" {
				return fmt.Errorf("interview phase %q declares no documentTitle", spec.Phase)
			}
		case KindAudit:
			if len(spec.RequiredTopics) != 0 {
				return fmt.Errorf("audit phase %q must not declare required topics", spec.Phase)
			}
			if spec.AuditRule == nil {
				return fmt.Errorf("audit phase %q declares no audit rule", spec.Phase)
			}
		default:
			return fmt.Errorf("phase %q declares unknown kind %q", spec.Phase, spec.Kind)
		}

		if spec.Artifact == "" {
			return fmt.Errorf("phase %q declares no artifact template", spec.Phase)
		}

		seenTopic := map[string]bool{}
		seenTitle := map[string]bool{}
		for _, topic := range spec.RequiredTopics {
			if topic.ID == "" {
				return fmt.Errorf("phase %q declares a topic with an empty id", spec.Phase)
			}
			if seenTopic[topic.ID] {
				return fmt.Errorf("phase %q declares topic %q more than once", spec.Phase, topic.ID)
			}
			seenTopic[topic.ID] = true

			// A topic with no title has no place to live in the rendered artifact,
			// so its answer could be recorded and never appear in the document.
			if spec.Kind == KindInterview && topic.Title == "" {
				return fmt.Errorf("phase %q topic %q declares no title", spec.Phase, topic.ID)
			}
			seenValue := map[string]bool{}
			for _, value := range topic.Values {
				if value == "" {
					return fmt.Errorf("phase %q topic %q declares an empty value", spec.Phase, topic.ID)
				}
				if seenValue[value] {
					return fmt.Errorf("phase %q topic %q repeats value %q", spec.Phase, topic.ID, value)
				}
				seenValue[value] = true
			}

			if seenTitle[topic.Title] {
				return fmt.Errorf("phase %q reuses section title %q", spec.Phase, topic.Title)
			}
			seenTitle[topic.Title] = true
		}
	}

	for _, id := range canonicalOrder {
		if !seenPhase[id] {
			return fmt.Errorf("question bank is missing canonical phase %q", id)
		}
	}
	return nil
}

// Phase returns the spec for one phase.
func (b QuestionBank) Phase(id PhaseID) (PhaseSpec, bool) {
	for _, spec := range b.Phases {
		if spec.Phase == id {
			return spec, true
		}
	}
	return PhaseSpec{}, false
}

// RequiredTopics returns the topic ids a phase requires for a node type, in
// declaration order. An unknown phase yields no topics rather than a panic;
// callers that need the distinction use Phase.
func (b QuestionBank) RequiredTopics(id PhaseID, nodeType NodeType) []string {
	spec, ok := b.Phase(id)
	if !ok {
		return nil
	}
	var out []string
	for _, topic := range spec.RequiredTopics {
		if topic.appliesTo(nodeType) {
			out = append(out, topic.ID)
		}
	}
	return out
}

// TopicsFor returns the full topic definitions a phase requires for a node
// type, in declaration order. Rendering needs the titles, not only the ids.
func (b QuestionBank) TopicsFor(id PhaseID, nodeType NodeType) []Topic {
	spec, ok := b.Phase(id)
	if !ok {
		return nil
	}
	var out []Topic
	for _, topic := range spec.RequiredTopics {
		if topic.appliesTo(nodeType) {
			out = append(out, topic)
		}
	}
	return out
}

// Topic returns one topic's full definition, which callers need for its closed
// value set and its title.
func (b QuestionBank) Topic(phase PhaseID, topicID string) (Topic, bool) {
	spec, ok := b.Phase(phase)
	if !ok {
		return Topic{}, false
	}
	for _, topic := range spec.RequiredTopics {
		if topic.ID == topicID {
			return topic, true
		}
	}
	return Topic{}, false
}

// KnownTopic reports whether a topic id is declared for a phase, regardless of
// node type. Used to reject answer records that reference topics the bank does
// not define — that mismatch means bank and skills have drifted.
func (b QuestionBank) KnownTopic(id PhaseID, topicID string) bool {
	spec, ok := b.Phase(id)
	if !ok {
		return false
	}
	for _, topic := range spec.RequiredTopics {
		if topic.ID == topicID {
			return true
		}
	}
	return false
}
