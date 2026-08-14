package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// submission builds a fully covered submission for one interview phase.
func (f *fixture) submission(phase PhaseID) Submission {
	f.t.Helper()

	var answers []Answer
	for _, topicID := range f.bank.RequiredTopics(phase, f.node.Type) {
		answers = append(answers, answerForTopic(f.bank, phase, topicID))
	}
	sections := map[string]string{}
	for _, topic := range f.bank.RequiredTopics(phase, f.node.Type) {
		sections[topic] = "prose in the user's language about " + topic
	}

	return Submission{
		Node:    f.node,
		Phase:   phase,
		Content: SectionInput{SchemaName: SectionsSchema, Sections: sections},
		Answers: &AnswerRecord{
			SchemaName: AnswerRecordSchema,
			Node:       f.node.Raw,
			Phase:      phase,
			Answers:    answers,
		},
	}
}

func TestCommitWritesArtifactRecordAndIndex(t *testing.T) {
	f := newFixture(t, "acme-hr")

	result := Commit(f.submission(PhaseIdea), f.env, f.bank)

	if result.Result != CommitWritten {
		t.Fatalf("result = %q, want %q (detail: %s)", result.Result, CommitWritten, result.Detail)
	}
	if result.SchemaName != CommitSchema {
		t.Errorf("schemaName = %q, want %q", result.SchemaName, CommitSchema)
	}
	// answer record, the authored prose, and the rendered artifact
	if len(result.Written) != 3 {
		t.Errorf("written = %v, want the answer record, the section input and the artifact", result.Written)
	}
	for _, path := range result.Written {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("reported %s as written but it does not exist: %v", path, err)
		}
	}
	if result.IndexUpdated == nil {
		t.Fatal("indexUpdated is nil after a successful commit")
	}
	if _, err := os.Stat(result.IndexUpdated.File); err != nil {
		t.Errorf("index was not created: %v", err)
	}
}

func TestRejectedCommitWritesNothingAtAll(t *testing.T) {
	f := newFixture(t, "acme-hr")

	sub := f.submission(PhaseRec)
	// Drop two topics so coverage fails.
	sub.Answers.Answers = sub.Answers.Answers[:len(sub.Answers.Answers)-2]

	result := Commit(sub, f.env, f.bank)

	if result.Result != CommitRejected {
		t.Fatalf("result = %q, want %q", result.Result, CommitRejected)
	}
	if len(result.Written) != 0 {
		t.Errorf("written = %v, want nothing on rejection", result.Written)
	}
	if result.Validation == nil {
		t.Fatal("rejection carries no validation result")
	}
	if !containsString(result.Validation.RejectedBecause, CheckRequiredTopicsCovered) {
		t.Errorf("rejectedBecause = %v, want %q", result.Validation.RejectedBecause, CheckRequiredTopicsCovered)
	}

	// The hard rule: no partial artifacts. Nothing may exist on disk.
	res, err := Resolve(f.node, f.env)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	for _, path := range []string{
		res.ArtifactPath(PhaseRec, f.bank),
		res.AnswerRecordPath(PhaseRec),
		res.IndexPath,
	} {
		if _, err := os.Stat(path); err == nil {
			t.Errorf("%s exists after a rejected commit", path)
		}
	}
}

func TestCommitRejectsAnAnsweredTopicWithNoProse(t *testing.T) {
	f := newFixture(t, "acme-hr")

	// This closes the gap a monolithic draft left open: the record could claim a
	// topic was answered while the document never mentioned it.
	sub := f.submission(PhaseIdea)
	delete(sub.Content.Sections, "why-now")

	result := Commit(sub, f.env, f.bank)

	if result.Result != CommitRejected {
		t.Fatalf("result = %q, want %q", result.Result, CommitRejected)
	}
	if !containsString(result.Validation.RejectedBecause, CheckSectionContentPresent) {
		t.Errorf("rejectedBecause = %v, want %q",
			result.Validation.RejectedBecause, CheckSectionContentPresent)
	}
}

