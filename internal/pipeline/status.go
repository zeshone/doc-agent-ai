package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// StatusSchema is the versioned name of the status contract.
const StatusSchema = "docagent.status/v1"

// Blocked reason codes. These are the vocabulary the orchestrator routes on, so
// they are stable identifiers rather than prose.
const (
	CodePrerequisiteIncomplete   = "prerequisite-phase-incomplete"
	CodeTopicUnanswered          = "topic-unanswered"
	CodeArtifactMissing          = "artifact-missing"
	CodeAnswerRecordMissing      = "answer-record-missing"
	CodeAnswerRecordUnverifiable = "answer-record-unverifiable"
	CodeAuditRecordMissing       = "audit-record-missing"
	CodeAuditRecordUnverifiable  = "audit-record-unverifiable"
	CodeDocsRootUnresolved       = "docs-root-unresolved"
)

// Next-action kinds.
const (
	// ActionStartPhase means the next phase has no recorded work yet.
	ActionStartPhase = "start-phase"
	// ActionResumePhase means the next phase has work on record but gaps remain.
	ActionResumePhase = "resume-phase"
	// ActionDecideOptionalPhase means an optional phase awaits the user's choice.
	ActionDecideOptionalPhase = "decide-optional-phase"
	// ActionComplete means every applicable phase is complete.
	ActionComplete = "complete"
	// ActionCannotDetermine means the program cannot name a correct next step.
	// It emits a null command rather than a plausible one: an unrunnable
	// instruction costs the operator a step and teaches them nothing.
	ActionCannotDetermine = "cannot-determine"
)

// slashCommands maps each phase to the command that runs it. The refine phase
// ships as `doc-refine` while its skill is `doc-refinement`, so the mapping is
// explicit rather than derived from the phase id.
var slashCommands = map[PhaseID]string{
	PhaseIdea:   "/doc-idea",
	PhaseRec:    "/doc-rec",
	PhasePRD:    "/doc-prd",
	PhaseRefine: "/doc-refine",
	PhaseTech:   "/doc-tech",
	PhaseDDD:    "/doc-ddd",
	PhasePTI:    "/doc-pti",
}

// StatusTarget describes what was inspected and where.
type StatusTarget struct {
	Node           string   `json:"node"`
	NodeType       NodeType `json:"nodeType"`
	ShortName      string   `json:"shortName"`
	Mode           string   `json:"mode,omitempty"`
	ModeResolvedBy string   `json:"modeResolvedBy,omitempty"`
	// Archetype is the machine-readable system archetype when it is known, from
	// the recorded answer or from adoption. Empty means unknown, which callers
	// must treat as unknown rather than assuming either value.
	Archetype      string `json:"archetype,omitempty"`
	DocsRoot       string `json:"docsRoot,omitempty"`
	DocsRootExists bool   `json:"docsRootExists"`
}

// PhaseStatus is the computed condition of one phase.
type PhaseStatus struct {
	ID    PhaseID    `json:"id"`
	State PhaseState `json:"state"`
	// Artifact is the expected filename, empty when it cannot be resolved.
	Artifact       string `json:"artifact,omitempty"`
	ArtifactExists bool   `json:"artifactExists"`
	RequiredTopics int    `json:"requiredTopics"`
	AnsweredTopics int    `json:"answeredTopics"`
	// DeferredTopics are gaps the user acknowledged. They do not block, and they
	// never disappear from the report.
	DeferredTopics []string `json:"deferredTopics"`
	// UnansweredTopics are gaps with nothing on record.
	UnansweredTopics []string `json:"unansweredTopics"`
}

// RepeatedVerbatim is one span recorded against several topics.
type RepeatedVerbatim struct {
	Verbatim string   `json:"verbatim"`
	Topics   []string `json:"topics"`
}

// BlockedReason is one machine-routable explanation of missing work.
type BlockedReason struct {
	Code    string  `json:"code"`
	Phase   PhaseID `json:"phase,omitempty"`
	TopicID string  `json:"topicId,omitempty"`
	Detail  string  `json:"detail"`
}

// NextAction names the single next step, or explicitly declines to.
type NextAction struct {
	Kind  string  `json:"kind"`
	Phase PhaseID `json:"phase,omitempty"`
	// Command is null whenever the program cannot construct a runnable one.
	Command      *string `json:"command"`
	Reason       string  `json:"reason"`
	OperatorHint string  `json:"operatorHint,omitempty"`
}

