package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sddNow = "2026-08-14T12:00:00Z"

// documentedNode runs every phase so the compaction has real sources to read.
func documentedNode(t *testing.T) *fixture {
	t.Helper()
	f := newFixture(t, "acme-hr")
	for _, phase := range []PhaseID{PhaseIdea, PhaseRec, PhasePRD, PhaseTech, PhaseDDD} {
		if r := Commit(f.submission(phase), f.env, f.bank); r.Result != CommitWritten {
			t.Fatalf("%s commit failed: %s", phase, r.Detail)
		}
	}
	if r := Commit(auditSubmission(f), f.env, f.bank); r.Result != CommitWritten {
		t.Fatalf("audit commit failed: %s", r.Detail)
	}
	return f
}

func validSDDInput() SDDInput {
	return SDDInput{
		SchemaName: SDDInputSchema,
		Business:   "# Contexto de negocio\n\nEl problema y a quien afecta.",
		Technical:  "# Contexto tecnico\n\nArquitectura y contratos.",
		Decisions: []SDDDecision{
			{
				ID: "D1", Layer: LayerBusiness,
				What:      "Ordenar patrocinados primero hasta 50 km",
				Why:       "El patrocinio es el ingreso del producto",
				SoThat:    "El patrocinador recibe lo que paga sin ocultar la distancia real",
				DecidedBy: "Confirmado por el interesado en rec/stakeholder-conflicts",
			},
			{
				ID: "D2", Layer: LayerTechnical,
				What:      "Cachear centroides de codigo postal en tabla propia",
				Why:       "La API de geolocalizacion no siempre responde",
				SoThat:    "La landing degrada a un radio aproximado en vez de fallar",
				DecidedBy: "Registrado en tech/architecture",
			},
		},
	}
}

func TestSDDContextIsWrittenWithItsManifest(t *testing.T) {
	f := documentedNode(t)

	result := CommitSDDContext(f.node, f.env, f.bank, validSDDInput(), sddNow)
	if result.Result != CommitWritten {
		t.Fatalf("result = %q, want %q (%v)", result.Result, CommitWritten, result.RejectedBecause)
	}
	if len(result.Written) != 3 {
		t.Fatalf("written = %v, want both layers and the manifest", result.Written)
	}

	res, err := Resolve(f.node, f.env)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	manifest, found, err := LoadSDDManifest(res.SDDManifestPath())
	if err != nil || !found {
		t.Fatalf("LoadSDDManifest: found=%v err=%v", found, err)
	}

	// Every source is named and fingerprinted: the derivation is no longer an
	// unsupported claim about which documents it read.
	if len(manifest.Sources) != 6 {
		t.Errorf("sources = %d, want the six that feed a compaction", len(manifest.Sources))
	}
	for _, source := range manifest.Sources {
		if source.Present && source.Revision == "" {
			t.Errorf("source %q is present with no fingerprint", source.Artifact)
		}
	}
	if !manifest.CoverageVerified {
		t.Error("coverageVerified = false for a node whose phases were all counted")
	}
}

func TestTheRefinementReportIsOneOfTheSources(t *testing.T) {
	// It did not exist as a rendered artifact when the compactor was written. An
	// agent implementing a story should be able to see that its estimability
	// failure was closed, and with what.
	f := documentedNode(t)

	if r := CommitSDDContext(f.node, f.env, f.bank, validSDDInput(), sddNow); r.Result != CommitWritten {
		t.Fatalf("commit failed: %v", r.RejectedBecause)
	}
	res, _ := Resolve(f.node, f.env)
	manifest, _, _ := LoadSDDManifest(res.SDDManifestPath())

	var found bool
	for _, source := range manifest.Sources {
		if source.Phase == PhaseRefine {
			found = true
			if !strings.HasSuffix(source.Artifact, "_refinement.md") {
				t.Errorf("refine contributes %q, want the rendered report", source.Artifact)
			}
			if !source.Present {
				t.Error("the refinement report was not read even though it exists")
			}
		}
		if source.Phase == PhasePTI {
			t.Error("the issue list was folded in, which fights the point of compacting")
		}
	}
	if !found {
		t.Error("the refinement report is not among the sources")
	}
}