func TestCommitRejectsSectionsForUnknownTopics(t *testing.T) {
	f := newFixture(t, "acme-hr")

	sub := f.submission(PhaseIdea)
	sub.Content.Sections["invented-topic"] = "prose nobody asked for"

	result := Commit(sub, f.env, f.bank)

	if result.Result != CommitRejected {
		t.Fatalf("result = %q, want %q", result.Result, CommitRejected)
	}
	if !containsString(result.Validation.RejectedBecause, CheckSectionsKnown) {
		t.Errorf("rejectedBecause = %v, want %q", result.Validation.RejectedBecause, CheckSectionsKnown)
	}
}

func TestCommitRejectsAnswersWithoutProvenance(t *testing.T) {
	f := newFixture(t, "acme-hr")

	sub := f.submission(PhaseIdea)
	// Strip the user's words from one entry: the exact fabrication this blocks.
	sub.Answers.Answers[0].Verbatim = ""

	result := Commit(sub, f.env, f.bank)

	if result.Result != CommitRejected {
		t.Fatalf("result = %q, want %q", result.Result, CommitRejected)
	}
	if !containsString(result.Validation.RejectedBecause, CheckAnswerRecordPresent) {
		t.Errorf("rejectedBecause = %v, want %q", result.Validation.RejectedBecause, CheckAnswerRecordPresent)
	}
}

func TestCommitRejectsARecordForADifferentNode(t *testing.T) {
	f := newFixture(t, "acme-hr")

	sub := f.submission(PhaseIdea)
	sub.Answers.Node = "some-other-system"

	if result := Commit(sub, f.env, f.bank); result.Result != CommitRejected {
		t.Fatalf("result = %q, want %q", result.Result, CommitRejected)
	}
}

func TestIndexRegionIsRecomputedAndAuthorProseIsPreserved(t *testing.T) {
	f := newFixture(t, "acme-hr")

	if result := Commit(f.submission(PhaseIdea), f.env, f.bank); result.Result != CommitWritten {
		t.Fatalf("first commit failed: %s", result.Detail)
	}

	res, err := Resolve(f.node, f.env)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// Simulate the author editing prose outside the machine-owned region.
	original, err := os.ReadFile(res.IndexPath)
	if err != nil {
		t.Fatalf("reading index: %v", err)
	}
	edited := strings.Replace(string(original), "TBD", "A payroll system for Acme HR.", 1)
	if err := os.WriteFile(res.IndexPath, []byte(edited), 0o644); err != nil {
		t.Fatalf("writing edited index: %v", err)
	}

	if result := Commit(f.submission(PhaseRec), f.env, f.bank); result.Result != CommitWritten {
		t.Fatalf("second commit failed: %s", result.Detail)
	}

	updated, err := os.ReadFile(res.IndexPath)
	if err != nil {
		t.Fatalf("reading updated index: %v", err)
	}
	text := string(updated)

	if !strings.Contains(text, "A payroll system for Acme HR.") {
		t.Error("author prose outside the region was destroyed")
	}
	if strings.Count(text, indexRegionBegin) != 1 {
		t.Errorf("region begin marker appears %d times, want 1", strings.Count(text, indexRegionBegin))
	}
	if !strings.Contains(text, "| rec | [x] |") {
		t.Error("rec was not marked complete in the recomputed region")
	}
	if !strings.Contains(text, NodeStatusInProgress) {
		t.Errorf("node status %q is missing from the region", NodeStatusInProgress)
	}
}

