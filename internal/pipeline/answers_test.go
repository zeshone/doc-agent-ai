package pipeline

import (
	"strings"
	"testing"
)

// answeredTopic builds a minimal valid answered entry.
func answeredTopic(topicID string) Answer {
	return Answer{
		TopicID:    topicID,
		Status:     AnswerAnswered,
		Source:     SourceUserAnswer,
		Prompt:     "what about this topic?",
		Verbatim:   "the user said this out loud",
		CapturedAt: "2026-08-12T18:40:12Z",
	}
}

func recordWith(phase PhaseID, answers ...Answer) AnswerRecord {
	return AnswerRecord{
		SchemaName: AnswerRecordSchema,
		Node:       "acme-hr/payroll",
		Phase:      phase,
		Answers:    answers,
	}
}

func TestValidAnswerRecordIsAccepted(t *testing.T) {
	bank := mustLoadBank(t)

	record := recordWith(PhasePRD,
		answeredTopic("success-metrics"),
		Answer{
			TopicID:    "risks-roadmap",
			Status:     AnswerDeferred,
			Source:     SourceUserAnswer,
			Prompt:     "what about this topic?",
			Verbatim:   "no idea yet, we have to sequence it first",
			CapturedAt: "2026-08-12T18:44:51Z",
		},
		Answer{
			TopicID:       "technical-constraints",
			Status:        AnswerAnswered,
			Source:        SourceInheritedParent,
			InheritedFrom: "acme-hr/acme-hr_prd.md#technical-constraints",
			CapturedAt:    "2026-08-12T18:45:02Z",
		},
		Answer{
			TopicID:    "security-privacy",
			Status:     AnswerAnswered,
			Source:     SourceBrainDump,
			Verbatim:   "only HR and the area manager can see salary data",
			CapturedAt: "2026-08-12T18:31:04Z",
		},
	)

	if err := record.Validate(bank); err != nil {
		t.Fatalf("Validate() returned error for a valid record: %v", err)
	}
}

func TestAnswerRecordRejectsMissingProvenance(t *testing.T) {
	bank := mustLoadBank(t)

	tests := []struct {
		name        string
		answer      Answer
		wantMessage string
	}{
		{
			// This is the whole point of option A: an answer with no verbatim span
			// is an unsupported claim, so it must not count as coverage.
			name: "user answer without a verbatim span",
			answer: Answer{
				TopicID:    "success-metrics",
				Status:     AnswerAnswered,
				Source:     SourceUserAnswer,
				Prompt:     "what about this topic?",
				CapturedAt: "2026-08-12T18:40:12Z",
			},
			wantMessage: "verbatim",
		},
		{
			name: "brain dump answer without a verbatim span",
			answer: Answer{
				TopicID:    "success-metrics",
				Status:     AnswerAnswered,
				Source:     SourceBrainDump,
				CapturedAt: "2026-08-12T18:40:12Z",
			},
			wantMessage: "verbatim",
		},
		{
			name: "inherited answer without a parent pointer",
			answer: Answer{
				TopicID:    "success-metrics",
				Status:     AnswerAnswered,
				Source:     SourceInheritedParent,
				CapturedAt: "2026-08-12T18:40:12Z",
			},
			wantMessage: "inheritedFrom",
		},
		{
			// An inherited answer has no user utterance to quote, so a verbatim
			// span there would be fabricated by construction.
			name: "inherited answer carrying a verbatim span",
			answer: Answer{
				TopicID:       "success-metrics",
				Status:        AnswerAnswered,
				Source:        SourceInheritedParent,
				InheritedFrom: "acme-hr/acme-hr_prd.md#success-metrics",
				Verbatim:      "the user never said this",
				CapturedAt:    "2026-08-12T18:40:12Z",
			},
			wantMessage: "verbatim",
		},
		{
			name: "deferred answer without the user saying so",
			answer: Answer{
				TopicID:    "success-metrics",
				Status:     AnswerDeferred,
				Source:     SourceUserAnswer,
				Prompt:     "what about this topic?",
				CapturedAt: "2026-08-12T18:40:12Z",
			},
			wantMessage: "verbatim",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := recordWith(PhasePRD, tt.answer).Validate(bank)
			if err == nil {
				t.Fatal("Validate() accepted a record with missing provenance")
			}
			if !strings.Contains(err.Error(), tt.wantMessage) {
				t.Errorf("error %q does not mention %q", err.Error(), tt.wantMessage)
			}
		})
	}
}