func TestDecisionsAreRenderedInABoundedShape(t *testing.T) {
	// The reason the command exists: an agent reads what was settled and why
	// without opening seven documents.
	f := documentedNode(t)

	if r := CommitSDDContext(f.node, f.env, f.bank, validSDDInput(), sddNow); r.Result != CommitWritten {
		t.Fatalf("commit failed: %v", r.RejectedBecause)
	}

	res, _ := Resolve(f.node, f.env)
	business, err := os.ReadFile(filepath.Join(res.SDDContextDir(), res.SDDOutputName(LayerBusiness)))
	if err != nil {
		t.Fatalf("reading business layer: %v", err)
	}
	text := string(business)

	for _, phrase := range []string{
		"## Decisions",
		"### D1 — Ordenar patrocinados primero hasta 50 km",
		"**Why:** El patrocinio es el ingreso del producto",
		"**So that:**",
		"**Decided by:** Confirmado por el interesado en rec/stakeholder-conflicts",
	} {
		if !strings.Contains(text, phrase) {
			t.Errorf("business layer is missing %q", phrase)
		}
	}
	// A technical decision must not leak into the business layer.
	if strings.Contains(text, "D2") {
		t.Error("a technical decision was rendered into the business context")
	}
}

func TestACompactionWithNoDecisionsIsRefused(t *testing.T) {
	f := documentedNode(t)

	input := validSDDInput()
	input.Decisions = nil

	result := CommitSDDContext(f.node, f.env, f.bank, input, sddNow)
	if result.Result != CommitRejected {
		t.Fatalf("result = %q, want %q", result.Result, CommitRejected)
	}
	if !containsString(result.RejectedBecause, CheckSDDDecisionsBound) {
		t.Errorf("rejectedBecause = %v, want %q", result.RejectedBecause, CheckSDDDecisionsBound)
	}
}

func TestADecisionMissingAnyOfItsFourFieldsIsRefused(t *testing.T) {
	f := documentedNode(t)

	for _, field := range []string{"what", "why", "soThat", "decidedBy"} {
		t.Run("without "+field, func(t *testing.T) {
			input := validSDDInput()
			switch field {
			case "what":
				input.Decisions[0].What = ""
			case "why":
				input.Decisions[0].Why = ""
			case "soThat":
				input.Decisions[0].SoThat = ""
			case "decidedBy":
				input.Decisions[0].DecidedBy = ""
			}
			result := CommitSDDContext(f.node, f.env, f.bank, input, sddNow)
			if result.Result != CommitRejected {
				t.Fatalf("a decision with no %s was accepted", field)
			}
		})
	}
}

func TestAnOpenQuestionClaimedButAbsentFromEverySourceIsRefused(t *testing.T) {
	// Claiming to have preserved something nobody wrote is inventing an open
	// question, which is the fabrication failure applied to synthesis.
	f := documentedNode(t)

	input := validSDDInput()
	input.PreservedTBDs = []string{"TBD: nobody ever wrote this"}

	result := CommitSDDContext(f.node, f.env, f.bank, input, sddNow)
	if result.Result != CommitRejected {
		t.Fatalf("result = %q, want %q", result.Result, CommitRejected)
	}
	if !containsString(result.RejectedBecause, CheckSDDTBDsNotInvented) {
		t.Errorf("rejectedBecause = %v, want %q", result.RejectedBecause, CheckSDDTBDsNotInvented)
	}
}

func TestAnOpenQuestionInASourceButDroppedFromTheCompactionIsRefused(t *testing.T) {
	f := documentedNode(t)

	// Put a real open question into a source artifact.
	res, _ := Resolve(f.node, f.env)
	const open = "TBD: quien vigila el buzon de respaldo"
	prd := res.ArtifactPath(PhasePRD, f.bank)
	raw, err := os.ReadFile(prd)
	if err != nil {
		t.Fatalf("reading prd: %v", err)
	}
	if err := os.WriteFile(prd, append(raw, []byte("\n"+open+"\n")...), 0o644); err != nil {
		t.Fatalf("writing prd: %v", err)
	}

	input := validSDDInput()
	input.PreservedTBDs = []string{open} // claimed, but neither layer contains it

	result := CommitSDDContext(f.node, f.env, f.bank, input, sddNow)
	if result.Result != CommitRejected {
		t.Fatalf("result = %q, want %q", result.Result, CommitRejected)
	}
	if !containsString(result.RejectedBecause, CheckSDDTBDsPreserved) {
		t.Errorf("rejectedBecause = %v, want %q", result.RejectedBecause, CheckSDDTBDsPreserved)
	}

	// Carrying it across is accepted.
	input.Business += "\n\n" + open
	if r := CommitSDDContext(f.node, f.env, f.bank, input, sddNow); r.Result != CommitWritten {
		t.Fatalf("a preserved open question was still refused: %v", r.RejectedBecause)
	}
}