// Status is the typed answer to "where is this node and what happens next".
type Status struct {
	SchemaName string        `json:"schemaName"`
	Target     StatusTarget  `json:"target"`
	Phases     []PhaseStatus `json:"phases"`
	Completed  []PhaseID     `json:"completed"`
	// Adopted lists inherited phases: documented, but coverage unverified. They
	// are neither completed nor missing, and saying so explicitly is the point.
	Adopted         []PhaseID       `json:"adopted"`
	Missing         []PhaseID       `json:"missing"`
	NextRecommended PhaseID         `json:"nextRecommended,omitempty"`
	BlockedReasons  []BlockedReason `json:"blockedReasons"`
	// RepeatedVerbatims names spans recorded for more than one topic. It never
	// blocks: one answer can legitimately cover two topics. It is surfaced because
	// the alternative reading — one answer stretched to pad coverage — is exactly
	// what nobody would notice on their own.
	RepeatedVerbatims []RepeatedVerbatim `json:"repeatedVerbatims,omitempty"`
	NextAction        NextAction         `json:"nextAction"`
}

// ComputeStatus derives a node's position from records on disk.
//
// It never returns an error: an unresolvable environment becomes a typed
// undetermined status so the caller always has something to route on. It never
// reads the master index either — the index is a rendered view, and a checkbox
// the model wrote is exactly the claim this function refuses to trust.
func ComputeStatus(node Node, env Environment, bank QuestionBank) Status {
	status := Status{
		SchemaName: StatusSchema,
		Target: StatusTarget{
			Node:      node.Raw,
			NodeType:  node.Type,
			ShortName: node.ShortName,
		},
		Completed:      []PhaseID{},
		Adopted:        []PhaseID{},
		Missing:        []PhaseID{},
		BlockedReasons: []BlockedReason{},
	}

	res, err := Resolve(node, env)
	if err != nil {
		return undeterminedStatus(status, bank, err)
	}

	status.Target.Mode = res.Mode
	status.Target.ModeResolvedBy = res.ModeResolvedBy
	status.Target.DocsRoot = res.DocsRoot
	status.Target.DocsRootExists = dirExists(res.DocsRoot)

	adoption, _, adoptionErr := LoadAdoption(res.AdoptionPath(), bank)
	if adoptionErr != nil {
		// A corrupt adoption record makes inherited state unknowable. Fail closed
		// rather than silently reporting inherited documentation as missing.
		return undeterminedStatus(status, bank, adoptionErr)
	}
	status.Target.Archetype = adoption.Archetype
	if status.Target.Archetype == "" {
		status.Target.Archetype = recordedArchetype(res, bank)
	}
	if status.Target.Archetype == "" {
		// The archetype is a property of the system, and deeper nodes inherit it.
		// Reporting it here means a caller reading a module's status is not told
		// "unknown" about a fact the system already recorded.
		status.Target.Archetype = inheritedArchetype(node, env, bank)
	}

	decisions, _, decisionsErr := LoadDecisions(res.DecisionsPath(), bank)
	if decisionsErr != nil {
		// A corrupt decisions file makes optional-phase state unknowable. Fail
		// closed rather than silently treating every optional phase as pending.
		return undeterminedStatus(status, bank, decisionsErr)
	}

	// seenVerbatim maps a recorded span to every topic it was recorded against.
	seenVerbatim := map[string][]string{}

	// prerequisitesMet tracks whether every earlier applicable phase is complete.
	prerequisitesMet := true
	var firstBlockedBy PhaseID

	for _, phaseID := range CanonicalPhaseOrder() {
		spec, ok := bank.Phase(phaseID)
		if !ok {
			continue
		}

		ps := PhaseStatus{
			ID:               phaseID,
			Artifact:         spec.ArtifactName(res.ArtifactPrefix),
			DeferredTopics:   []string{},
			UnansweredTopics: []string{},
		}

		// An optional phase the user declined drops out entirely and cannot block
		// anything downstream.
		if spec.Optional && decisions.Optional[phaseID] == DecisionDeclined {
			ps.State = StateNotApplicable
			status.Phases = append(status.Phases, ps)
			continue
		}

		artifactPath := filepath.Join(res.DocsRoot, ps.Artifact)
		ps.ArtifactExists = fileExists(artifactPath)

		var (
			recordPresent bool
			recordUsable  bool
			reasons       []BlockedReason
		)

		if spec.Kind == KindAudit {
			recordPresent, recordUsable, reasons = auditState(res, bank, phaseID)
		} else {
			var coverage TopicCoverage
			coverage, recordPresent, recordUsable, reasons = interviewState(res, bank, phaseID, node.Type)
			collectVerbatims(res, bank, phaseID, seenVerbatim)
			ps.RequiredTopics = len(coverage.Required)
			ps.AnsweredTopics = len(coverage.Answered)
			ps.DeferredTopics = coverage.Deferred
			ps.UnansweredTopics = coverage.Unanswered
		}

		artifactSatisfied := ps.ArtifactExists || spec.ArtifactOptional
		if !artifactSatisfied && recordPresent {
			reasons = append(reasons, BlockedReason{
				Code:   CodeArtifactMissing,
				Phase:  phaseID,
				Detail: fmt.Sprintf("phase %q has answers on record but %s does not exist", phaseID, ps.Artifact),
			})
		}

		// "Started" for an audit phase depends on its own record only. Its
		// artifact belongs to an earlier phase, so file presence proves nothing.
		started := recordPresent
		if spec.Kind != KindAudit {
			started = started || ps.ArtifactExists
		}

		_, inherited := adoption.Phases[phaseID]

		switch {
		case recordUsable && artifactSatisfied:
			ps.State = StateComplete
		case inherited && !recordPresent:
			// Inherited documentation: the artifact stands, its coverage does not.
			ps.State = StateAdopted
		case started:
			ps.State = StateIncomplete
		case prerequisitesMet:
			ps.State = StatePending
		default:
			ps.State = StateBlocked
		}

		if ps.State == StateIncomplete {
			status.BlockedReasons = append(status.BlockedReasons, reasons...)
		}

		if ps.State != StateComplete && ps.State != StateAdopted {
			if prerequisitesMet {
				firstBlockedBy = phaseID
			}
			prerequisitesMet = false
		}

		status.Phases = append(status.Phases, ps)
	}

	for _, ps := range status.Phases {
		switch ps.State {
		case StateComplete:
			status.Completed = append(status.Completed, ps.ID)
		case StateAdopted:
			status.Adopted = append(status.Adopted, ps.ID)
		case StateNotApplicable:
			// Declined work is not missing work.
		default:
			status.Missing = append(status.Missing, ps.ID)
		}
	}

	status.RepeatedVerbatims = repeatedVerbatims(seenVerbatim)
	status.NextRecommended = firstBlockedBy
	status.NextAction = nextAction(status, bank, decisions, node)
	return status
}

