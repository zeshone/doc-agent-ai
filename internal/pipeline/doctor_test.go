package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testNow = "2026-08-13T12:00:00Z"

// legacyVault writes documentation the way it existed before answer records:
// artifacts and a hand-written index, and nothing machine-readable.
func legacyVault(t *testing.T, rawNode string, indexBody string, phases ...PhaseID) (Node, Environment, Resolution, QuestionBank) {
	t.Helper()

	bank := mustLoadBank(t)
	node, err := ParseNode(rawNode)
	if err != nil {
		t.Fatalf("ParseNode: %v", err)
	}
	env := Environment{ProjectRoot: t.TempDir(), GlobalMode: ModeVault, GlobalBasePath: t.TempDir()}
	res, err := Resolve(node, env)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if err := os.MkdirAll(res.DocsRoot, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	for _, phase := range phases {
		spec, _ := bank.Phase(phase)
		path := filepath.Join(res.DocsRoot, spec.ArtifactName(res.ArtifactPrefix))
		if err := os.WriteFile(path, []byte("# legacy artifact\n\nreal content written months ago\n"), 0o644); err != nil {
			t.Fatalf("writing legacy artifact: %v", err)
		}
	}
	if indexBody != "" {
		if err := os.WriteFile(res.IndexPath, []byte(indexBody), 0o644); err != nil {
			t.Fatalf("writing legacy index: %v", err)
		}
	}
	return node, env, res, bank
}

// realWorldIndex mirrors a real vault: its own checkbox block, its own status
// line, and an archetype phrased in none of the literals a matcher would expect.
const realWorldIndex = `# Deze 3.0

**Status**: documented
**Tipo de sistema**: Producto evolutivo (crece con módulos)

## Fases de documentación

- [x] Idea
- [x] Requirements
- [x] PRD
`

func TestDoctorCheckChangesNothing(t *testing.T) {
	node, env, res, bank := legacyVault(t, "deze", realWorldIndex, PhaseIdea, PhaseRec, PhasePRD)

	before, err := os.ReadFile(res.IndexPath)
	if err != nil {
		t.Fatalf("reading index: %v", err)
	}

	report := Doctor(node, env, bank, DoctorOptions{Apply: false, Now: testNow})

	if report.Applied {
		t.Error("a check run reported itself as applied")
	}
	if len(report.Findings) == 0 {
		t.Fatal("check found nothing in a vault full of unmanaged documentation")
	}
	if _, err := os.Stat(res.AdoptionPath()); err == nil {
		t.Error("a check run wrote an adoption record")
	}
	after, err := os.ReadFile(res.IndexPath)
	if err != nil {
		t.Fatalf("reading index: %v", err)
	}
	if string(after) != string(before) {
		t.Error("a check run modified the index")
	}
}

func TestDoctorAdoptsPhasesThatHaveArtifactsButNoRecords(t *testing.T) {
	node, env, res, bank := legacyVault(t, "deze", realWorldIndex, PhaseIdea, PhaseRec, PhasePRD)

	report := Doctor(node, env, bank, DoctorOptions{Apply: true, Now: testNow})
	if report.HasBlockers() {
		t.Fatalf("blocked: %v", report.Blocked)
	}

	adoption, found, err := LoadAdoption(res.AdoptionPath(), bank)
	if err != nil || !found {
		t.Fatalf("LoadAdoption: found=%v err=%v", found, err)
	}
	for _, phase := range []PhaseID{PhaseIdea, PhaseRec, PhasePRD} {
		if _, ok := adoption.Phases[phase]; !ok {
			t.Errorf("phase %q was not adopted", phase)
		}
	}
	if _, ok := adoption.Phases[PhaseTech]; ok {
		t.Error("a phase with no artifact was adopted")
	}
}

func TestAdoptedPhasesNeverClaimVerifiedCoverage(t *testing.T) {
	node, env, res, bank := legacyVault(t, "deze", realWorldIndex, PhaseIdea, PhaseRec, PhasePRD)

	if r := Doctor(node, env, bank, DoctorOptions{Apply: true, Now: testNow}); r.HasBlockers() {
		t.Fatalf("blocked: %v", r.Blocked)
	}

	status := ComputeStatus(node, env, bank)

	for _, phase := range []PhaseID{PhaseIdea, PhaseRec, PhasePRD} {
		if got := phaseStatus(t, status, phase).State; got != StateAdopted {
			t.Errorf("%s state = %q, want %q", phase, got, StateAdopted)
		}
	}
	// This is the whole point: adopted is never complete.
	if len(status.Completed) != 0 {
		t.Errorf("completed = %v, want empty — no coverage was ever counted", status.Completed)
	}
	if len(status.Adopted) != 3 {
		t.Errorf("adopted = %v, want three phases", status.Adopted)
	}
	for _, phase := range status.Missing {
		if phase == PhaseIdea || phase == PhaseRec || phase == PhasePRD {
			t.Errorf("adopted phase %q is reported as missing", phase)
		}
	}

	raw, err := os.ReadFile(res.IndexPath)
	if err != nil {
		t.Fatalf("reading index: %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, "| idea | [~] |") {
		t.Error("an adopted phase did not get its own mark in the index")
	}
	if strings.Contains(text, "| idea | [x] |") {
		t.Error("an adopted phase was marked as verified coverage")
	}
	if !strings.Contains(text, "unverified") {
		t.Error("the index does not say the adopted coverage is unverified")
	}
	// The author's own content must survive untouched.
	if !strings.Contains(text, "**Status**: documented") || !strings.Contains(text, "- [x] Idea") {
		t.Error("doctor destroyed the author's existing index content")
	}
}

func TestAdoptedPhasesDoNotBlockNewWork(t *testing.T) {
	// The reason adopted exists: a new module must not require re-interviewing an
	// already documented parent.
	node, env, _, bank := legacyVault(t, "deze", realWorldIndex, PhaseIdea, PhaseRec, PhasePRD)

	if r := Doctor(node, env, bank, DoctorOptions{Apply: true, Now: testNow}); r.HasBlockers() {
		t.Fatalf("blocked: %v", r.Blocked)
	}

	status := ComputeStatus(node, env, bank)
	if status.NextRecommended != PhaseRefine {
		t.Errorf("nextRecommended = %q, want %q — adoption should unblock the next phase",
			status.NextRecommended, PhaseRefine)
	}
	if got := phaseStatus(t, status, PhaseRefine).State; got != StatePending {
		t.Errorf("refine state = %q, want %q", got, StatePending)
	}
}

func TestDoctorReadsTheArchetypeARealIndexActuallyUses(t *testing.T) {
	// "**Tipo de sistema**: Producto evolutivo" matches none of the literals the
	// prose check looked for, which is exactly why that check had to go.
	node, env, res, bank := legacyVault(t, "deze", realWorldIndex, PhaseIdea)

	report := Doctor(node, env, bank, DoctorOptions{Apply: true, Now: testNow})
	if report.HasBlockers() {
		t.Fatalf("blocked: %v", report.Blocked)
	}

	adoption, _, err := LoadAdoption(res.AdoptionPath(), bank)
	if err != nil {
		t.Fatalf("LoadAdoption: %v", err)
	}
	if adoption.Archetype != ArchetypeEvolving {
		t.Errorf("archetype = %q, want %q", adoption.Archetype, ArchetypeEvolving)
	}
	if got := ComputeStatus(node, env, bank).Target.Archetype; got != ArchetypeEvolving {
		t.Errorf("status archetype = %q, want %q", got, ArchetypeEvolving)
	}
	// The source must be stated, because inference is not a recorded fact.
	var stated bool
	for _, f := range report.Findings {
		if f.Kind == FindingArchetype && strings.Contains(f.Detail, "inferred") {
			stated = true
		}
	}
	if !stated {
		t.Error("doctor inferred the archetype without saying so")
	}
}

func TestDoctorBlocksRatherThanGuessingAnAmbiguousArchetype(t *testing.T) {
	node, env, res, bank := legacyVault(t, "deze", "# deze\n\nNo hint about the shape of this system.\n", PhaseIdea)

	report := Doctor(node, env, bank, DoctorOptions{Apply: true, Now: testNow})

	if !report.HasBlockers() {
		t.Fatal("doctor guessed an archetype it could not determine")
	}
	if !strings.Contains(strings.Join(report.Blocked, " "), "--archetype") {
		t.Errorf("the blocker does not name the way out: %v", report.Blocked)
	}
	// Blocked means nothing was written.
	if _, err := os.Stat(res.AdoptionPath()); err == nil {
		t.Error("doctor wrote an adoption record while blocked")
	}
}

func TestDoctorAcceptsAnExplicitArchetype(t *testing.T) {
	node, env, res, bank := legacyVault(t, "deze", "# deze\n\nNo hint at all.\n", PhaseIdea)

	report := Doctor(node, env, bank, DoctorOptions{Apply: true, Archetype: ArchetypeBounded, Now: testNow})
	if report.HasBlockers() {
		t.Fatalf("blocked: %v", report.Blocked)
	}

	adoption, _, err := LoadAdoption(res.AdoptionPath(), bank)
	if err != nil {
		t.Fatalf("LoadAdoption: %v", err)
	}
	if adoption.Archetype != ArchetypeBounded {
		t.Errorf("archetype = %q, want %q", adoption.Archetype, ArchetypeBounded)
	}
}

func TestDoctorLeavesAlreadyManagedPhasesAlone(t *testing.T) {
	f := newFixture(t, "acme-hr")
	if r := Commit(f.submission(PhaseIdea), f.env, f.bank); r.Result != CommitWritten {
		t.Fatalf("commit failed: %s", r.Detail)
	}

	report := Doctor(f.node, f.env, f.bank, DoctorOptions{Apply: true, Now: testNow})

	var adoptedIdea, reportedManaged bool
	for _, finding := range report.Findings {
		if finding.Phase == PhaseIdea && finding.Kind == FindingAdoptPhase {
			adoptedIdea = true
		}
		if finding.Phase == PhaseIdea && finding.Kind == FindingAlreadyManaged {
			reportedManaged = true
		}
	}
	if adoptedIdea {
		t.Error("a phase with counted coverage was adopted, downgrading it")
	}
	if !reportedManaged {
		t.Error("doctor did not report the phase as already managed")
	}

	// The counted phase must still read as complete, not adopted.
	if got := phaseStatus(t, ComputeStatus(f.node, f.env, f.bank), PhaseIdea).State; got != StateComplete {
		t.Errorf("idea state = %q, want %q", got, StateComplete)
	}
}

func TestDoctorNeverAdoptsAnAuditPhase(t *testing.T) {
	// refine's artifact IS the PRD file, so its presence says nothing about
	// whether an audit ever happened. Adopting it would invent an audit.
	node, env, res, bank := legacyVault(t, "deze", realWorldIndex, PhaseIdea, PhaseRec, PhasePRD)

	if r := Doctor(node, env, bank, DoctorOptions{Apply: true, Now: testNow}); r.HasBlockers() {
		t.Fatalf("blocked: %v", r.Blocked)
	}

	adoption, _, err := LoadAdoption(res.AdoptionPath(), bank)
	if err != nil {
		t.Fatalf("LoadAdoption: %v", err)
	}
	if _, ok := adoption.Phases[PhaseRefine]; ok {
		t.Error("the audit phase was adopted from the PRD file's mere existence")
	}
}

func TestDoctorRecursiveAdoptsNestedModules(t *testing.T) {
	node, env, res, bank := legacyVault(t, "deze", realWorldIndex, PhaseIdea, PhaseRec)

	// A legacy module, laid out the way a real vault nests them.
	moduleDir := filepath.Join(res.DocsRoot, "modules", "catalogos")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	for _, name := range []string{"catalogos.md", "catalogos_idea.md", "catalogos_requirements.md"} {
		if err := os.WriteFile(filepath.Join(moduleDir, name), []byte("# legacy\n\ncontent\n"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	report := Doctor(node, env, bank, DoctorOptions{Apply: true, Recursive: true, Now: testNow})
	if report.HasBlockers() {
		t.Fatalf("blocked: %v", report.Blocked)
	}

	child, err := ParseNode("deze/catalogos")
	if err != nil {
		t.Fatalf("ParseNode: %v", err)
	}
	childStatus := ComputeStatus(child, env, bank)
	for _, phase := range []PhaseID{PhaseIdea, PhaseRec} {
		if got := phaseStatus(t, childStatus, phase).State; got != StateAdopted {
			t.Errorf("module %s state = %q, want %q", phase, got, StateAdopted)
		}
	}
}

func TestDoctorBlocksWhenThereIsNothingToAdopt(t *testing.T) {
	bank := mustLoadBank(t)
	node, err := ParseNode("ghost")
	if err != nil {
		t.Fatalf("ParseNode: %v", err)
	}
	env := Environment{ProjectRoot: t.TempDir(), GlobalMode: ModeVault, GlobalBasePath: t.TempDir()}

	report := Doctor(node, env, bank, DoctorOptions{Apply: true, Now: testNow})
	if !report.HasBlockers() {
		t.Fatal("doctor claimed success against a docs root that does not exist")
	}
}

func TestAdoptionRecordRejectsStructuralProblems(t *testing.T) {
	bank := mustLoadBank(t)

	tests := []struct {
		name   string
		mutate func(*Adoption)
	}{
		{"wrong schema", func(a *Adoption) { a.SchemaName = "docagent.adoption/v99" }},
		{"empty node", func(a *Adoption) { a.Node = "" }},
		{"unknown phase", func(a *Adoption) {
			a.Phases["tech-specs"] = AdoptedPhase{Artifact: "x.md", Evidence: "y"}
		}},
		{"invalid archetype", func(a *Adoption) { a.Archetype = "somewhat-evolving" }},
		{"adopted phase with no artifact", func(a *Adoption) {
			a.Phases[PhaseIdea] = AdoptedPhase{Evidence: "y"}
		}},
		{"adopted phase with no evidence", func(a *Adoption) {
			a.Phases[PhaseIdea] = AdoptedPhase{Artifact: "x.md"}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adoption := Adoption{
				SchemaName: AdoptionSchema,
				Node:       "deze",
				AdoptedAt:  testNow,
				Archetype:  ArchetypeEvolving,
				Phases:     map[PhaseID]AdoptedPhase{},
			}
			tt.mutate(&adoption)
			if err := adoption.Validate(bank); err == nil {
				t.Fatal("Validate accepted an invalid adoption record")
			}
		})
	}
}

func TestClosedValueTopicsRequireAMachineReadableChoice(t *testing.T) {
	bank := mustLoadBank(t)

	base := func() Answer {
		return Answer{
			TopicID:    "archetype",
			Status:     AnswerAnswered,
			Source:     SourceUserAnswer,
			Verbatim:   "es un producto que va a crecer con modulos",
			CapturedAt: testNow,
		}
	}
	record := func(a Answer) AnswerRecord {
		return AnswerRecord{SchemaName: AnswerRecordSchema, Node: "deze", Phase: PhaseRec, Answers: []Answer{a}}
	}

	t.Run("a valid choice is accepted", func(t *testing.T) {
		a := base()
		a.Value = ArchetypeEvolving
		if err := record(a).Validate(bank); err != nil {
			t.Fatalf("Validate: %v", err)
		}
	})
	t.Run("a missing choice is rejected", func(t *testing.T) {
		if err := record(base()).Validate(bank); err == nil {
			t.Fatal("an answered closed-value topic was accepted with no value")
		}
	})
	t.Run("a value outside the set is rejected", func(t *testing.T) {
		a := base()
		a.Value = "somewhat-evolving"
		if err := record(a).Validate(bank); err == nil {
			t.Fatal("a value outside the declared set was accepted")
		}
	})
	t.Run("a value on an open topic is rejected", func(t *testing.T) {
		a := answerForTopic(bank, PhaseRec, "stakeholders")
		a.Value = ArchetypeEvolving
		if err := record(a).Validate(bank); err == nil {
			t.Fatal("a value was accepted on a topic with no closed set")
		}
	})
	t.Run("a deferred closed-value topic owes no choice", func(t *testing.T) {
		a := base()
		a.Status = AnswerDeferred
		if err := record(a).Validate(bank); err != nil {
			t.Fatalf("Validate: %v", err)
		}
	})
}

func TestStatusReportsTheArchetypeRecordedByAnInterview(t *testing.T) {
	f := newFixture(t, "acme-hr")
	f.completePhase(PhaseIdea)
	f.completePhase(PhaseRec)

	// completePhase supplies the first declared value, which is bounded.
	if got := ComputeStatus(f.node, f.env, f.bank).Target.Archetype; got != ArchetypeBounded {
		t.Errorf("archetype = %q, want %q", got, ArchetypeBounded)
	}
}

func TestDoctorAdoptsRefineOnlyFromADedicatedReport(t *testing.T) {
	// The PRD file's presence proves nothing about the audit. A dedicated
	// <node>_refinement.md exists only because someone produced it, so it is real
	// evidence — and real vaults hold exactly that file.
	node, env, res, bank := legacyVault(t, "deze", realWorldIndex, PhaseIdea, PhaseRec, PhasePRD)

	if r := Doctor(node, env, bank, DoctorOptions{Apply: true, Now: testNow}); r.HasBlockers() {
		t.Fatalf("blocked: %v", r.Blocked)
	}
	adoption, _, err := LoadAdoption(res.AdoptionPath(), bank)
	if err != nil {
		t.Fatalf("LoadAdoption: %v", err)
	}
	if _, ok := adoption.Phases[PhaseRefine]; ok {
		t.Fatal("refine was adopted from the shared PRD artifact")
	}

	// Now add the dedicated report a previous version of the agent wrote.
	report := filepath.Join(res.DocsRoot, "deze_refinement.md")
	if err := os.WriteFile(report, []byte("# audit\n\nstory findings\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if r := Doctor(node, env, bank, DoctorOptions{Apply: true, Now: testNow}); r.HasBlockers() {
		t.Fatalf("blocked: %v", r.Blocked)
	}
	adoption, _, err = LoadAdoption(res.AdoptionPath(), bank)
	if err != nil {
		t.Fatalf("LoadAdoption: %v", err)
	}
	adopted, ok := adoption.Phases[PhaseRefine]
	if !ok {
		t.Fatal("refine was not adopted despite a dedicated audit report")
	}
	if adopted.Artifact != "deze_refinement.md" {
		t.Errorf("adopted artifact = %q, want the dedicated report", adopted.Artifact)
	}
}

func TestDoctorRecognisesTheLegacyIdeaFilename(t *testing.T) {
	// Real vaults hold <node>_idea.md; the current convention writes
	// <node>_idea-brief.md. Adoption has to recognise what exists, not what the
	// current naming would have produced.
	bank := mustLoadBank(t)
	spec, _ := bank.Phase(PhaseIdea)

	if got := spec.ArtifactName("deze"); got != "deze_idea-brief.md" {
		t.Errorf("current name = %q", got)
	}
	legacy := spec.LegacyArtifactNames("deze")
	if len(legacy) == 0 || legacy[0] != "deze_idea.md" {
		t.Errorf("legacy names = %v, want deze_idea.md", legacy)
	}

	node, env, res, bank := legacyVault(t, "deze", realWorldIndex)
	if err := os.WriteFile(filepath.Join(res.DocsRoot, "deze_idea.md"), []byte("# idea\n\ncontent\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if r := Doctor(node, env, bank, DoctorOptions{Apply: true, Now: testNow}); r.HasBlockers() {
		t.Fatalf("blocked: %v", r.Blocked)
	}
	adoption, _, err := LoadAdoption(res.AdoptionPath(), bank)
	if err != nil {
		t.Fatalf("LoadAdoption: %v", err)
	}
	if adoption.Phases[PhaseIdea].Artifact != "deze_idea.md" {
		t.Errorf("adopted artifact = %q, want the legacy name", adoption.Phases[PhaseIdea].Artifact)
	}
}

func TestRecursiveDoctorLeavesNoStaleModuleStatusInTheParent(t *testing.T) {
	// A parent's module table reads each child's recorded status, so the parent
	// must be processed last. Otherwise every child reads as unmanaged and stays
	// that way in the file even after being adopted in the same run.
	node, env, res, bank := legacyVault(t, "deze", realWorldIndex, PhaseIdea, PhaseRec)

	for _, name := range []string{"catalogos", "bitacora"} {
		dir := filepath.Join(res.DocsRoot, "modules", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		for _, file := range []string{name + ".md", name + "_idea.md", name + "_requirements.md"} {
			if err := os.WriteFile(filepath.Join(dir, file), []byte("# legacy\n\ncontent\n"), 0o644); err != nil {
				t.Fatalf("write: %v", err)
			}
		}
	}

	if r := Doctor(node, env, bank, DoctorOptions{Apply: true, Recursive: true, Now: testNow}); r.HasBlockers() {
		t.Fatalf("blocked: %v", r.Blocked)
	}

	raw, err := os.ReadFile(res.IndexPath)
	if err != nil {
		t.Fatalf("reading parent index: %v", err)
	}
	text := string(raw)

	if strings.Contains(text, "not managed") {
		t.Error("the parent index reports a module as unmanaged after adopting it in the same run")
	}
	for _, name := range []string{"catalogos", "bitacora"} {
		if !strings.Contains(text, "[["+name+"]]") {
			t.Errorf("module %q is missing from the parent index", name)
		}
	}
	// Each module has two of seven phases adopted, so "in progress" is the exact
	// answer. Asserting the computed value rather than merely "something" is what
	// makes this a regression test for the ordering bug.
	for _, name := range []string{"catalogos", "bitacora"} {
		want := "| [[" + name + "]] | " + NodeStatusInProgress + " |"
		if !strings.Contains(text, want) {
			t.Errorf("parent index is missing %q", want)
		}
	}
}

func TestFullyAdoptedModuleReadsAdoptedInTheParentTable(t *testing.T) {
	node, env, res, bank := legacyVault(t, "deze", realWorldIndex, PhaseIdea)

	// Every phase this module owes, present as legacy documentation.
	dir := filepath.Join(res.DocsRoot, "modules", "catalogos")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	files := []string{
		"catalogos.md", "catalogos_idea.md", "catalogos_requirements.md",
		"catalogos_prd.md", "catalogos_refinement.md", "catalogos_tech-spec.md",
		"catalogos_db-design.md", "catalogos_issues.md",
	}
	for _, file := range files {
		if err := os.WriteFile(filepath.Join(dir, file), []byte("# legacy\n\ncontent\n"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	if r := Doctor(node, env, bank, DoctorOptions{Apply: true, Recursive: true, Now: testNow}); r.HasBlockers() {
		t.Fatalf("blocked: %v", r.Blocked)
	}

	raw, err := os.ReadFile(res.IndexPath)
	if err != nil {
		t.Fatalf("reading parent index: %v", err)
	}
	want := "| [[catalogos]] | " + NodeStatusAdopted + " |"
	if !strings.Contains(string(raw), want) {
		t.Errorf("parent index is missing %q", want)
	}
}

func TestModuleInheritsTheSystemArchetype(t *testing.T) {
	// A module does not decide the shape of its system. Reporting "unknown" for a
	// fact the system already recorded would be a foot-gun for any caller that
	// queries the module instead of the system.
	node, env, res, bank := legacyVault(t, "deze", realWorldIndex, PhaseIdea)

	if r := Doctor(node, env, bank, DoctorOptions{Apply: true, Now: testNow}); r.HasBlockers() {
		t.Fatalf("blocked: %v", r.Blocked)
	}
	_ = res

	child, err := ParseNode("deze/personas")
	if err != nil {
		t.Fatalf("ParseNode: %v", err)
	}
	if got := ComputeStatus(child, env, bank).Target.Archetype; got != ArchetypeEvolving {
		t.Errorf("module archetype = %q, want the inherited %q", got, ArchetypeEvolving)
	}
}

func TestFullyAdoptedNodeDoesNotClaimCompletion(t *testing.T) {
	node, env, res, bank := legacyVault(t, "deze", realWorldIndex,
		PhaseIdea, PhaseRec, PhasePRD, PhaseTech, PhaseDDD, PhasePTI)

	// refine shares its artifact with prd, so it is adopted only from a dedicated
	// report — the file real vaults actually hold.
	if err := os.WriteFile(filepath.Join(res.DocsRoot, "deze_refinement.md"),
		[]byte("# audit\n\nstory findings\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if r := Doctor(node, env, bank, DoctorOptions{Apply: true, Now: testNow}); r.HasBlockers() {
		t.Fatalf("blocked: %v", r.Blocked)
	}

	status := ComputeStatus(node, env, bank)
	if len(status.Missing) != 0 {
		t.Errorf("missing = %v, want empty", status.Missing)
	}
	// The reason must not read as verified coverage.
	if strings.Contains(status.NextAction.Reason, "every applicable phase is complete") {
		t.Errorf("a fully adopted node claims completion: %q", status.NextAction.Reason)
	}
	if !strings.Contains(status.NextAction.Reason, "unverified") {
		t.Errorf("the reason does not say coverage is unverified: %q", status.NextAction.Reason)
	}
}
