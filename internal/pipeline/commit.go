package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CommitSchema is the versioned name of the commit contract.
const CommitSchema = "docagent.commit/v1"

// Commit outcomes.
const (
	CommitWritten      = "written"
	CommitRejected     = "rejected"
	CommitUndetermined = "undetermined"
)

// Index region markers. Everything between them is machine-owned and recomputed
// on every commit; everything outside is the author's and is preserved verbatim.
const (
	indexRegionBegin = "<!-- doc-agent:state:begin -->"
	indexRegionEnd   = "<!-- doc-agent:state:end -->"
)

// Node statuses from skills/doc-arch/SKILL.md:71-78. "in review" is absent: it
// depends on whether issues were published to GitHub, which this package does
// not observe, and inventing it would be a claim rather than a computed fact.
const (
	NodeStatusStarted    = "started"
	NodeStatusInProgress = "in progress"
	NodeStatusDocumented = "documented"
	// NodeStatusAdopted is a node whose phases are all inherited. It is not
	// "documented", because that word would claim coverage nobody counted.
	NodeStatusAdopted = "adopted (coverage unverified)"
)

// IndexUpdate reports what happened to the node index.
type IndexUpdate struct {
	File                 string  `json:"file"`
	PhaseMarked          PhaseID `json:"phaseMarked"`
	NodeStatusRecomputed string  `json:"nodeStatusRecomputed"`
}

// CommitResult is the typed outcome of a commit attempt.
type CommitResult struct {
	SchemaName string  `json:"schemaName"`
	Result     string  `json:"result"`
	Node       string  `json:"node"`
	Phase      PhaseID `json:"phase"`
	// Written lists the files that landed. Empty on any non-written result.
	Written      []string     `json:"written"`
	IndexUpdated *IndexUpdate `json:"indexUpdated,omitempty"`
	// ParentIndex reports the ancestor indexes refreshed so they list this node.
	// Absent when the node has no parent, or when an ancestor carries no managed
	// region — see propagateToAncestors.
	ParentIndex     []IndexUpdate `json:"parentIndex,omitempty"`
	NextRecommended PhaseID       `json:"nextRecommended,omitempty"`
	// Validation carries the reason whenever the submission was refused.
	Validation *ValidationResult `json:"validation,omitempty"`
	Detail     string            `json:"detail,omitempty"`
}

