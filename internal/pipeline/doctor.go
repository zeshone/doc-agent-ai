package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DoctorSchema is the versioned name of the doctor report contract.
const DoctorSchema = "docagent.doctor/v1"

// Finding kinds.
const (
	// FindingAdoptPhase marks a phase with an artifact and no record.
	FindingAdoptPhase = "adopt-phase"
	// FindingAddIndexRegion marks an index that exists without a managed region.
	FindingAddIndexRegion = "add-index-region"
	// FindingCreateIndex marks a node with no index at all.
	FindingCreateIndex = "create-index"
	// FindingArchetype records the archetype that will be stored, and its source.
	FindingArchetype = "archetype"
	// FindingAlreadyManaged marks a phase the pipeline already counts.
	FindingAlreadyManaged = "already-managed"
)

// DoctorFinding is one thing doctor found, and would do or did.
type DoctorFinding struct {
	Kind   string  `json:"kind"`
	Node   string  `json:"node,omitempty"`
	Phase  PhaseID `json:"phase,omitempty"`
	Detail string  `json:"detail"`
}

// DoctorReport is the typed outcome of a doctor run.
type DoctorReport struct {
	SchemaName string `json:"schemaName"`
	Node       string `json:"node"`
	Mode       string `json:"mode,omitempty"`
	DocsRoot   string `json:"docsRoot,omitempty"`
	// Applied is false for a dry run. A dry run changes nothing.
	Applied  bool            `json:"applied"`
	Findings []DoctorFinding `json:"findings"`
	// Blocked lists what doctor refuses to decide on its own.
	Blocked []string `json:"blocked"`
}

// Blocked reports whether anything needs a human decision.
func (r DoctorReport) HasBlockers() bool { return len(r.Blocked) > 0 }

// DoctorOptions carries the caller's decisions.
type DoctorOptions struct {
	// Apply switches from reporting to writing. Default false on purpose: this
	// command touches real documentation.
	Apply bool
	// Archetype forces the value when doctor cannot determine it. Empty means
	// "determine it, and block if you cannot".
	Archetype string
	// Recursive also processes nodes nested under this one, which is the realistic
	// case: a system that predates records usually has modules that do too.
	Recursive bool
	// Now is the adoption timestamp, supplied by the caller so runs are reproducible.
	Now string
}

// Doctor aligns pre-existing documentation with the pipeline.
//
// The one thing it will never do is fabricate an answer record. The user's words
// from an interview months ago exist nowhere, and generating them from the
// artifacts would invent quotes attributed to them — the exact failure this
// package exists to prevent, made worse by carrying a tool's endorsement.
//
// So it cannot produce `complete`. It produces `adopted`: documented, inherited,
// coverage explicitly unverified.
func Doctor(node Node, env Environment, bank QuestionBank, opts DoctorOptions) DoctorReport {
	report := DoctorReport{
		SchemaName: DoctorSchema,
		Node:       node.Raw,
		Applied:    opts.Apply,
		Findings:   []DoctorFinding{},
		Blocked:    []string{},
	}

	res, err := Resolve(node, env)
	if err != nil {
		report.Blocked = append(report.Blocked, err.Error())
		return report
	}
	report.Mode = res.Mode
	report.DocsRoot = res.DocsRoot

	if !dirExists(res.DocsRoot) {
		report.Blocked = append(report.Blocked,
			fmt.Sprintf("%s does not exist, so there is no documentation to adopt", res.DocsRoot))
		return report
	}

	// Deepest first. A parent's module table reads each child's recorded node
	// status, so processing the parent before its children would stamp them all as
	// unmanaged and leave that stale the moment the children were adopted.
	targets := []Node{}
	if opts.Recursive {
		targets = append(targets, descendants(res, env)...)
	}
	sortByDepthDescending(targets)
	targets = append(targets, node)

	for _, target := range targets {
		inspectNode(target, env, bank, opts, &report)
	}
	return report
}

