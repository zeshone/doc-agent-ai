package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Documentation storage modes, per src/templates/path-resolution.md.tmpl.
const (
	ModeVault     = "vault"
	ModeInProject = "in-project"
)

// markerFileName is the per-project mode marker written at the project root.
const markerFileName = ".doc-agent.json"

// stateDirName holds machine-owned pipeline state next to the documentation.
//
// Deliberately not ".doc-agent" — that would sit confusingly beside the
// existing ".doc-agent.json" mode marker. Renaming the marker instead would
// break every existing install for no gain.
const stateDirName = ".doc-agent-state"

// Environment is the ambient input to path resolution. It is passed in rather
// than read from the process so resolution stays testable against a temp dir
// and never depends on a real home directory.
type Environment struct {
	// ProjectRoot is where the mode marker is looked up and where in-project
	// documentation is written.
	ProjectRoot string
	// GlobalMode is the configured mode from ~/.doc-agent-ai.json. Empty means
	// unset, which resolves to vault.
	GlobalMode string
	// GlobalBasePath is the vault root. Required in vault mode.
	GlobalBasePath string
}

// Resolution is the resolved on-disk location of one node's documentation.
type Resolution struct {
	Node Node `json:"-"`
	// Mode is the effective storage mode.
	Mode string `json:"mode"`
	// ModeResolvedBy records which precedence level decided the mode: "marker",
	// "global" or "default". Reported so an operator can see why files landed
	// where they did.
	ModeResolvedBy string `json:"modeResolvedBy"`
	// DocsRoot is the directory holding this node's artifacts.
	DocsRoot string `json:"docsRoot"`
	// ArtifactPrefix is prepended to artifact filenames. It is the node short
	// name in vault mode and empty in-project mode, where files are bare
	// (docs/doc-agent/_prd.md).
	ArtifactPrefix string `json:"-"`
	// IndexPath is the node's index file.
	IndexPath string `json:"indexPath"`
}

// marker is the subset of the project marker file this package reads. Other
// keys are ignored so the file stays additive.
type marker struct {
	Mode string `json:"mode"`
}

// Resolve computes where a node's documentation lives.
//
// It fails rather than guesses. An unrecognised marker mode or a vault with no
// base path is an error, because the fallback would silently write the user's
// documentation somewhere they did not choose.
func Resolve(node Node, env Environment) (Resolution, error) {
	mode, resolvedBy, err := resolveMode(env)
	if err != nil {
		return Resolution{}, err
	}

	res := Resolution{
		Node:           node,
		Mode:           mode,
		ModeResolvedBy: resolvedBy,
	}

	switch mode {
	case ModeVault:
		if env.GlobalBasePath == "" {
			return Resolution{}, fmt.Errorf(
				"vault mode needs a base path but none is configured: run `doc-agent-ai install --docs-mode vault --path <path>` or set a project marker")
		}
		// skills/doc-arch/SKILL.md:23-27 — <base>/<system>/ then modules/<name>/
		// for each level below the system.
		parts := []string{env.GlobalBasePath, node.Segments[0]}
		for _, segment := range node.Segments[1:] {
			parts = append(parts, "modules", segment)
		}
		res.DocsRoot = filepath.Join(parts...)
		res.ArtifactPrefix = node.ShortName

	case ModeInProject:
		// src/templates/path-resolution.md.tmpl:5 — docs/doc-agent/ with no
		// <system> folder and no modules/ nesting.
		parts := []string{env.ProjectRoot, "docs", "doc-agent"}
		parts = append(parts, node.Segments[1:]...)
		res.DocsRoot = filepath.Join(parts...)
		res.ArtifactPrefix = ""

	default:
		return Resolution{}, fmt.Errorf("unsupported documentation mode %q", mode)
	}

	// skills/doc-arch/SKILL.md:33 — the index is <node>.md, placed in the node's
	// own folder. This keeps its name in both modes; only artifact prefixes differ.
	res.IndexPath = filepath.Join(res.DocsRoot, node.ShortName+".md")
	return res, nil
}

func resolveMode(env Environment) (mode string, resolvedBy string, err error) {
	if env.ProjectRoot != "" {
		markerPath := filepath.Join(env.ProjectRoot, markerFileName)
		raw, readErr := os.ReadFile(markerPath)
		switch {
		case readErr == nil:
			var m marker
			if err := json.Unmarshal(raw, &m); err != nil {
				return "", "", fmt.Errorf("parsing %s: %w", markerPath, err)
			}
			if m.Mode != "" {
				if !validMode(m.Mode) {
					return "", "", fmt.Errorf(
						"%s declares mode %q: expected %q or %q", markerPath, m.Mode, ModeVault, ModeInProject)
				}
				return m.Mode, "marker", nil
			}
		case !os.IsNotExist(readErr):
			return "", "", fmt.Errorf("reading %s: %w", markerPath, readErr)
		}
	}

	if env.GlobalMode != "" {
		if !validMode(env.GlobalMode) {
			return "", "", fmt.Errorf(
				"configured global mode %q is not %q or %q", env.GlobalMode, ModeVault, ModeInProject)
		}
		return env.GlobalMode, "global", nil
	}

	// path-resolution precedence ends at vault, preserving pre-v4 behaviour.
	return ModeVault, "default", nil
}

func validMode(mode string) bool {
	return mode == ModeVault || mode == ModeInProject
}

// ArtifactPath is the full path of a phase's artifact.
func (r Resolution) ArtifactPath(phase PhaseID, bank QuestionBank) string {
	spec, ok := bank.Phase(phase)
	if !ok {
		return ""
	}
	return filepath.Join(r.DocsRoot, spec.ArtifactName(r.ArtifactPrefix))
}

// StateDir is the machine-owned state directory for this node.
func (r Resolution) StateDir() string {
	return filepath.Join(r.DocsRoot, stateDirName)
}

// AnswerRecordPath is where a phase's answer record lives. The filename keeps
// the node short name even in-project mode, so a directory listing stays
// readable when several nodes' records are inspected together.
func (r Resolution) AnswerRecordPath(phase PhaseID) string {
	return filepath.Join(r.StateDir(), "answers",
		fmt.Sprintf("%s.%s.json", r.Node.ShortName, phase))
}

// SectionInputPath is where the per-topic prose that produced an artifact is
// stored. Persisting the input, not only the rendered output, is what makes a
// phase re-submittable: a later phase that needs to correct one section reads
// this back and replaces that one key instead of reassembling every section.
func (r Resolution) SectionInputPath(phase PhaseID) string {
	return filepath.Join(r.StateDir(), "sections",
		fmt.Sprintf("%s.%s.json", r.Node.ShortName, phase))
}

// AuditRecordPath is where an audit phase's verdicts live. Audit phases keep a
// separate record because their artifact belongs to an earlier phase: refine
// audits the PRD, so PRD file presence says nothing about the audit.
func (r Resolution) AuditRecordPath(phase PhaseID) string {
	return filepath.Join(r.StateDir(), "audits",
		fmt.Sprintf("%s.%s.json", r.Node.ShortName, phase))
}

// DecisionsPath records decisions about optional phases, so a phase the user
// declined stays declined across sessions instead of being re-asked forever.
func (r Resolution) DecisionsPath() string {
	return filepath.Join(r.StateDir(), "decisions.json")
}