// Commit validates a submission and, only if it passes, writes it.
//
// This function is why "if the validator rejects, write nothing" is a fact
// rather than a request: the program holds the pen. A rejected submission
// returns before any file is touched.
//
// Ordering note: records are renamed into place before the artifact, and the
// index last. A crash between renames can leave the index stale, which is
// cosmetic and self-healing — ComputeStatus never reads the index, so the next
// commit recomputes it from the records.
func Commit(sub Submission, env Environment, bank QuestionBank) CommitResult {
	result := CommitResult{
		SchemaName: CommitSchema,
		Node:       sub.Node.Raw,
		Phase:      sub.Phase,
		Written:    []string{},
	}

	spec, ok := bank.Phase(sub.Phase)
	if !ok {
		result.Result = CommitUndetermined
		result.Detail = fmt.Sprintf("phase %q is not declared in the question bank", sub.Phase)
		return result
	}

	res, err := Resolve(sub.Node, env)
	if err != nil {
		result.Result = CommitUndetermined
		result.Detail = err.Error()
		return result
	}

	validation := Validate(sub, bank)
	if !validation.Accepted() {
		result.Result = CommitRejected
		result.Validation = &validation
		return result
	}

	if err := os.MkdirAll(res.DocsRoot, 0o755); err != nil {
		result.Result = CommitUndetermined
		result.Detail = fmt.Sprintf("creating docs root %s: %v", res.DocsRoot, err)
		return result
	}

	// Records first: they are the evidence the artifact's coverage rests on.
	if sub.Answers != nil {
		path := res.AnswerRecordPath(sub.Phase)
		if err := writeJSONAtomic(path, sub.Answers); err != nil {
			result.Result = CommitUndetermined
			result.Detail = err.Error()
			return result
		}
		result.Written = append(result.Written, path)
	}
	if sub.Audit != nil {
		path := res.AuditRecordPath(sub.Phase)
		if err := writeJSONAtomic(path, sub.Audit); err != nil {
			result.Result = CommitUndetermined
			result.Detail = err.Error()
			return result
		}
		result.Written = append(result.Written, path)
	}

	// Audit phases own no artifact of their own.
	if spec.Kind != KindAudit {
		// Store the authored prose before rendering. The artifact is derived; the
		// input is not recoverable from it, and without the input no phase could be
		// partially corrected without re-interviewing the user.
		sectionPath := res.SectionInputPath(sub.Phase)
		stored := sub.Content
		stored.SchemaName = SectionsSchema
		if err := writeJSONAtomic(sectionPath, stored); err != nil {
			result.Result = CommitUndetermined
			result.Detail = err.Error()
			return result
		}
		result.Written = append(result.Written, sectionPath)
	}

	if spec.Kind != KindAudit {
		// The program renders the document: canonical English headings from the
		// bank, authored prose beneath them. The model never hands over finished
		// markdown, so structure cannot disagree with the coverage record.
		var record AnswerRecord
		if sub.Answers != nil {
			record = *sub.Answers
		}
		rendered, err := Render(spec, sub.Node, bank, sub.Content, record)
		if err != nil {
			result.Result = CommitUndetermined
			result.Detail = err.Error()
			return result
		}
		path := res.ArtifactPath(sub.Phase, bank)
		if err := writeFileAtomic(path, rendered); err != nil {
			result.Result = CommitUndetermined
			result.Detail = err.Error()
			return result
		}
		result.Written = append(result.Written, path)
	}

	// The index is rendered from freshly computed status, never from a claim.
	status := ComputeStatus(sub.Node, env, bank)
	nodeStatus, err := writeIndex(res, env, bank, status)
	if err != nil {
		// The durable evidence already landed, so this is not a rejection: report
		// the index failure honestly and let the next commit reconcile it.
		result.Result = CommitWritten
		result.Detail = fmt.Sprintf("artifact and records were written, but the index was not updated: %v", err)
		result.NextRecommended = status.NextRecommended
		return result
	}

	result.Result = CommitWritten
	result.IndexUpdated = &IndexUpdate{
		File:                 res.IndexPath,
		PhaseMarked:          sub.Phase,
		NodeStatusRecomputed: nodeStatus,
	}
	result.NextRecommended = status.NextRecommended

	// A documented module that never appears in its parent's index is invisible
	// to anyone reading the system from the top.
	parents, skipped := propagateToAncestors(sub.Node, env, bank)
	result.ParentIndex = parents
	if skipped != "" {
		result.Detail = skipped
	}
	return result
}

// propagateToAncestors refreshes each ancestor index so it lists this node.
//
// It only touches an ancestor that ALREADY carries a managed region. Creating
// one here would compute it from answer records the ancestor may not have —
// documentation written before records existed would suddenly gain a region
// claiming nothing is done, contradicting its own prose in the same file. An
// unmanaged ancestor is reported instead, and adopting it is a separate,
// deliberate step.
func propagateToAncestors(node Node, env Environment, bank QuestionBank) (updated []IndexUpdate, skipped string) {
	current := node
	for {
		parent, ok := current.Parent()
		if !ok {
			return updated, skipped
		}

		parentRes, err := Resolve(parent, env)
		if err != nil {
			return updated, fmt.Sprintf("ancestor %s could not be resolved: %v", parent.Raw, err)
		}

		existing, err := os.ReadFile(parentRes.IndexPath)
		if err != nil || !strings.Contains(string(existing), indexRegionBegin) {
			if skipped == "" {
				skipped = fmt.Sprintf(
					"%s carries no doc-agent managed region, so it was left untouched and does not list this node yet",
					parentRes.IndexPath)
			}
			return updated, skipped
		}

		parentStatus := ComputeStatus(parent, env, bank)
		nodeStatus, err := writeIndex(parentRes, env, bank, parentStatus)
		if err != nil {
			return updated, fmt.Sprintf("ancestor index %s was not refreshed: %v", parentRes.IndexPath, err)
		}
		updated = append(updated, IndexUpdate{
			File:                 parentRes.IndexPath,
			NodeStatusRecomputed: nodeStatus,
		})

		current = parent
	}
}

// LoadSectionInput reads back the stored prose for a phase. A missing file is a
// normal state — phases committed before section storage existed have none.
func LoadSectionInput(path string) (SectionInput, bool, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return SectionInput{}, false, nil
	}
	if err != nil {
		return SectionInput{}, false, fmt.Errorf("reading section input %s: %w", path, err)
	}
	var input SectionInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return SectionInput{}, true, fmt.Errorf("parsing section input %s: %w", path, err)
	}
	if input.SchemaName != SectionsSchema {
		return SectionInput{}, true, fmt.Errorf(
			"section input %s declares schema %q, expected %q", path, input.SchemaName, SectionsSchema)
	}
	return input, true, nil
}