// inspectNode plans, and optionally applies, the adoption of one node.
func inspectNode(node Node, env Environment, bank QuestionBank, opts DoctorOptions, report *DoctorReport) {
	res, err := Resolve(node, env)
	if err != nil {
		report.Blocked = append(report.Blocked, fmt.Sprintf("%s: %v", node.Raw, err))
		return
	}

	existing, _, err := LoadAdoption(res.AdoptionPath(), bank)
	if err != nil {
		report.Blocked = append(report.Blocked, fmt.Sprintf("%s: %v", node.Raw, err))
		return
	}

	adoption := Adoption{
		SchemaName: AdoptionSchema,
		Node:       node.Raw,
		AdoptedAt:  opts.Now,
		Archetype:  existing.Archetype,
		Phases:     map[PhaseID]AdoptedPhase{},
	}
	for phase, adopted := range existing.Phases {
		adoption.Phases[phase] = adopted
	}

	// --- phases ---
	for _, phaseID := range CanonicalPhaseOrder() {
		spec, ok := bank.Phase(phaseID)
		if !ok {
			continue
		}

		artifact, ok := adoptableArtifact(spec, res)
		if !ok {
			continue
		}

		hasRecord := false
		if spec.Kind == KindAudit {
			_, found, _ := LoadAuditRecord(res.AuditRecordPath(phaseID), bank)
			hasRecord = found
		} else {
			_, found, _ := LoadAnswerRecord(res.AnswerRecordPath(phaseID), bank)
			hasRecord = found
		}

		if hasRecord {
			report.Findings = append(report.Findings, DoctorFinding{
				Kind: FindingAlreadyManaged, Node: node.Raw, Phase: phaseID,
				Detail: "coverage is already counted from a record; nothing to adopt",
			})
			continue
		}

		adoption.Phases[phaseID] = AdoptedPhase{
			Artifact: artifact,
			Evidence: "artifact present with no answer record; coverage unverified",
		}
		report.Findings = append(report.Findings, DoctorFinding{
			Kind: FindingAdoptPhase, Node: node.Raw, Phase: phaseID,
			Detail: fmt.Sprintf("%s exists with no answer record: adopt as unverified", artifact),
		})
	}

	// --- archetype (system nodes only; deeper nodes inherit it) ---
	if node.Type == NodeSystem {
		value, source := resolveArchetype(res, bank, opts, adoption)
		switch value {
		case "":
			report.Blocked = append(report.Blocked, fmt.Sprintf(
				"%s: the archetype could not be determined from %s. Re-run with --archetype %s or --archetype %s",
				node.Raw, res.IndexPath, ArchetypeBounded, ArchetypeEvolving))
		default:
			adoption.Archetype = value
			report.Findings = append(report.Findings, DoctorFinding{
				Kind: FindingArchetype, Node: node.Raw,
				Detail: fmt.Sprintf("archetype %q, %s", value, source),
			})
		}
	}

	// --- index ---
	indexRaw, indexErr := os.ReadFile(res.IndexPath)
	switch {
	case os.IsNotExist(indexErr):
		report.Findings = append(report.Findings, DoctorFinding{
			Kind: FindingCreateIndex, Node: node.Raw,
			Detail: fmt.Sprintf("%s does not exist and will be created", res.IndexPath),
		})
	case indexErr != nil:
		report.Blocked = append(report.Blocked, fmt.Sprintf("%s: %v", node.Raw, indexErr))
		return
	case !strings.Contains(string(indexRaw), indexRegionBegin):
		report.Findings = append(report.Findings, DoctorFinding{
			Kind: FindingAddIndexRegion, Node: node.Raw,
			Detail: fmt.Sprintf("%s has no managed region; one will be appended and existing prose kept",
				res.IndexPath),
		})
	}

	if !opts.Apply || len(report.Blocked) > 0 {
		return
	}

	if len(adoption.Phases) > 0 || adoption.Archetype != "" {
		if err := writeJSONAtomic(res.AdoptionPath(), adoption); err != nil {
			report.Blocked = append(report.Blocked, fmt.Sprintf("%s: %v", node.Raw, err))
			return
		}
	}

	status := ComputeStatus(node, env, bank)
	if _, err := writeIndex(res, env, bank, status); err != nil {
		report.Blocked = append(report.Blocked, fmt.Sprintf("%s: index not written: %v", node.Raw, err))
	}
}