func TestIndexWithoutARegionGetsOneAppended(t *testing.T) {
	f := newFixture(t, "acme-hr")

	res, err := Resolve(f.node, f.env)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	// An index written by an earlier version of the agent, with no region.
	legacy := "# acme-hr\n\nExisting hand-written index.\n"
	if err := os.WriteFile(res.IndexPath, []byte(legacy), 0o644); err != nil {
		t.Fatalf("writing legacy index: %v", err)
	}

	if result := Commit(f.submission(PhaseIdea), f.env, f.bank); result.Result != CommitWritten {
		t.Fatalf("commit failed: %s", result.Detail)
	}

	updated, err := os.ReadFile(res.IndexPath)
	if err != nil {
		t.Fatalf("reading index: %v", err)
	}
	text := string(updated)

	if !strings.Contains(text, "Existing hand-written index.") {
		t.Error("legacy index content was destroyed")
	}
	if !strings.Contains(text, indexRegionBegin) {
		t.Error("state region was not appended to the legacy index")
	}
}

func TestUnbalancedIndexRegionIsRefusedRatherThanGuessed(t *testing.T) {
	f := newFixture(t, "acme-hr")

	res, err := Resolve(f.node, f.env)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	broken := "# acme-hr\n\nprose\n\n" + indexRegionBegin + "\ntruncated\n"
	if err := os.WriteFile(res.IndexPath, []byte(broken), 0o644); err != nil {
		t.Fatalf("writing broken index: %v", err)
	}

	result := Commit(f.submission(PhaseIdea), f.env, f.bank)

	// The durable evidence still lands; only the index is left alone, and the
	// failure is reported instead of prose being destroyed.
	if result.Result != CommitWritten {
		t.Fatalf("result = %q, want %q", result.Result, CommitWritten)
	}
	if result.Detail == "" {
		t.Error("index failure was not reported")
	}
	if result.IndexUpdated != nil {
		t.Error("indexUpdated is set even though the index was not written")
	}

	after, err := os.ReadFile(res.IndexPath)
	if err != nil {
		t.Fatalf("reading index: %v", err)
	}
	if !strings.Contains(string(after), "prose") {
		t.Error("prose was destroyed by an unbalanced region")
	}
}

func TestCommitOfAuditPhaseWritesOnlyTheAuditRecord(t *testing.T) {
	f := newFixture(t, "acme-hr")

	sub := Submission{
		Node:  f.node,
		Phase: PhaseRefine,
		Audit: &AuditRecord{
			SchemaName: AuditRecordSchema,
			Node:       f.node.Raw,
			Phase:      PhaseRefine,
			Subjects: []AuditSubject{{
				ID: "US-1",
				Verdicts: map[string]Verdict{
					"independent": VerdictPass, "negotiable": VerdictPass,
					"valuable": VerdictPass, "estimable": VerdictPass,
					"small": VerdictPass, "testable": VerdictFail,
				},
				Notes: "story spans two releases",
			}},
		},
	}

	result := Commit(sub, f.env, f.bank)

	if result.Result != CommitWritten {
		t.Fatalf("result = %q, want %q (detail: %s)", result.Result, CommitWritten, result.Detail)
	}
	if len(result.Written) != 1 {
		t.Errorf("written = %v, want only the audit record", result.Written)
	}
	if !strings.Contains(result.Written[0], "audits") {
		t.Errorf("written path %q is not the audit record", result.Written[0])
	}
}

func TestAuditCommitRejectsAnEmptySubjectList(t *testing.T) {
	f := newFixture(t, "acme-hr")

	sub := Submission{
		Node:  f.node,
		Phase: PhaseRefine,
		Audit: &AuditRecord{
			SchemaName: AuditRecordSchema,
			Node:       f.node.Raw,
			Phase:      PhaseRefine,
			Subjects:   nil,
		},
	}

	result := Commit(sub, f.env, f.bank)

	// "I audited nothing" must not read as a completed audit.
	if result.Result != CommitRejected {
		t.Fatalf("result = %q, want %q", result.Result, CommitRejected)
	}
}