// RecordDecision persists a decision about an optional phase so it survives
// across sessions instead of being re-asked every time.
func RecordDecision(node Node, env Environment, bank QuestionBank,
	phase PhaseID, decision OptionalPhaseDecision) error {

	spec, ok := bank.Phase(phase)
	if !ok {
		return fmt.Errorf("phase %q is not declared in the question bank", phase)
	}
	if !spec.Optional {
		return fmt.Errorf("phase %q is not optional, so it carries no decision", phase)
	}
	if decision != DecisionAccepted && decision != DecisionDeclined {
		return fmt.Errorf("decision %q is not %q or %q", decision, DecisionAccepted, DecisionDeclined)
	}

	res, err := Resolve(node, env)
	if err != nil {
		return err
	}

	decisions, _, err := LoadDecisions(res.DecisionsPath(), bank)
	if err != nil {
		return err
	}
	if decisions.SchemaName == "" {
		decisions = Decisions{SchemaName: DecisionsSchema, Node: node.Raw}
	}
	if decisions.Optional == nil {
		decisions.Optional = map[PhaseID]OptionalPhaseDecision{}
	}
	decisions.Optional[phase] = decision

	return writeJSONAtomic(res.DecisionsPath(), decisions)
}

// writeIndex recomputes the machine-owned region of the node index and returns
// the recomputed node status.
func writeIndex(res Resolution, env Environment, bank QuestionBank, status Status) (string, error) {
	nodeStatus := deriveNodeStatus(status)
	region := renderIndexRegion(res, env, bank, status, nodeStatus)

	existing, err := os.ReadFile(res.IndexPath)
	switch {
	case os.IsNotExist(err):
		// A fresh index gets a minimal skeleton. The description line is left for
		// the author to fill: this package owns state, not prose.
		content := fmt.Sprintf("# %s\n\nTBD\n\n%s\n", res.Node.ShortName, region)
		return nodeStatus, writeFileAtomic(res.IndexPath, []byte(content))
	case err != nil:
		return "", fmt.Errorf("reading index %s: %w", res.IndexPath, err)
	}

	updated, err := replaceRegion(string(existing), region)
	if err != nil {
		return "", err
	}
	return nodeStatus, writeFileAtomic(res.IndexPath, []byte(updated))
}

// replaceRegion swaps the machine-owned region, preserving everything else. An
// index with no region gets one appended rather than being rewritten.
func replaceRegion(existing, region string) (string, error) {
	begin := strings.Index(existing, indexRegionBegin)
	end := strings.Index(existing, indexRegionEnd)

	switch {
	case begin < 0 && end < 0:
		separator := "\n\n"
		if strings.HasSuffix(existing, "\n\n") {
			separator = ""
		} else if strings.HasSuffix(existing, "\n") {
			separator = "\n"
		}
		return existing + separator + region + "\n", nil
	case begin < 0 || end < 0:
		// One marker without the other means an edit truncated the region. Refuse
		// rather than guess where it should end and risk destroying prose.
		return "", fmt.Errorf(
			"index has an unbalanced doc-agent state region: repair or remove the %s / %s markers",
			indexRegionBegin, indexRegionEnd)
	case end < begin:
		return "", fmt.Errorf("index has the doc-agent state markers in reverse order")
	}

	return existing[:begin] + region + existing[end+len(indexRegionEnd):], nil
}