func TestAnswerRecordRejectsStructuralProblems(t *testing.T) {
	bank := mustLoadBank(t)

	tests := []struct {
		name   string
		mutate func(*AnswerRecord)
	}{
		{
			name:   "wrong schema name",
			mutate: func(r *AnswerRecord) { r.SchemaName = "docagent.answers/v99" },
		},
		{
			name:   "empty node",
			mutate: func(r *AnswerRecord) { r.Node = "" },
		},
		{
			name:   "unknown phase",
			mutate: func(r *AnswerRecord) { r.Phase = "tech-specs" },
		},
		{
			name: "topic the question bank does not declare",
			mutate: func(r *AnswerRecord) {
				r.Answers = append(r.Answers, answeredTopic("invented-topic"))
			},
		},
		{
			name: "duplicate topic",
			mutate: func(r *AnswerRecord) {
				r.Answers = append(r.Answers, answeredTopic("success-metrics"))
			},
		},
		{
			name:   "unknown status",
			mutate: func(r *AnswerRecord) { r.Answers[0].Status = "probably" },
		},
		{
			name:   "unknown source",
			mutate: func(r *AnswerRecord) { r.Answers[0].Source = "vibes" },
		},
		{
			name:   "empty topic id",
			mutate: func(r *AnswerRecord) { r.Answers[0].TopicID = "" },
		},
		{
			name:   "missing capture timestamp",
			mutate: func(r *AnswerRecord) { r.Answers[0].CapturedAt = "" },
		},
		{
			name:   "capture timestamp is not RFC3339",
			mutate: func(r *AnswerRecord) { r.Answers[0].CapturedAt = "last tuesday" },
		},
		{
			name:   "verbatim span is only whitespace",
			mutate: func(r *AnswerRecord) { r.Answers[0].Verbatim = "   \n  " },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := recordWith(PhasePRD, answeredTopic("success-metrics"))
			tt.mutate(&record)
			if err := record.Validate(bank); err == nil {
				t.Fatal("Validate() accepted a structurally invalid record")
			}
		})
	}
}

func TestCoverageCountsRecordedAnswersNotClaims(t *testing.T) {
	bank := mustLoadBank(t)

	record := recordWith(PhasePRD,
		answeredTopic("primary-user-flows"),
		answeredTopic("user-stories"),
		Answer{
			TopicID:    "risks-roadmap",
			Status:     AnswerDeferred,
			Source:     SourceUserAnswer,
			Prompt:     "what about this topic?",
			Verbatim:   "we do not know yet",
			CapturedAt: "2026-08-12T18:44:51Z",
		},
	)

	coverage := Coverage(bank, PhasePRD, NodeSystem, record)

	if got := len(coverage.Answered); got != 2 {
		t.Errorf("answered = %d, want 2 (%v)", got, coverage.Answered)
	}
	if got := len(coverage.Deferred); got != 1 {
		t.Errorf("deferred = %d, want 1 (%v)", got, coverage.Deferred)
	}
	// prd declares nine topics; three are accounted for, so six remain absent.
	if got := len(coverage.Unanswered); got != 6 {
		t.Errorf("unanswered = %d, want 6 (%v)", got, coverage.Unanswered)
	}
	if coverage.Complete() {
		t.Error("Complete() = true with six topics unanswered")
	}
}

func TestCoverageIsCompleteWhenDeferralsFillTheRemainder(t *testing.T) {
	bank := mustLoadBank(t)

	// A deferral is an acknowledged gap, not a hidden one. Global agent rule 8
	// (skills/doc-arch/SKILL.md:170) permits TBD, so a deferred topic must not
	// block the pipeline forever — but it stays counted and visible.
	var answers []Answer
	for _, topicID := range bank.RequiredTopics(PhasePRD, NodeSystem) {
		answers = append(answers, Answer{
			TopicID:    topicID,
			Status:     AnswerDeferred,
			Source:     SourceUserAnswer,
			Prompt:     "what about this topic?",
			Verbatim:   "not known yet",
			CapturedAt: "2026-08-12T18:44:51Z",
		})
	}

	coverage := Coverage(bank, PhasePRD, NodeSystem, recordWith(PhasePRD, answers...))

	if !coverage.Complete() {
		t.Errorf("Complete() = false with every topic explicitly deferred (%v)", coverage.Unanswered)
	}
	if len(coverage.Deferred) != len(coverage.Required) {
		t.Errorf("deferred = %d, want %d", len(coverage.Deferred), len(coverage.Required))
	}
}

func TestCoverageIgnoresTopicsTheNodeTypeDoesNotOwe(t *testing.T) {
	bank := mustLoadBank(t)

	// inheritance-mode applies only to modules. Answering it on a system node is
	// harmless extra information, not an error and not a missing requirement.
	var answers []Answer
	for _, topicID := range bank.RequiredTopics(PhaseTech, NodeSystem) {
		answers = append(answers, answeredTopic(topicID))
	}
	answers = append(answers, answeredTopic("inheritance-mode"))

	coverage := Coverage(bank, PhaseTech, NodeSystem, recordWith(PhaseTech, answers...))

	if !coverage.Complete() {
		t.Errorf("Complete() = false for a fully covered system node (%v)", coverage.Unanswered)
	}
}