func TestRecordDecisionPersistsAndSurvivesReload(t *testing.T) {
	f := newFixture(t, "acme-hr")

	if err := RecordDecision(f.node, f.env, f.bank, PhaseDDD, DecisionDeclined); err != nil {
		t.Fatalf("RecordDecision: %v", err)
	}

	if got := phaseStatus(t, f.status(), PhaseDDD).State; got != StateNotApplicable {
		t.Errorf("ddd state = %q, want %q", got, StateNotApplicable)
	}

	// Changing the decision must overwrite rather than accumulate.
	if err := RecordDecision(f.node, f.env, f.bank, PhaseDDD, DecisionAccepted); err != nil {
		t.Fatalf("RecordDecision: %v", err)
	}
	if got := phaseStatus(t, f.status(), PhaseDDD).State; got == StateNotApplicable {
		t.Error("ddd is still not-applicable after the decision was reversed")
	}
}

func TestRecordDecisionRejectsNonOptionalPhases(t *testing.T) {
	f := newFixture(t, "acme-hr")

	if err := RecordDecision(f.node, f.env, f.bank, PhasePRD, DecisionDeclined); err == nil {
		t.Fatal("RecordDecision accepted a decision for a mandatory phase")
	}
}

func TestCommitUndeterminedWhenTheDestinationCannotBeResolved(t *testing.T) {
	node, err := ParseNode("acme-hr")
	if err != nil {
		t.Fatalf("ParseNode: %v", err)
	}
	bank := mustLoadBank(t)

	result := Commit(Submission{
		Node:  node,
		Phase: PhaseIdea,
	}, Environment{ProjectRoot: t.TempDir(), GlobalMode: ModeVault}, bank)

	if result.Result != CommitUndetermined {
		t.Fatalf("result = %q, want %q", result.Result, CommitUndetermined)
	}
	if len(result.Written) != 0 {
		t.Errorf("written = %v, want nothing", result.Written)
	}
}

func TestAtomicWriteLeavesNoTemporaryFilesBehind(t *testing.T) {
	f := newFixture(t, "acme-hr")

	if result := Commit(f.submission(PhaseIdea), f.env, f.bank); result.Result != CommitWritten {
		t.Fatalf("commit failed: %s", result.Detail)
	}

	entries, err := os.ReadDir(f.res.DocsRoot)
	if err != nil {
		t.Fatalf("reading docs root: %v", err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp-") {
			t.Errorf("temporary file %q was left behind", entry.Name())
		}
	}
}

// managedParentFixture commits a system phase so the parent index gains a
// managed region, then returns a fixture for a module underneath it.
func managedParentFixture(t *testing.T) (*fixture, *fixture) {
	t.Helper()

	parent := newFixture(t, "acme-hr")
	if r := Commit(parent.submission(PhaseIdea), parent.env, parent.bank); r.Result != CommitWritten {
		t.Fatalf("parent commit failed: %s", r.Detail)
	}

	child := &fixture{t: t, bank: parent.bank, env: parent.env}
	node, err := ParseNode("acme-hr/payroll")
	if err != nil {
		t.Fatalf("ParseNode: %v", err)
	}
	child.node = node
	res, err := Resolve(node, parent.env)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	child.res = res
	return parent, child
}

func TestModuleCommitMakesTheModuleVisibleInTheParentIndex(t *testing.T) {
	parent, child := managedParentFixture(t)

	result := Commit(child.submission(PhaseIdea), child.env, child.bank)
	if result.Result != CommitWritten {
		t.Fatalf("module commit failed: %s", result.Detail)
	}
	if len(result.ParentIndex) != 1 {
		t.Fatalf("parentIndex = %v, want the system index refreshed", result.ParentIndex)
	}
	if result.ParentIndex[0].File != parent.res.IndexPath {
		t.Errorf("refreshed %q, want %q", result.ParentIndex[0].File, parent.res.IndexPath)
	}

	raw, err := os.ReadFile(parent.res.IndexPath)
	if err != nil {
		t.Fatalf("reading parent index: %v", err)
	}
	// A documented module that never appears in its parent's index is invisible
	// to anyone reading the system from the top.
	if !strings.Contains(string(raw), "[[payroll]]") {
		t.Error("the module is not listed in the parent index")
	}
}