// collectVerbatims records which topics each span was used for.
func collectVerbatims(res Resolution, bank QuestionBank, phase PhaseID, seen map[string][]string) {
	record, found, err := LoadAnswerRecord(res.AnswerRecordPath(phase), bank)
	if err != nil || !found {
		return
	}
	for _, answer := range record.Answers {
		span := strings.TrimSpace(answer.Verbatim)
		if span == "" {
			continue
		}
		seen[span] = append(seen[span], string(phase)+"/"+answer.TopicID)
	}
}

// repeatedVerbatims returns spans used for more than one topic, in a stable order.
func repeatedVerbatims(seen map[string][]string) []RepeatedVerbatim {
	var out []RepeatedVerbatim
	for span, topics := range seen {
		if len(topics) < 2 {
			continue
		}
		out = append(out, RepeatedVerbatim{Verbatim: span, Topics: topics})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Verbatim < out[j].Verbatim })
	return out
}

// interviewState loads and counts an interview phase's answer record.
func interviewState(res Resolution, bank QuestionBank, phase PhaseID, nodeType NodeType) (
	coverage TopicCoverage, present bool, usable bool, reasons []BlockedReason) {

	path := res.AnswerRecordPath(phase)
	record, found, err := LoadAnswerRecord(path, bank)

	switch {
	case err != nil:
		// A record that exists but cannot be trusted is worse than none: report
		// it rather than treating the phase as untouched.
		reasons = append(reasons, BlockedReason{
			Code:   CodeAnswerRecordUnverifiable,
			Phase:  phase,
			Detail: err.Error(),
		})
		coverage = Coverage(bank, phase, nodeType, AnswerRecord{})
		return coverage, true, false, reasons

	case !found:
		coverage = Coverage(bank, phase, nodeType, AnswerRecord{})
		reasons = append(reasons, BlockedReason{
			Code:   CodeAnswerRecordMissing,
			Phase:  phase,
			Detail: fmt.Sprintf("phase %q has no answer record at %s", phase, path),
		})
		return coverage, false, false, reasons
	}

	coverage = Coverage(bank, phase, nodeType, record)
	for _, topicID := range coverage.Unanswered {
		reasons = append(reasons, BlockedReason{
			Code:    CodeTopicUnanswered,
			Phase:   phase,
			TopicID: topicID,
			Detail:  fmt.Sprintf("phase %q requires topic %q and no answer is on record", phase, topicID),
		})
	}
	return coverage, true, coverage.Complete(), reasons
}

