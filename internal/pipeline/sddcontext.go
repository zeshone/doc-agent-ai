package pipeline

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Versioned names of the SDD-context contracts.
const (
	// SDDInputSchema is what a caller submits: the two compacted documents and
	// the TBDs it claims to have carried across.
	SDDInputSchema = "docagent.sddinput/v1"
	// SDDManifestSchema is what the program records: which artifacts fed the
	// compaction, and their state at the time.
	SDDManifestSchema = "docagent.sddmanifest/v1"
)

// Layers of the compacted context.
const (
	LayerBusiness  = "business"
	LayerTechnical = "technical"
)

// Freshness of a recorded compaction.
const (
	// SDDFresh means every source still hashes to what it did at compaction.
	SDDFresh = "fresh"
	// SDDStale means at least one source changed underneath it.
	SDDStale = "stale"
	// SDDAbsent means no compaction has been recorded.
	SDDAbsent = "absent"
)

// sddSourcePhases names the artifacts each layer compacts, in reading order.
//
// pti is deliberately absent: an agent working from this context is normally
// working on one issue already, and folding the whole issue list back in fights
// the point of compacting seven documents into two.
var sddSourcePhases = []struct {
	Phase PhaseID
	Layer string
	// Report selects the phase's rendered report rather than its own artifact,
	// which is how refine contributes: its artifact is the PRD it audits.
	Report bool
}{
	{Phase: PhaseIdea, Layer: LayerBusiness},
	{Phase: PhaseRec, Layer: LayerBusiness},
	{Phase: PhasePRD, Layer: LayerBusiness},
	{Phase: PhaseRefine, Layer: LayerBusiness, Report: true},
	{Phase: PhaseTech, Layer: LayerTechnical},
	{Phase: PhaseDDD, Layer: LayerTechnical},
}

// SDDDecision is one recorded decision in the bounded shape an agent can read
// without opening the documentation it came from.
//
// All four fields are required. A decision missing its reason is a fact with no
// justification, and one missing what it buys is a change with no purpose —
// either way the reader has to go back to the full documents, which is the cost
// this whole command exists to remove.
type SDDDecision struct {
	// ID is a stable handle so other sections can refer to a decision.
	ID string `json:"id"`
	// Layer places the decision in the business or the technical context.
	Layer string `json:"layer"`
	// What was decided.
	What string `json:"what"`
	// Why it was decided — the reason or the constraint behind it.
	Why string `json:"why"`
	// SoThat is the outcome the decision buys.
	SoThat string `json:"soThat"`
	// DecidedBy is how it was settled: who or what established it, and where that
	// is recorded. Without it a decision is an assertion with no provenance.
	DecidedBy string `json:"decidedBy"`
}

// SDDInput is the compaction a caller submits for verification.
//
// The prose is the model's: no program can compact seven documents into two.
// What the program can do is refuse to write a compaction whose sources it
// cannot name, and refuse one that quietly resolved an open question.
type SDDInput struct {
	SchemaName string `json:"schemaName"`
	Business   string `json:"business"`
	Technical  string `json:"technical"`
	// PreservedTBDs are the open questions the caller claims to have carried
	// across verbatim. Each is checked against the actual bytes on both sides: a
	// claim absent from every source was invented, and one absent from the output
	// was not preserved.
	PreservedTBDs []string `json:"preservedTbds"`
	// Decisions carry the settled choices in a bounded shape. The program renders
	// them, so an agent reading the context finds the same structure every time
	// instead of whatever prose the compaction happened to produce.
	Decisions []SDDDecision `json:"decisions"`
}

// SDDSource is one artifact that fed a compaction.
type SDDSource struct {
	Phase    PhaseID `json:"phase"`
	Artifact string  `json:"artifact"`
	Layer    string  `json:"layer"`
	Present  bool    `json:"present"`
	// Revision fingerprints the bytes read. Empty when the artifact was absent.
	Revision string `json:"revision,omitempty"`
	// Adopted marks a source whose phase carries inherited documentation, so the
	// compaction inherits unverified coverage and says so rather than laundering
	// it into something that looks checked.
	Adopted bool `json:"adopted,omitempty"`
}