func TestCoverageOnEmptyRecordReportsEverythingUnanswered(t *testing.T) {
	bank := mustLoadBank(t)

	coverage := Coverage(bank, PhasePRD, NodeSystem, AnswerRecord{})

	if len(coverage.Unanswered) != len(coverage.Required) {
		t.Errorf("unanswered = %d, want %d", len(coverage.Unanswered), len(coverage.Required))
	}
	if coverage.Complete() {
		t.Error("Complete() = true for an empty record")
	}
}

// answerForTopic builds a valid answered entry, supplying the machine-readable
// choice whenever the bank closed that topic to a fixed set.
func answerForTopic(bank QuestionBank, phase PhaseID, topicID string) Answer {
	answer := Answer{
		TopicID:    topicID,
		Status:     AnswerAnswered,
		Source:     SourceUserAnswer,
		Prompt:     "what should we know about " + topicID + "?",
		Verbatim:   "recorded user words for " + topicID,
		CapturedAt: "2026-08-12T18:40:12Z",
	}
	if topic, ok := bank.Topic(phase, topicID); ok && len(topic.Values) > 0 {
		answer.Value = topic.Values[0]
	}
	return answer
}

func mustLoadBank(t *testing.T) QuestionBank {
	t.Helper()
	bank, err := LoadQuestionBank()
	if err != nil {
		t.Fatalf("LoadQuestionBank() returned error: %v", err)
	}
	return bank
}

func TestTheQuestionIsHalfTheRecord(t *testing.T) {
	// "son justo esos!" is a real answer and completely unreadable on its own.
	// Without the question, a later reader cannot judge whether the words fit the
	// topic — which is the audit the verbatim exists to make possible.
	bank := mustLoadBank(t)

	tests := []struct {
		name     string
		answer   Answer
		wantErr  bool
		mentions string
	}{
		{
			name: "a direct reply carries the question that produced it",
			answer: Answer{
				TopicID: "success-metrics", Status: AnswerAnswered, Source: SourceUserAnswer,
				Prompt:     "how will you know in six months that this worked?",
				Verbatim:   "contactos a vendedores y patrocinios vendidos",
				CapturedAt: "2026-08-13T12:00:00Z",
			},
		},
		{
			name: "a direct reply without the question is refused",
			answer: Answer{
				TopicID: "success-metrics", Status: AnswerAnswered, Source: SourceUserAnswer,
				Verbatim: "son justo esos!", CapturedAt: "2026-08-13T12:00:00Z",
			},
			wantErr: true, mentions: "prompt",
		},
		{
			// Requiring a prompt everywhere would push the agent to invent one, which
			// is the same class of defect the verbatim rule exists to prevent.
			name: "words offered unasked need no question",
			answer: Answer{
				TopicID: "success-metrics", Status: AnswerAnswered, Source: SourceVolunteered,
				Verbatim: "por cierto, tambien medimos patrocinios", CapturedAt: "2026-08-13T12:00:00Z",
			},
		},
		{
			name: "unasked words must not carry an invented question",
			answer: Answer{
				TopicID: "success-metrics", Status: AnswerAnswered, Source: SourceVolunteered,
				Prompt: "what are your metrics?", Verbatim: "algo", CapturedAt: "2026-08-13T12:00:00Z",
			},
			wantErr: true, mentions: "nobody asked",
		},
		{
			name: "the brain dump was never a question either",
			answer: Answer{
				TopicID: "success-metrics", Status: AnswerAnswered, Source: SourceBrainDump,
				Prompt: "what are your metrics?", Verbatim: "algo", CapturedAt: "2026-08-13T12:00:00Z",
			},
			wantErr: true, mentions: "nobody asked",
		},
		{
			name: "inherited context has neither a quote nor a question",
			answer: Answer{
				TopicID: "success-metrics", Status: AnswerAnswered, Source: SourceInheritedParent,
				InheritedFrom: "parent_prd.md#success-metrics",
				Prompt:        "invented", CapturedAt: "2026-08-13T12:00:00Z",
			},
			wantErr: true, mentions: "prompt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := recordWith(PhasePRD, tt.answer).Validate(bank)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Validate accepted a record the rule should refuse")
				}
				if tt.mentions != "" && !strings.Contains(err.Error(), tt.mentions) {
					t.Errorf("error %q does not mention %q", err, tt.mentions)
				}
				return
			}
			if err != nil {
				t.Fatalf("Validate: %v", err)
			}
		})
	}
}

func TestRefusalNamesTheWayOutForUnaskedWords(t *testing.T) {
	// A rule that refuses without naming the alternative pushes the agent to
	// fabricate a question rather than reclassify the source.
	bank := mustLoadBank(t)

	err := recordWith(PhasePRD, Answer{
		TopicID: "success-metrics", Status: AnswerAnswered, Source: SourceUserAnswer,
		Verbatim: "algo", CapturedAt: "2026-08-13T12:00:00Z",
	}).Validate(bank)

	if err == nil {
		t.Fatal("Validate accepted a prompt-less direct reply")
	}
	if !strings.Contains(err.Error(), string(SourceVolunteered)) {
		t.Errorf("the refusal does not name %q as the alternative: %v", SourceVolunteered, err)
	}
}
