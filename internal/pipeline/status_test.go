package pipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// fixture builds a vault-mode node on disk that tests mutate phase by phase.
type fixture struct {
	t    *testing.T
	bank QuestionBank
	env  Environment
	node Node
	res  Resolution
}

func newFixture(t *testing.T, rawNode string) *fixture {
	t.Helper()

	bank := mustLoadBank(t)
	node, err := ParseNode(rawNode)
	if err != nil {
		t.Fatalf("ParseNode(%q): %v", rawNode, err)
	}
	env := Environment{
		ProjectRoot:    t.TempDir(),
		GlobalMode:     ModeVault,
		GlobalBasePath: t.TempDir(),
	}
	res, err := Resolve(node, env)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if err := os.MkdirAll(res.DocsRoot, 0o755); err != nil {
		t.Fatalf("creating docs root: %v", err)
	}

	return &fixture{t: t, bank: bank, env: env, node: node, res: res}
}

// completePhase writes a full answer record plus the artifact for one phase.
func (f *fixture) completePhase(phase PhaseID) {
	f.t.Helper()

	var answers []Answer
	for _, topicID := range f.bank.RequiredTopics(phase, f.node.Type) {
		answers = append(answers, Answer{
			TopicID:    topicID,
			Status:     AnswerAnswered,
			Source:     SourceUserAnswer,
			Verbatim:   "recorded user words for " + topicID,
			CapturedAt: "2026-08-12T18:40:12Z",
		})
	}
	f.writeAnswers(phase, answers)
	f.writeArtifact(phase)
}

func (f *fixture) writeAnswers(phase PhaseID, answers []Answer) {
	f.t.Helper()

	record := AnswerRecord{
		SchemaName: AnswerRecordSchema,
		Node:       f.node.Raw,
		Phase:      phase,
		Answers:    answers,
	}
	path := f.res.AnswerRecordPath(phase)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		f.t.Fatalf("creating answers dir: %v", err)
	}
	raw, err := json.Marshal(record)
	if err != nil {
		f.t.Fatalf("marshalling answers: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		f.t.Fatalf("writing answers: %v", err)
	}
}

func (f *fixture) writeArtifact(phase PhaseID) {
	f.t.Helper()

	path := f.res.ArtifactPath(phase, f.bank)
	if err := os.WriteFile(path, []byte("# artifact\n"), 0o644); err != nil {
		f.t.Fatalf("writing artifact: %v", err)
	}
}

func (f *fixture) completeRefineAudit() {
	f.t.Helper()

	record := AuditRecord{
		SchemaName: AuditRecordSchema,
		Node:       f.node.Raw,
		Phase:      PhaseRefine,
		Subjects: []AuditSubject{{
			ID: "US-1",
			Verdicts: map[string]Verdict{
				"independent": VerdictPass,
				"negotiable":  VerdictPass,
				"valuable":    VerdictPass,
				"estimable":   VerdictPass,
				"small":       VerdictPass,
				"testable":    VerdictPass,
			},
		}},
	}
	path := f.res.AuditRecordPath(PhaseRefine)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		f.t.Fatalf("creating audits dir: %v", err)
	}
	raw, err := json.Marshal(record)
	if err != nil {
		f.t.Fatalf("marshalling audit: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		f.t.Fatalf("writing audit: %v", err)
	}
}

func (f *fixture) declineDDD() {
	f.t.Helper()

	decisions := Decisions{
		SchemaName: DecisionsSchema,
		Node:       f.node.Raw,
		Optional:   map[PhaseID]OptionalPhaseDecision{PhaseDDD: DecisionDeclined},
	}
	path := f.res.DecisionsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		f.t.Fatalf("creating state dir: %v", err)
	}
	raw, err := json.Marshal(decisions)
	if err != nil {
		f.t.Fatalf("marshalling decisions: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		f.t.Fatalf("writing decisions: %v", err)
	}
}

func (f *fixture) status() Status {
	f.t.Helper()
	return ComputeStatus(f.node, f.env, f.bank)
}

func phaseStatus(t *testing.T, status Status, phase PhaseID) PhaseStatus {
	t.Helper()
	for _, ps := range status.Phases {
		if ps.ID == phase {
			return ps
		}
	}
	t.Fatalf("status has no phase %q", phase)
	return PhaseStatus{}
}