// SDDManifest records what a compaction was derived from.
type SDDManifest struct {
	SchemaName  string      `json:"schemaName"`
	Node        string      `json:"node"`
	GeneratedAt string      `json:"generatedAt"`
	Sources     []SDDSource `json:"sources"`
	Outputs     []string    `json:"outputs"`
	// PreservedTBDs is what the compaction carried across, after verification.
	PreservedTBDs []string `json:"preservedTbds,omitempty"`
	// CoverageVerified is false when any source came from adopted documentation.
	CoverageVerified bool `json:"coverageVerified"`
}

// Validate enforces the input contract's shape.
func (in SDDInput) Validate() error {
	if in.SchemaName != SDDInputSchema {
		return fmt.Errorf("sdd input declares schema %q, expected %q", in.SchemaName, SDDInputSchema)
	}
	if strings.TrimSpace(in.Business) == "" {
		return fmt.Errorf("the business layer is empty")
	}
	if strings.TrimSpace(in.Technical) == "" {
		return fmt.Errorf("the technical layer is empty")
	}
	for i, tbd := range in.PreservedTBDs {
		if strings.TrimSpace(tbd) == "" {
			return fmt.Errorf("preserved TBD %d is empty", i+1)
		}
	}

	seen := map[string]bool{}
	for i, decision := range in.Decisions {
		position := fmt.Sprintf("decision %d", i+1)
		if id := strings.TrimSpace(decision.ID); id == "" {
			return fmt.Errorf("%s has no id", position)
		} else if seen[id] {
			return fmt.Errorf("decision id %q appears more than once", id)
		} else {
			seen[id] = true
			position = fmt.Sprintf("decision %q", id)
		}
		if decision.Layer != LayerBusiness && decision.Layer != LayerTechnical {
			return fmt.Errorf("%s declares layer %q, expected %q or %q",
				position, decision.Layer, LayerBusiness, LayerTechnical)
		}
		for field, value := range map[string]string{
			"what": decision.What, "why": decision.Why,
			"soThat": decision.SoThat, "decidedBy": decision.DecidedBy,
		} {
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("%s has an empty %s: a decision without it sends the reader "+
					"back to the full documents, which is the cost this command exists to remove",
					position, field)
			}
		}
	}
	return nil
}

// renderDecisions draws the bounded decision block for one layer. The program
// owns this structure so every compacted context reads the same way.
func renderDecisions(decisions []SDDDecision, layer string) string {
	var picked []SDDDecision
	for _, decision := range decisions {
		if decision.Layer == layer {
			picked = append(picked, decision)
		}
	}
	if len(picked) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n## Decisions\n")
	for _, decision := range picked {
		fmt.Fprintf(&b, "\n### %s — %s\n", decision.ID, strings.TrimSpace(decision.What))
		fmt.Fprintf(&b, "\n- **Why:** %s\n", strings.TrimSpace(decision.Why))
		fmt.Fprintf(&b, "- **So that:** %s\n", strings.TrimSpace(decision.SoThat))
		fmt.Fprintf(&b, "- **Decided by:** %s\n", strings.TrimSpace(decision.DecidedBy))
	}
	return b.String()
}

// Validate enforces the manifest contract.
func (m SDDManifest) Validate() error {
	if m.SchemaName != SDDManifestSchema {
		return fmt.Errorf("sdd manifest declares schema %q, expected %q", m.SchemaName, SDDManifestSchema)
	}
	if strings.TrimSpace(m.Node) == "" {
		return fmt.Errorf("sdd manifest has an empty node")
	}
	for _, source := range m.Sources {
		if source.Present && source.Revision == "" {
			return fmt.Errorf("source %q is recorded present with no revision", source.Artifact)
		}
		if !source.Present && source.Revision != "" {
			return fmt.Errorf("source %q is recorded absent with a revision", source.Artifact)
		}
	}
	return nil
}

// SDDManifestPath is where the manifest lives, beside the compacted outputs.
func (r Resolution) SDDManifestPath() string {
	return filepath.Join(r.SDDContextDir(), "manifest.json")
}

// SDDContextDir is the directory holding the compacted context.
func (r Resolution) SDDContextDir() string {
	return filepath.Join(r.DocsRoot, "agent_sdd_context_project")
}