// auditState loads an audit phase's verdict record.
func auditState(res Resolution, bank QuestionBank, phase PhaseID) (
	present bool, usable bool, reasons []BlockedReason) {

	path := res.AuditRecordPath(phase)
	_, found, err := LoadAuditRecord(path, bank)

	switch {
	case err != nil:
		return true, false, []BlockedReason{{
			Code:   CodeAuditRecordUnverifiable,
			Phase:  phase,
			Detail: err.Error(),
		}}
	case !found:
		return false, false, []BlockedReason{{
			Code:   CodeAuditRecordMissing,
			Phase:  phase,
			Detail: fmt.Sprintf("phase %q has no audit record at %s", phase, path),
		}}
	}
	return true, true, nil
}

// nextAction picks the single next step from the computed phase states.
func nextAction(status Status, bank QuestionBank, decisions Decisions, node Node) NextAction {
	if status.NextRecommended == "" {
		reason := "every applicable phase is complete"
		if len(status.Adopted) > 0 {
			// Saying "complete" here would claim coverage nobody counted.
			reason = fmt.Sprintf(
				"no phase is outstanding, but %d of them are adopted: documented from before answer records existed, with coverage unverified",
				len(status.Adopted))
		}
		return NextAction{
			Kind:    ActionComplete,
			Command: nil,
			Reason:  reason,
		}
	}

	phase := status.NextRecommended
	spec, ok := bank.Phase(phase)
	if !ok {
		return NextAction{
			Kind:         ActionCannotDetermine,
			Command:      nil,
			Reason:       fmt.Sprintf("phase %q is not declared in the question bank", phase),
			OperatorHint: "the installed question bank and this binary disagree; reinstall doc-agent-ai",
		}
	}

	command, known := slashCommands[phase]
	if !known {
		return NextAction{
			Kind:         ActionCannotDetermine,
			Phase:        phase,
			Command:      nil,
			Reason:       fmt.Sprintf("no command is mapped for phase %q", phase),
			OperatorHint: "report this as a defect: the phase exists but has no runnable command",
		}
	}
	full := fmt.Sprintf("%s %s", command, node.Raw)

	// An undecided optional phase needs the user's choice, not a launch.
	if spec.Optional {
		if _, decided := decisions.Optional[phase]; !decided {
			return NextAction{
				Kind:    ActionDecideOptionalPhase,
				Phase:   phase,
				Command: &full,
				Reason: fmt.Sprintf(
					"phase %q is optional and has no recorded decision: ask the user before running it", phase),
			}
		}
	}

	var current PhaseStatus
	for _, ps := range status.Phases {
		if ps.ID == phase {
			current = ps
		}
	}

	kind := ActionStartPhase
	reason := fmt.Sprintf("phase %q has no recorded work yet", phase)
	if current.State == StateIncomplete {
		kind = ActionResumePhase
		reason = fmt.Sprintf("phase %q is incomplete: %d of %d required topics have no recorded answer",
			phase, len(current.UnansweredTopics), current.RequiredTopics)
	}

	return NextAction{
		Kind:    kind,
		Phase:   phase,
		Command: &full,
		Reason:  reason,
	}
}

// undeterminedStatus marks every phase undetermined and refuses to name a step.
func undeterminedStatus(status Status, bank QuestionBank, cause error) Status {
	status.Phases = nil
	status.Adopted = []PhaseID{}
	for _, phaseID := range CanonicalPhaseOrder() {
		status.Phases = append(status.Phases, PhaseStatus{
			ID:               phaseID,
			State:            StateUndetermined,
			DeferredTopics:   []string{},
			UnansweredTopics: []string{},
		})
		status.Missing = append(status.Missing, phaseID)
	}
	status.BlockedReasons = append(status.BlockedReasons, BlockedReason{
		Code:   CodeDocsRootUnresolved,
		Detail: cause.Error(),
	})
	status.NextAction = NextAction{
		Kind:         ActionCannotDetermine,
		Command:      nil,
		Reason:       cause.Error(),
		OperatorHint: "resolve the documentation destination, then run status again",
	}
	return status
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