func TestFreshNodeRecommendsTheFirstPhase(t *testing.T) {
	f := newFixture(t, "acme-hr")
	status := f.status()

	if status.SchemaName != StatusSchema {
		t.Errorf("schemaName = %q, want %q", status.SchemaName, StatusSchema)
	}
	if status.NextRecommended != PhaseIdea {
		t.Errorf("nextRecommended = %q, want %q", status.NextRecommended, PhaseIdea)
	}
	if got := phaseStatus(t, status, PhaseIdea).State; got != StatePending {
		t.Errorf("idea state = %q, want %q", got, StatePending)
	}
	// Nothing downstream may be pending while idea is unfinished.
	if got := phaseStatus(t, status, PhasePRD).State; got != StateBlocked {
		t.Errorf("prd state = %q, want %q", got, StateBlocked)
	}
	if len(status.Completed) != 0 {
		t.Errorf("completed = %v, want empty", status.Completed)
	}
}

func TestPhaseWithFullCoverageBecomesComplete(t *testing.T) {
	f := newFixture(t, "acme-hr")
	f.completePhase(PhaseIdea)
	f.completePhase(PhaseRec)

	status := f.status()

	for _, phase := range []PhaseID{PhaseIdea, PhaseRec} {
		if got := phaseStatus(t, status, phase).State; got != StateComplete {
			t.Errorf("%s state = %q, want %q", phase, got, StateComplete)
		}
	}
	if status.NextRecommended != PhasePRD {
		t.Errorf("nextRecommended = %q, want %q", status.NextRecommended, PhasePRD)
	}
	if got := phaseStatus(t, status, PhasePRD).State; got != StatePending {
		t.Errorf("prd state = %q, want %q", got, StatePending)
	}
}

func TestMissingTopicKeepsThePhaseIncompleteAndNamesEachGap(t *testing.T) {
	f := newFixture(t, "acme-hr")
	f.completePhase(PhaseIdea)
	f.completePhase(PhaseRec)

	// Answer every prd topic except two.
	required := f.bank.RequiredTopics(PhasePRD, NodeSystem)
	var answers []Answer
	for _, topicID := range required[:len(required)-2] {
		answers = append(answers, Answer{
			TopicID:    topicID,
			Status:     AnswerAnswered,
			Source:     SourceUserAnswer,
			Verbatim:   "recorded words",
			CapturedAt: "2026-08-12T18:40:12Z",
		})
	}
	f.writeAnswers(PhasePRD, answers)
	f.writeArtifact(PhasePRD)

	status := f.status()
	prd := phaseStatus(t, status, PhasePRD)

	if prd.State != StateIncomplete {
		t.Errorf("prd state = %q, want %q", prd.State, StateIncomplete)
	}
	if len(prd.UnansweredTopics) != 2 {
		t.Errorf("unanswered = %v, want 2 entries", prd.UnansweredTopics)
	}
	// The pipeline must not advance past an incomplete phase.
	if status.NextRecommended != PhasePRD {
		t.Errorf("nextRecommended = %q, want %q", status.NextRecommended, PhasePRD)
	}
	if len(status.BlockedReasons) == 0 {
		t.Fatal("blockedReasons is empty for an incomplete phase")
	}
	for _, topicID := range prd.UnansweredTopics {
		if !hasBlockedReason(status, CodeTopicUnanswered, PhasePRD, topicID) {
			t.Errorf("no %q reason for topic %q", CodeTopicUnanswered, topicID)
		}
	}
}

func TestDeferredTopicsCompleteThePhaseButStayVisible(t *testing.T) {
	f := newFixture(t, "acme-hr")
	f.completePhase(PhaseIdea)
	f.completePhase(PhaseRec)

	var answers []Answer
	for i, topicID := range f.bank.RequiredTopics(PhasePRD, NodeSystem) {
		status := AnswerAnswered
		if i%2 == 0 {
			status = AnswerDeferred
		}
		answers = append(answers, Answer{
			TopicID:    topicID,
			Status:     status,
			Source:     SourceUserAnswer,
			Verbatim:   "recorded words",
			CapturedAt: "2026-08-12T18:40:12Z",
		})
	}
	f.writeAnswers(PhasePRD, answers)
	f.writeArtifact(PhasePRD)

	prd := phaseStatus(t, f.status(), PhasePRD)

	if prd.State != StateComplete {
		t.Errorf("prd state = %q, want %q with every topic accounted for", prd.State, StateComplete)
	}
	if len(prd.DeferredTopics) == 0 {
		t.Error("deferred topics disappeared from the report")
	}
}