// SDDOutputName resolves a layer's output filename.
//
// Naming is mode-aware, matching what the skill already ships: vault mode
// prefixes with the node short name, in-project mode keeps the bare form.
func (r Resolution) SDDOutputName(layer string) string {
	suffix := "_sdd-context.md"
	if layer == LayerTechnical {
		suffix = "_sdd-tech-context.md"
	}
	return r.ArtifactPrefix + suffix
}

// fileRevision fingerprints a file, returning "" when it does not exist.
func fileRevision(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// currentSDDSources reads the artifacts that feed a compaction right now.
func currentSDDSources(res Resolution, bank QuestionBank, adopted map[PhaseID]bool) []SDDSource {
	var out []SDDSource
	for _, entry := range sddSourcePhases {
		spec, ok := bank.Phase(entry.Phase)
		if !ok {
			continue
		}
		name := spec.ArtifactName(res.ArtifactPrefix)
		if entry.Report {
			name = spec.ReportArtifactName(res.ArtifactPrefix)
			if name == "" {
				continue
			}
		}
		revision := fileRevision(filepath.Join(res.DocsRoot, name))

		// Fall back to a legacy filename when the current one is absent. Inherited
		// vaults hold <node>_idea.md while the current convention writes
		// <node>_idea-brief.md, and reading past it would drop the idea context
		// from an adopted system without saying anything.
		if revision == "" && !entry.Report {
			for _, legacy := range spec.LegacyArtifactNames(res.ArtifactPrefix) {
				if legacy == name {
					continue
				}
				if r := fileRevision(filepath.Join(res.DocsRoot, legacy)); r != "" {
					name, revision = legacy, r
					break
				}
			}
		}
		out = append(out, SDDSource{
			Phase:    entry.Phase,
			Artifact: name,
			Layer:    entry.Layer,
			Present:  revision != "",
			Revision: revision,
			Adopted:  adopted[entry.Phase],
		})
	}
	return out
}

// LoadSDDManifest reads the recorded manifest. Absence means nothing has been
// compacted yet, which is a normal state rather than an error.
func LoadSDDManifest(path string) (SDDManifest, bool, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return SDDManifest{}, false, nil
	}
	if err != nil {
		return SDDManifest{}, false, fmt.Errorf("reading sdd manifest %s: %w", path, err)
	}
	var manifest SDDManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return SDDManifest{}, true, fmt.Errorf("parsing sdd manifest %s: %w", path, err)
	}
	if err := manifest.Validate(); err != nil {
		return SDDManifest{}, true, fmt.Errorf("sdd manifest %s is invalid: %w", path, err)
	}
	return manifest, true, nil
}

// SDDCommitResult is the typed outcome of submitting a compaction.
type SDDCommitResult struct {
	SchemaName string   `json:"schemaName"`
	Result     string   `json:"result"`
	Node       string   `json:"node"`
	Written    []string `json:"written"`
	// Checks records what was verified, in the same shape validation uses
	// elsewhere, so a refusal names exactly what to fix.
	Checks          []Check  `json:"checks"`
	RejectedBecause []string `json:"rejectedBecause,omitempty"`
	Detail          string   `json:"detail,omitempty"`
}

// SDD check ids.
const (
	CheckSDDSourcesPresent  = "sdd-sources-present"
	CheckSDDTBDsPreserved   = "sdd-tbds-preserved"
	CheckSDDTBDsNotInvented = "sdd-tbds-not-invented"
	CheckSDDCoverageStated  = "sdd-coverage-stated"
	CheckSDDDecisionsBound  = "sdd-decisions-bounded"
)