// adoptableArtifact finds the file that makes a phase adoptable.
//
// An audit phase is deliberately excluded from its own `artifact`, because it
// shares that file with an earlier phase and its presence proves nothing about
// whether the audit ran. A dedicated legacy report is different: it exists only
// because someone produced it.
func adoptableArtifact(spec PhaseSpec, res Resolution) (string, bool) {
	if spec.Kind != KindAudit {
		if name := spec.ArtifactName(res.ArtifactPrefix); fileExists(filepath.Join(res.DocsRoot, name)) {
			return name, true
		}
	}
	for _, name := range spec.LegacyArtifactNames(res.ArtifactPrefix) {
		if name == spec.ArtifactName(res.ArtifactPrefix) {
			continue
		}
		if fileExists(filepath.Join(res.DocsRoot, name)) {
			return name, true
		}
	}
	return "", false
}

// resolveArchetype determines the archetype and says where the value came from.
//
// Precedence puts recorded facts above inference: an explicit flag, then a value
// already adopted, then the answer a pipeline interview recorded, and only then a
// read of the index prose. Inference is last and never silent — the source is
// reported so a human can disagree.
func resolveArchetype(res Resolution, bank QuestionBank, opts DoctorOptions, adoption Adoption) (value, source string) {
	if opts.Archetype != "" {
		if !validArchetype(opts.Archetype) {
			return "", ""
		}
		return opts.Archetype, "supplied with --archetype"
	}
	if adoption.Archetype != "" {
		return adoption.Archetype, "already recorded by a previous adoption"
	}
	if recorded := recordedArchetype(res, bank); recorded != "" {
		return recorded, "recorded by the rec interview"
	}

	raw, err := os.ReadFile(res.IndexPath)
	if err != nil {
		return "", ""
	}
	lower := strings.ToLower(string(raw))

	// Heuristic, and reported as one. Real vaults phrase this fact at least three
	// different ways, so matching a single literal was never going to hold.
	evolving := containsAny(lower, "evolutiv", "evolving", "crece con m")
	bounded := containsAny(lower, "bounded", "acotado", "delimitado", "single delivery", "entrega unica", "entrega única")

	switch {
	case evolving && !bounded:
		return ArchetypeEvolving, "inferred from the index prose"
	case bounded && !evolving:
		return ArchetypeBounded, "inferred from the index prose"
	}
	return "", ""
}

func containsAny(haystack string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(haystack, needle) {
			return true
		}
	}
	return false
}

// sortByDepthDescending puts deeper nodes first, so a parent is always processed
// after every child whose status its own index will report.
func sortByDepthDescending(nodes []Node) {
	for i := 1; i < len(nodes); i++ {
		for j := i; j > 0 && len(nodes[j].Segments) > len(nodes[j-1].Segments); j-- {
			nodes[j], nodes[j-1] = nodes[j-1], nodes[j]
		}
	}
}

// descendants returns the nodes nested under this one, depth first.
func descendants(res Resolution, env Environment) []Node {
	var out []Node
	for _, child := range discoverChildNodes(res) {
		childNode, err := ParseNode(res.Node.Raw + "/" + child.name)
		if err != nil {
			continue
		}
		out = append(out, childNode)

		childRes, err := Resolve(childNode, env)
		if err != nil {
			continue
		}
		out = append(out, descendants(childRes, env)...)
	}
	return out
}