func TestRewritingASourceMakesTheCompactionStale(t *testing.T) {
	// A stale context is worse than none: it looks current and is not.
	f := documentedNode(t)

	if r := CommitSDDContext(f.node, f.env, f.bank, validSDDInput(), sddNow); r.Result != CommitWritten {
		t.Fatalf("commit failed: %v", r.RejectedBecause)
	}
	status := f.status()
	if status.SDDContext == nil || status.SDDContext.State != SDDFresh {
		t.Fatalf("sdd context did not start fresh: %+v", status.SDDContext)
	}

	corrected := f.submission(PhasePRD)
	corrected.Content.Sections["user-stories"] = "Una historia distinta."
	if r := Commit(corrected, f.env, f.bank); r.Result != CommitWritten {
		t.Fatalf("prd correction failed: %s", r.Detail)
	}

	status = f.status()
	if status.SDDContext.State != SDDStale {
		t.Errorf("state = %q, want %q after a source changed", status.SDDContext.State, SDDStale)
	}
	if len(status.SDDContext.Drifted) == 0 {
		t.Error("the drifted source is not named")
	}
}

func TestNoCompactionReportsAbsentRatherThanFresh(t *testing.T) {
	f := documentedNode(t)

	status := f.status()
	if status.SDDContext == nil || status.SDDContext.State != SDDAbsent {
		t.Errorf("state = %+v, want %q", status.SDDContext, SDDAbsent)
	}
}

func TestACompactionOfAdoptedDocumentationSaysSo(t *testing.T) {
	// Inherited documentation has unverified coverage. A context derived from it
	// inherits that, and laundering it into something that looks checked is the
	// one thing adoption exists to prevent.
	node, env, res, bank := legacyVault(t, "deze", realWorldIndex,
		PhaseIdea, PhaseRec, PhasePRD, PhaseTech, PhaseDDD)
	if r := Doctor(node, env, bank, DoctorOptions{Apply: true, Now: sddNow}); r.HasBlockers() {
		t.Fatalf("doctor blocked: %v", r.Blocked)
	}

	result := CommitSDDContext(node, env, bank, validSDDInput(), sddNow)
	if result.Result != CommitWritten {
		t.Fatalf("result = %q, want %q (%v)", result.Result, CommitWritten, result.RejectedBecause)
	}

	manifest, _, err := LoadSDDManifest(res.SDDManifestPath())
	if err != nil {
		t.Fatalf("LoadSDDManifest: %v", err)
	}
	if manifest.CoverageVerified {
		t.Error("a compaction of adopted documentation claims verified coverage")
	}

	var flagged int
	for _, source := range manifest.Sources {
		if source.Adopted {
			flagged++
		}
	}
	if flagged == 0 {
		t.Error("no source is marked adopted")
	}
}

func TestSDDCommitRefusesWhenThereIsNothingToCompact(t *testing.T) {
	f := newFixture(t, "acme-hr")

	result := CommitSDDContext(f.node, f.env, f.bank, validSDDInput(), sddNow)
	if result.Result != CommitRejected {
		t.Fatalf("result = %q, want %q", result.Result, CommitRejected)
	}
	if !containsString(result.RejectedBecause, CheckSDDSourcesPresent) {
		t.Errorf("rejectedBecause = %v", result.RejectedBecause)
	}
}

func TestLegacyArtifactNamesAreReadAsSources(t *testing.T) {
	// An inherited vault holds <node>_idea.md while the current convention writes
	// <node>_idea-brief.md. Reading past it drops the idea context from an adopted
	// system silently, which is the worst way to lose it.
	node, env, res, bank := legacyVault(t, "deze", realWorldIndex, PhaseRec, PhasePRD, PhaseTech, PhaseDDD)
	if err := os.WriteFile(filepath.Join(res.DocsRoot, "deze_idea.md"),
		[]byte("# idea\n\ncontenido heredado\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if r := Doctor(node, env, bank, DoctorOptions{Apply: true, Now: sddNow}); r.HasBlockers() {
		t.Fatalf("doctor blocked: %v", r.Blocked)
	}

	if r := CommitSDDContext(node, env, bank, validSDDInput(), sddNow); r.Result != CommitWritten {
		t.Fatalf("commit failed: %v", r.RejectedBecause)
	}
	manifest, _, err := LoadSDDManifest(res.SDDManifestPath())
	if err != nil {
		t.Fatalf("LoadSDDManifest: %v", err)
	}

	for _, source := range manifest.Sources {
		if source.Phase != PhaseIdea {
			continue
		}
		if !source.Present {
			t.Fatal("the idea context was skipped because only the current filename was tried")
		}
		if source.Artifact != "deze_idea.md" {
			t.Errorf("idea source = %q, want the legacy filename that exists", source.Artifact)
		}
		return
	}
	t.Fatal("idea is not among the sources at all")
}