// CommitSDDContext verifies a compaction and writes it only if it holds.
//
// It cannot judge whether the compaction is faithful to its sources — that is a
// judgement about meaning, and a check claiming to make it would be theatre. It
// verifies what is verifiable: that the sources are named and fingerprinted,
// that every open question the caller claims to have carried is really present
// on both sides, and that a compaction of unverified documentation says so.
func CommitSDDContext(node Node, env Environment, bank QuestionBank, input SDDInput, now string) SDDCommitResult {
	result := SDDCommitResult{
		SchemaName: SDDManifestSchema,
		Node:       node.Raw,
		Written:    []string{},
	}

	if err := input.Validate(); err != nil {
		result.Result = CommitRejected
		result.Checks = []Check{{ID: CheckSDDSourcesPresent, Result: CheckFail, Detail: err.Error()}}
		result.RejectedBecause = []string{CheckSDDSourcesPresent}
		return result
	}

	res, err := Resolve(node, env)
	if err != nil {
		result.Result = CommitUndetermined
		result.Detail = err.Error()
		return result
	}

	adoption, _, err := LoadAdoption(res.AdoptionPath(), bank)
	if err != nil {
		result.Result = CommitUndetermined
		result.Detail = err.Error()
		return result
	}
	adopted := map[PhaseID]bool{}
	for phase := range adoption.Phases {
		adopted[phase] = true
	}

	sources := currentSDDSources(res, bank, adopted)

	var present int
	var inherited []PhaseID
	for _, source := range sources {
		if source.Present {
			present++
			if source.Adopted {
				inherited = append(inherited, source.Phase)
			}
		}
	}

	if present == 0 {
		result.Result = CommitRejected
		result.Checks = []Check{{
			ID: CheckSDDSourcesPresent, Result: CheckFail,
			Detail: "none of the source artifacts exist, so there is nothing to compact",
		}}
		result.RejectedBecause = []string{CheckSDDSourcesPresent}
		return result
	}
	result.Checks = append(result.Checks, Check{
		ID: CheckSDDSourcesPresent, Result: CheckPass,
		Detail: fmt.Sprintf("%d of %d source artifacts present", present, len(sources)),
	})

	// Read the source bytes once, for the TBD verification below.
	var corpus strings.Builder
	for _, source := range sources {
		if !source.Present {
			continue
		}
		if raw, err := os.ReadFile(filepath.Join(res.DocsRoot, source.Artifact)); err == nil {
			corpus.Write(raw)
			corpus.WriteString("\n")
		}
	}
	sourceText := corpus.String()
	outputText := input.Business + "\n" + input.Technical

	var invented, dropped []string
	for _, tbd := range input.PreservedTBDs {
		claim := strings.TrimSpace(tbd)
		if !strings.Contains(sourceText, claim) {
			invented = append(invented, claim)
			continue
		}
		if !strings.Contains(outputText, claim) {
			dropped = append(dropped, claim)
		}
	}

	if len(invented) > 0 {
		result.Checks = append(result.Checks, Check{
			ID: CheckSDDTBDsNotInvented, Result: CheckFail,
			Detail: fmt.Sprintf(
				"%d claimed open question(s) appear in no source artifact: they were invented, not preserved",
				len(invented)),
			TopicIDs: invented,
		})
	} else {
		result.Checks = append(result.Checks, Check{ID: CheckSDDTBDsNotInvented, Result: CheckPass})
	}

	if len(dropped) > 0 {
		result.Checks = append(result.Checks, Check{
			ID: CheckSDDTBDsPreserved, Result: CheckFail,
			Detail: fmt.Sprintf("%d open question(s) claimed as preserved are absent from the compaction",
				len(dropped)),
			TopicIDs: dropped,
		})
	} else {
		detail := fmt.Sprintf("%d open question(s) verified present in both a source and the compaction",
			len(input.PreservedTBDs))
		if len(input.PreservedTBDs) == 0 && strings.Contains(sourceText, "TBD") {
			// Not a refusal: "TBD" appears inside prose that is not an open question.
			// It is reported because the alternative reading is that every open
			// question was quietly resolved.
			detail = "no open questions were declared, yet the sources mention TBD; confirm none were resolved silently"
		}
		result.Checks = append(result.Checks, Check{ID: CheckSDDTBDsPreserved, Result: CheckPass, Detail: detail})
	}

	// A context with no decisions is a summary, and a summary sends the reader
	// back to the full documents — the exact cost this command exists to remove.
	if len(input.Decisions) == 0 {
		result.Checks = append(result.Checks, Check{
			ID: CheckSDDDecisionsBound, Result: CheckFail,
			Detail: "no decisions were recorded; an agent reading this would still have to open the source documents to learn what was settled and why",
		})
	} else {
		result.Checks = append(result.Checks, Check{
			ID: CheckSDDDecisionsBound, Result: CheckPass,
			Detail: fmt.Sprintf("%d decision(s), each with what, why, so-that and how it was decided",
				len(input.Decisions)),
		})
	}

	coverageVerified := len(inherited) == 0
	statement := "every source carries counted coverage"
	if !coverageVerified {
		statement = fmt.Sprintf("%d source phase(s) are adopted: this compaction inherits unverified coverage",
			len(inherited))
	}
	result.Checks = append(result.Checks, Check{
		ID: CheckSDDCoverageStated, Result: CheckPass, Detail: statement,
	})

	for _, check := range result.Checks {
		if check.Result != CheckPass {
			result.RejectedBecause = append(result.RejectedBecause, check.ID)
		}
	}
	if len(result.RejectedBecause) > 0 {
		result.Result = CommitRejected
		return result
	}

	outputs := map[string]string{
		LayerBusiness:  strings.TrimRight(input.Business, "\n") + "\n" + renderDecisions(input.Decisions, LayerBusiness),
		LayerTechnical: strings.TrimRight(input.Technical, "\n") + "\n" + renderDecisions(input.Decisions, LayerTechnical),
	}
	var written []string
	for _, layer := range []string{LayerBusiness, LayerTechnical} {
		path := filepath.Join(res.SDDContextDir(), res.SDDOutputName(layer))
		if err := writeFileAtomic(path, []byte(outputs[layer])); err != nil {
			result.Result = CommitUndetermined
			result.Detail = err.Error()
			return result
		}
		written = append(written, path)
	}

	manifest := SDDManifest{
		SchemaName:       SDDManifestSchema,
		Node:             node.Raw,
		GeneratedAt:      now,
		Sources:          sources,
		Outputs:          written,
		PreservedTBDs:    input.PreservedTBDs,
		CoverageVerified: coverageVerified,
	}
	if err := writeJSONAtomic(res.SDDManifestPath(), manifest); err != nil {
		result.Result = CommitUndetermined
		result.Detail = err.Error()
		return result
	}

	// Refresh the index so the compacted context appears there without anyone
	// writing that section by hand.
	status := ComputeStatus(node, env, bank)
	if _, err := writeIndex(res, env, bank, status); err != nil {
		result.Result = CommitWritten
		result.Written = append(written, res.SDDManifestPath())
		result.Detail = fmt.Sprintf("context written, but the index was not refreshed: %v", err)
		return result
	}

	result.Result = CommitWritten
	result.Written = append(written, res.SDDManifestPath())
	return result
}