func TestArtifactWrittenWithoutAnyAnswerRecordIsCaught(t *testing.T) {
	// This is the bypass case: the model wrote the document with its own tool
	// and skipped commit-phase. There is no answer record, so the phase cannot
	// be complete and the gap is visible instead of silent.
	f := newFixture(t, "acme-hr")
	f.completePhase(PhaseIdea)
	f.completePhase(PhaseRec)
	f.writeArtifact(PhasePRD)

	status := f.status()
	prd := phaseStatus(t, status, PhasePRD)

	if prd.State != StateIncomplete {
		t.Errorf("prd state = %q, want %q", prd.State, StateIncomplete)
	}
	if !prd.ArtifactExists {
		t.Error("artifactExists = false, want true")
	}
	if !hasBlockedReasonCode(status, CodeAnswerRecordMissing) {
		t.Errorf("no %q reason for a bypassed artifact", CodeAnswerRecordMissing)
	}
}

func TestCorruptAnswerRecordIsReportedNotIgnored(t *testing.T) {
	f := newFixture(t, "acme-hr")
	f.completePhase(PhaseIdea)

	path := f.res.AnswerRecordPath(PhaseRec)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("creating answers dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("{ not json"), 0o644); err != nil {
		t.Fatalf("writing corrupt record: %v", err)
	}

	status := f.status()

	if got := phaseStatus(t, status, PhaseRec).State; got != StateIncomplete {
		t.Errorf("rec state = %q, want %q", got, StateIncomplete)
	}
	if !hasBlockedReasonCode(status, CodeAnswerRecordUnverifiable) {
		t.Errorf("no %q reason for a corrupt record", CodeAnswerRecordUnverifiable)
	}
}

func TestRefineNeedsItsAuditRecordEvenThoughThePRDFileExists(t *testing.T) {
	f := newFixture(t, "acme-hr")
	f.completePhase(PhaseIdea)
	f.completePhase(PhaseRec)
	f.completePhase(PhasePRD)

	// refine's artifact IS the prd file, which already exists. Presence of that
	// file must not read as a completed audit.
	status := f.status()
	if got := phaseStatus(t, status, PhaseRefine).State; got != StatePending {
		t.Errorf("refine state = %q, want %q", got, StatePending)
	}
	if status.NextRecommended != PhaseRefine {
		t.Errorf("nextRecommended = %q, want %q", status.NextRecommended, PhaseRefine)
	}

	f.completeRefineAudit()

	status = f.status()
	if got := phaseStatus(t, status, PhaseRefine).State; got != StateComplete {
		t.Errorf("refine state = %q, want %q after the audit", got, StateComplete)
	}
	if status.NextRecommended != PhaseTech {
		t.Errorf("nextRecommended = %q, want %q", status.NextRecommended, PhaseTech)
	}
}

func TestOptionalPhaseAsksForADecisionThenRespectsIt(t *testing.T) {
	f := newFixture(t, "acme-hr")
	for _, phase := range []PhaseID{PhaseIdea, PhaseRec, PhasePRD} {
		f.completePhase(phase)
	}
	f.completeRefineAudit()
	f.completePhase(PhaseTech)

	status := f.status()
	if status.NextAction.Kind != ActionDecideOptionalPhase {
		t.Errorf("nextAction kind = %q, want %q", status.NextAction.Kind, ActionDecideOptionalPhase)
	}
	if status.NextRecommended != PhaseDDD {
		t.Errorf("nextRecommended = %q, want %q", status.NextRecommended, PhaseDDD)
	}

	f.declineDDD()

	status = f.status()
	if got := phaseStatus(t, status, PhaseDDD).State; got != StateNotApplicable {
		t.Errorf("ddd state = %q, want %q after decline", got, StateNotApplicable)
	}
	if status.NextRecommended != PhasePTI {
		t.Errorf("nextRecommended = %q, want %q", status.NextRecommended, PhasePTI)
	}
	// A declined optional phase must not appear as missing work.
	for _, phase := range status.Missing {
		if phase == PhaseDDD {
			t.Error("declined ddd is reported as missing")
		}
	}
}

func TestDeclinedOptionalPhaseDoesNotBlockDownstreamPhases(t *testing.T) {
	f := newFixture(t, "acme-hr")
	for _, phase := range []PhaseID{PhaseIdea, PhaseRec, PhasePRD} {
		f.completePhase(phase)
	}
	f.completeRefineAudit()
	f.completePhase(PhaseTech)
	f.declineDDD()

	if got := phaseStatus(t, f.status(), PhasePTI).State; got != StatePending {
		t.Errorf("pti state = %q, want %q", got, StatePending)
	}
}

func TestFullyDocumentedNodeReportsCompletion(t *testing.T) {
	f := newFixture(t, "acme-hr")
	for _, phase := range []PhaseID{PhaseIdea, PhaseRec, PhasePRD} {
		f.completePhase(phase)
	}
	f.completeRefineAudit()
	f.completePhase(PhaseTech)
	f.declineDDD()
	f.completePhase(PhasePTI)

	status := f.status()

	if status.NextAction.Kind != ActionComplete {
		t.Errorf("nextAction kind = %q, want %q", status.NextAction.Kind, ActionComplete)
	}
	if status.NextRecommended != "" {
		t.Errorf("nextRecommended = %q, want empty", status.NextRecommended)
	}
	if len(status.Missing) != 0 {
		t.Errorf("missing = %v, want empty", status.Missing)
	}
	if len(status.BlockedReasons) != 0 {
		t.Errorf("blockedReasons = %v, want empty", status.BlockedReasons)
	}
}

func TestUnresolvableEnvironmentYieldsUndeterminedNotAGuess(t *testing.T) {
	node, err := ParseNode("acme-hr")
	if err != nil {
		t.Fatalf("ParseNode: %v", err)
	}

	// Vault mode with no base path cannot be resolved.
	status := ComputeStatus(node, Environment{
		ProjectRoot: t.TempDir(),
		GlobalMode:  ModeVault,
	}, mustLoadBank(t))

	if status.NextAction.Kind != ActionCannotDetermine {
		t.Errorf("nextAction kind = %q, want %q", status.NextAction.Kind, ActionCannotDetermine)
	}
	if status.NextAction.Command != nil {
		t.Errorf("command = %v, want null when the action cannot be determined", *status.NextAction.Command)
	}
	if !hasBlockedReasonCode(status, CodeDocsRootUnresolved) {
		t.Errorf("no %q reason", CodeDocsRootUnresolved)
	}
	for _, ps := range status.Phases {
		if ps.State != StateUndetermined {
			t.Errorf("%s state = %q, want %q", ps.ID, ps.State, StateUndetermined)
		}
	}
}

func TestNextActionCommandNamesTheRealSlashCommand(t *testing.T) {
	f := newFixture(t, "acme-hr/payroll")
	status := f.status()

	if status.NextAction.Command == nil {
		t.Fatal("command is null for a determinable next action")
	}
	// src/content/commands/ ships doc-idea, so the emitted command must be
	// runnable rather than plausible.
	if want := "/doc-idea acme-hr/payroll"; *status.NextAction.Command != want {
		t.Errorf("command = %q, want %q", *status.NextAction.Command, want)
	}
}

func TestStatusSerialisesWithItsSchemaName(t *testing.T) {
	f := newFixture(t, "acme-hr")

	raw, err := json.Marshal(f.status())
	if err != nil {
		t.Fatalf("marshalling status: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshalling status: %v", err)
	}
	if decoded["schemaName"] != StatusSchema {
		t.Errorf("schemaName = %v, want %q", decoded["schemaName"], StatusSchema)
	}
	for _, key := range []string{"target", "phases", "completed", "missing", "blockedReasons", "nextAction"} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("serialised status is missing key %q", key)
		}
	}
}

func hasBlockedReason(status Status, code string, phase PhaseID, topicID string) bool {
	for _, reason := range status.BlockedReasons {
		if reason.Code == code && reason.Phase == phase && reason.TopicID == topicID {
			return true
		}
	}
	return false
}

func hasBlockedReasonCode(status Status, code string) bool {
	for _, reason := range status.BlockedReasons {
		if reason.Code == code {
			return true
		}
	}
	return false
}