func TestUnmanagedParentIsLeftUntouchedAndReported(t *testing.T) {
	// Real vaults hold documentation written before answer records existed.
	// Injecting a computed region there would claim nothing is done, contradicting
	// the file's own prose. It must be reported, never rewritten.
	f := newFixture(t, "acme-hr/payroll")

	parentIndex := filepath.Join(filepath.Dir(filepath.Dir(f.res.DocsRoot)), "acme-hr.md")
	legacy := "# acme-hr\n\n**Status**: documented\n\n- [x] Idea\n- [x] Requirements\n"
	if err := os.MkdirAll(filepath.Dir(parentIndex), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(parentIndex, []byte(legacy), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	result := Commit(f.submission(PhaseIdea), f.env, f.bank)
	if result.Result != CommitWritten {
		t.Fatalf("commit failed: %s", result.Detail)
	}
	if len(result.ParentIndex) != 0 {
		t.Errorf("parentIndex = %v, want nothing touched", result.ParentIndex)
	}
	if result.Detail == "" {
		t.Error("an unmanaged parent was skipped silently")
	}

	after, err := os.ReadFile(parentIndex)
	if err != nil {
		t.Fatalf("reading parent index: %v", err)
	}
	if string(after) != legacy {
		t.Error("the unmanaged parent index was modified")
	}
	if strings.Contains(string(after), indexRegionBegin) {
		t.Error("a managed region was injected into an unmanaged parent index")
	}
}

func TestModulesTableExcludesDirectoriesThatAreNotNodes(t *testing.T) {
	parent, child := managedParentFixture(t)

	if r := Commit(child.submission(PhaseIdea), child.env, child.bank); r.Result != CommitWritten {
		t.Fatalf("module commit failed: %s", r.Detail)
	}
	// A sibling directory with no matching index is not a node.
	noise := filepath.Join(filepath.Dir(child.res.DocsRoot), "agent_sdd_context_project")
	if err := os.MkdirAll(noise, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if r := Commit(child.submission(PhaseRec), child.env, child.bank); r.Result != CommitWritten {
		t.Fatalf("second module commit failed: %s", r.Detail)
	}

	raw, err := os.ReadFile(parent.res.IndexPath)
	if err != nil {
		t.Fatalf("reading parent index: %v", err)
	}
	if strings.Contains(string(raw), "agent_sdd_context_project") {
		t.Error("a directory without its own index was listed as a module")
	}
	if !strings.Contains(string(raw), "[[payroll]]") {
		t.Error("the real module went missing")
	}
}

func TestSystemCommitReportsNoParent(t *testing.T) {
	f := newFixture(t, "acme-hr")

	result := Commit(f.submission(PhaseIdea), f.env, f.bank)
	if result.Result != CommitWritten {
		t.Fatalf("commit failed: %s", result.Detail)
	}
	if len(result.ParentIndex) != 0 {
		t.Errorf("parentIndex = %v, want empty for a system node", result.ParentIndex)
	}
}

func TestCommitStoresTheAuthoredProseNotOnlyTheArtifact(t *testing.T) {
	// The artifact is derived and the prose is not recoverable from it, so a
	// phase could never be partially corrected without re-interviewing the user.
	f := newFixture(t, "acme-hr")

	result := Commit(f.submission(PhaseRec), f.env, f.bank)
	if result.Result != CommitWritten {
		t.Fatalf("commit failed: %s", result.Detail)
	}

	path := f.res.SectionInputPath(PhaseRec)
	if !containsString(result.Written, path) {
		t.Errorf("written = %v, want it to include the stored section input %s", result.Written, path)
	}

	stored, found, err := LoadSectionInput(path)
	if err != nil {
		t.Fatalf("LoadSectionInput: %v", err)
	}
	if !found {
		t.Fatal("section input was not stored")
	}

	original := f.submission(PhaseRec).Content
	if len(stored.Sections) != len(original.Sections) {
		t.Errorf("stored %d sections, want %d", len(stored.Sections), len(original.Sections))
	}
	for id, prose := range original.Sections {
		if stored.Sections[id] != prose {
			t.Errorf("section %q was altered in storage", id)
		}
	}
}

func TestStoredProseSurvivesARoundTripAndSupportsOneSectionCorrection(t *testing.T) {
	// This is the workflow refine needs: read the prose back, replace exactly one
	// section, and re-submit without re-asking the user anything.
	f := newFixture(t, "acme-hr")

	if result := Commit(f.submission(PhasePRD), f.env, f.bank); result.Result != CommitWritten {
		t.Fatalf("first commit failed")
	}

	stored, found, err := LoadSectionInput(f.res.SectionInputPath(PhasePRD))
	if err != nil || !found {
		t.Fatalf("LoadSectionInput: found=%v err=%v", found, err)
	}

	const refined = "As an HR clerk I want to close payroll in one day so the audit passes."
	stored.Sections["user-stories"] = refined

	corrected := f.submission(PhasePRD)
	corrected.Content = stored

	result := Commit(corrected, f.env, f.bank)
	if result.Result != CommitWritten {
		t.Fatalf("re-submission failed: %s", result.Detail)
	}

	artifact, err := os.ReadFile(f.res.ArtifactPath(PhasePRD, f.bank))
	if err != nil {
		t.Fatalf("reading artifact: %v", err)
	}
	if !strings.Contains(string(artifact), refined) {
		t.Error("the corrected section did not reach the rendered artifact")
	}
	// Every other section must be untouched by a single-section correction.
	for id, prose := range f.submission(PhasePRD).Content.Sections {
		if id == "user-stories" {
			continue
		}
		if !strings.Contains(string(artifact), prose) {
			t.Errorf("section %q was lost during the correction", id)
		}
	}
}

func TestLoadSectionInputRejectsAWrongSchema(t *testing.T) {
	f := newFixture(t, "acme-hr")

	path := f.res.SectionInputPath(PhasePRD)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"schemaName":"docagent.sections/v99","sections":{}}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if _, _, err := LoadSectionInput(path); err == nil {
		t.Fatal("LoadSectionInput accepted an unknown schema version")
	}
}

// auditSubmission builds a valid refine audit over one story.
func auditSubmission(f *fixture) Submission {
	return Submission{
		Node:  f.node,
		Phase: PhaseRefine,
		Audit: &AuditRecord{
			SchemaName: AuditRecordSchema,
			Node:       f.node.Raw,
			Phase:      PhaseRefine,
			Subjects: []AuditSubject{{
				ID: "H1",
				Verdicts: map[string]Verdict{
					"independent": VerdictPass, "negotiable": VerdictPass,
					"valuable": VerdictPass, "estimable": VerdictPass,
					"small": VerdictPass, "testable": VerdictPass,
				},
			}},
		},
	}
}

func TestAuditIsAnchoredToTheProseItJudged(t *testing.T) {
	f := newFixture(t, "acme-hr")
	if r := Commit(f.submission(PhasePRD), f.env, f.bank); r.Result != CommitWritten {
		t.Fatalf("prd commit failed: %s", r.Detail)
	}
	if r := Commit(auditSubmission(f), f.env, f.bank); r.Result != CommitWritten {
		t.Fatalf("audit commit failed: %s", r.Detail)
	}

	stored, found, err := LoadAuditRecord(f.res.AuditRecordPath(PhaseRefine), f.bank)
	if err != nil || !found {
		t.Fatalf("LoadAuditRecord: found=%v err=%v", found, err)
	}
	if stored.AuditedRevision == "" {
		t.Fatal("the audit was stored with no anchor to what it judged")
	}
	if got := phaseStatus(t, ComputeStatus(f.node, f.env, f.bank), PhaseRefine).State; got != StateComplete {
		t.Errorf("refine state = %q, want %q", got, StateComplete)
	}
}

func TestRewritingTheJudgedProseInvalidatesTheAudit(t *testing.T) {
	// This is the live failure: the PRD was corrected and the audit stayed
	// "complete", so 84 verdicts described stories that no longer existed. The
	// repo's own ruleset dismisses stale reviews on push for the same reason.
	f := newFixture(t, "acme-hr")
	// A full chain up to prd, so nextRecommended genuinely reflects the audit
	// rather than an earlier gap.
	for _, phase := range []PhaseID{PhaseIdea, PhaseRec, PhasePRD} {
		if r := Commit(f.submission(phase), f.env, f.bank); r.Result != CommitWritten {
			t.Fatalf("%s commit failed: %s", phase, r.Detail)
		}
	}
	if r := Commit(auditSubmission(f), f.env, f.bank); r.Result != CommitWritten {
		t.Fatalf("audit commit failed: %s", r.Detail)
	}
	if got := phaseStatus(t, ComputeStatus(f.node, f.env, f.bank), PhaseRefine).State; got != StateComplete {
		t.Fatalf("refine did not start complete: %q", got)
	}

	// Correct the very stories the audit judged.
	corrected := f.submission(PhasePRD)
	corrected.Content.Sections["user-stories"] = "As an HR clerk I want payroll closed in one day."
	if r := Commit(corrected, f.env, f.bank); r.Result != CommitWritten {
		t.Fatalf("correction failed: %s", r.Detail)
	}

	status := ComputeStatus(f.node, f.env, f.bank)
	if got := phaseStatus(t, status, PhaseRefine).State; got == StateComplete {
		t.Error("the audit still reports complete after the stories it judged were rewritten")
	}
	if !hasBlockedReasonCode(status, CodeAuditStale) {
		t.Errorf("no %q reason after the judged prose changed", CodeAuditStale)
	}
	// Downstream work must wait: tech builds from stories no gate has seen.
	if status.NextRecommended != PhaseRefine {
		t.Errorf("nextRecommended = %q, want %q", status.NextRecommended, PhaseRefine)
	}

	// Re-auditing against the corrected prose clears it.
	if r := Commit(auditSubmission(f), f.env, f.bank); r.Result != CommitWritten {
		t.Fatalf("re-audit failed: %s", r.Detail)
	}
	if got := phaseStatus(t, ComputeStatus(f.node, f.env, f.bank), PhaseRefine).State; got != StateComplete {
		t.Errorf("refine state = %q after re-auditing, want %q", got, StateComplete)
	}
}

func TestASubmissionMayNotSupplyItsOwnAnchor(t *testing.T) {
	// An anchor the auditor supplies is an anchor the auditor can move.
	f := newFixture(t, "acme-hr")
	if r := Commit(f.submission(PhasePRD), f.env, f.bank); r.Result != CommitWritten {
		t.Fatalf("prd commit failed: %s", r.Detail)
	}

	sub := auditSubmission(f)
	sub.Audit.AuditedRevision = "whatever-i-say-it-is"

	result := Commit(sub, f.env, f.bank)
	if result.Result != CommitRejected {
		t.Fatalf("result = %q, want %q", result.Result, CommitRejected)
	}
	if !containsString(result.Validation.RejectedBecause, CheckAuditRecordPresent) {
		t.Errorf("rejectedBecause = %v", result.Validation.RejectedBecause)
	}
}

func TestAnAuditWithNothingToAnchorAgainstIsNotCalledStale(t *testing.T) {
	// Adopted documentation has no stored prose, so no revision can be computed.
	// Absence is not evidence of a mismatch, and claiming staleness there would
	// block legacy work on a comparison that never happened.
	f := newFixture(t, "acme-hr")

	if r := Commit(auditSubmission(f), f.env, f.bank); r.Result != CommitWritten {
		t.Fatalf("audit commit failed: %s", r.Detail)
	}

	status := ComputeStatus(f.node, f.env, f.bank)
	if hasBlockedReasonCode(status, CodeAuditStale) {
		t.Error("staleness was claimed with no revision on either side")
	}
}