// SDDStatus is the freshness of a node's compacted context.
type SDDStatus struct {
	State string `json:"state"`
	// Drifted names sources that no longer hash to what they did at compaction.
	Drifted []string `json:"drifted,omitempty"`
	// Appeared names sources that did not exist at compaction and do now.
	Appeared []string `json:"appeared,omitempty"`
	// CoverageVerified is false when the compaction drew on adopted documentation.
	CoverageVerified bool     `json:"coverageVerified"`
	Outputs          []string `json:"outputs,omitempty"`
}

// computeSDDStatus compares the recorded manifest against the sources today.
func computeSDDStatus(res Resolution, bank QuestionBank, adopted map[PhaseID]bool) *SDDStatus {
	manifest, found, err := LoadSDDManifest(res.SDDManifestPath())
	if err != nil {
		return &SDDStatus{State: SDDStale, Drifted: []string{err.Error()}}
	}
	if !found {
		return &SDDStatus{State: SDDAbsent}
	}

	recorded := map[string]SDDSource{}
	for _, source := range manifest.Sources {
		recorded[source.Artifact] = source
	}

	status := &SDDStatus{
		State:            SDDFresh,
		CoverageVerified: manifest.CoverageVerified,
		Outputs:          manifest.Outputs,
	}
	for _, current := range currentSDDSources(res, bank, adopted) {
		was, known := recorded[current.Artifact]
		switch {
		case !known && current.Present:
			status.Appeared = append(status.Appeared, current.Artifact)
		case known && was.Revision != current.Revision:
			status.Drifted = append(status.Drifted, current.Artifact)
		}
	}
	if len(status.Drifted) > 0 || len(status.Appeared) > 0 {
		status.State = SDDStale
	}
	return status
}