// renderIndexRegion draws the state table. It is language-independent on
// purpose: phase ids and states are machine vocabulary, so this region reads the
// same whatever language the surrounding documentation is written in.
func renderIndexRegion(res Resolution, env Environment, bank QuestionBank, status Status, nodeStatus string) string {
	var b strings.Builder
	b.WriteString(indexRegionBegin)
	b.WriteString("\n")
	b.WriteString("<!-- Generated by doc-agent-ai from the recorded answers.\n")
	b.WriteString("     Do not edit inside this region: it is recomputed on every commit. -->\n\n")
	b.WriteString("| Phase | Done | State | Coverage |\n")
	b.WriteString("|---|---|---|---|\n")

	for _, ps := range status.Phases {
		// [x] asserts counted coverage. Inherited documentation has none, so it gets
		// its own mark rather than borrowing the one that means verified.
		done := "[ ]"
		switch ps.State {
		case StateComplete:
			done = "[x]"
		case StateAdopted:
			done = "[~]"
		}
		coverage := "—"
		if ps.RequiredTopics > 0 {
			accounted := ps.AnsweredTopics + len(ps.DeferredTopics)
			coverage = fmt.Sprintf("%d/%d", accounted, ps.RequiredTopics)
			if len(ps.DeferredTopics) > 0 {
				coverage += fmt.Sprintf(" (%d deferred)", len(ps.DeferredTopics))
			}
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", ps.ID, done, ps.State, coverage))
	}

	if children := discoverChildNodes(res); len(children) > 0 {
		b.WriteString("\n| Module | Status |\n|---|---|\n")
		for _, child := range children {
			b.WriteString(fmt.Sprintf("| [[%s]] | %s |\n", child.name, child.status))
		}
	}

	if len(status.Adopted) > 0 {
		b.WriteString(fmt.Sprintf(
			"\n> `[~]` marks inherited documentation adopted from before answer records existed. "+
				"It is present and usable, and its coverage is **unverified**: %d phase(s).\n",
			len(status.Adopted)))
	}

	b.WriteString(fmt.Sprintf("\n**Node status:** %s\n", nodeStatus))
	if status.NextRecommended != "" {
		b.WriteString(fmt.Sprintf("**Next:** %s\n", status.NextRecommended))
	}
	b.WriteString(indexRegionEnd)
	return b.String()
}

// childNode is one discovered descendant and its computed node status.
type childNode struct {
	name   string
	status string
}

// discoverChildNodes finds the nodes nested under this one.
//
// A child is a subdirectory holding its own `<name>.md` index. That requirement
// is what keeps sibling directories which are not nodes — the SDD context folder
// and the features tree in in-project mode — out of the modules table.
func discoverChildNodes(res Resolution) []childNode {
	container := res.DocsRoot
	if res.Mode == ModeVault {
		container = filepath.Join(res.DocsRoot, "modules")
	}

	entries, err := os.ReadDir(container)
	if err != nil {
		return nil
	}

	var children []childNode
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !fileExists(filepath.Join(container, name, name+".md")) {
			continue
		}
		children = append(children, childNode{name: name, status: childStatus(res, name)})
	}
	return children
}

// childStatus reads a child's node status out of its own managed region rather
// than recomputing it, so the parent never has to load the child's records.
// A child with no managed region reports as adopted-unknown rather than as done.
func childStatus(res Resolution, name string) string {
	container := res.DocsRoot
	if res.Mode == ModeVault {
		container = filepath.Join(res.DocsRoot, "modules")
	}

	raw, err := os.ReadFile(filepath.Join(container, name, name+".md"))
	if err != nil {
		return "unknown"
	}
	const marker = "**Node status:** "
	at := strings.Index(string(raw), marker)
	if at < 0 {
		return "not managed"
	}
	rest := string(raw)[at+len(marker):]
	if end := strings.IndexAny(rest, "\r\n"); end >= 0 {
		return strings.TrimSpace(rest[:end])
	}
	return strings.TrimSpace(rest)
}

// deriveNodeStatus maps completed-phase counts onto the documented statuses.
func deriveNodeStatus(status Status) string {
	applicable := 0
	for _, ps := range status.Phases {
		if ps.State != StateNotApplicable {
			applicable++
		}
	}
	completed := len(status.Completed)
	adopted := len(status.Adopted)

	switch {
	case applicable > 0 && completed == applicable:
		return NodeStatusDocumented
	case applicable > 0 && adopted == applicable:
		return NodeStatusAdopted
	case completed > 0 || adopted > 0:
		return NodeStatusInProgress
	default:
		return NodeStatusStarted
	}
}

// writeFileAtomic writes via a temporary file in the destination directory and
// renames it into place, so a crash never leaves a partial artifact behind.
func writeFileAtomic(path string, content []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("creating temp file in %s: %w", dir, err)
	}
	tmpName := tmp.Name()
	// Best-effort cleanup: harmless once the rename has consumed the temp file.
	defer os.Remove(tmpName)

	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return fmt.Errorf("writing temp file for %s: %w", path, err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("syncing temp file for %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file for %s: %w", path, err)
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return fmt.Errorf("setting permissions for %s: %w", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("renaming temp file into %s: %w", path, err)
	}
	return nil
}

func writeJSONAtomic(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling %s: %w", path, err)
	}
	return writeFileAtomic(path, append(raw, '\n'))
}
