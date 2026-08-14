package docagent

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	pipelinepkg "github.com/zeshone/doc-agent-ai/internal/pipeline"
)

// ---------------------------------------------------------------------------
// Question bank / skills consistency guard
// ---------------------------------------------------------------------------
//
// The question bank moved the required interview topics out of the skill prose.
// That only helps while the two stay aligned: a topic added to the bank with no
// prose behind it makes the agent block on something it never learned to ask,
// and prose describing a topic the bank does not require is unenforced again.
//
// These tests fail loudly on either kind of drift.

// workflowOrderLine matches the documented phase chain in the orchestrator skill,
// e.g. "Always: idea → rec → prd → refine → tech → [ddd] → pti".
var workflowOrderLine = regexp.MustCompile(`Always:\s*(.+)$`)

func TestQuestionBankPhaseOrderMatchesTheOrchestratorSkill(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("skills", "doc-arch", "SKILL.md"))
	if err != nil {
		t.Fatalf("reading orchestrator skill: %v", err)
	}

	var documented []string
	for _, line := range strings.Split(string(raw), "\n") {
		match := workflowOrderLine.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		for _, token := range strings.Split(match[1], "→") {
			token = strings.TrimSpace(token)
			token = strings.Trim(token, "[]`")
			if token != "" {
				documented = append(documented, token)
			}
		}
		break
	}

	if len(documented) == 0 {
		t.Fatal("could not find the documented workflow order in skills/doc-arch/SKILL.md; " +
			"if the wording changed, update this guard rather than deleting it")
	}

	canonical := pipelinepkg.CanonicalPhaseOrder()
	if len(documented) != len(canonical) {
		t.Fatalf("skill documents %d phases %v, question bank declares %d %v",
			len(documented), documented, len(canonical), canonical)
	}
	for i := range canonical {
		if documented[i] != string(canonical[i]) {
			t.Errorf("phase %d: skill says %q, code says %q", i, documented[i], canonical[i])
		}
	}
}

func TestEveryQuestionBankPhaseShipsASkillAndACommand(t *testing.T) {
	bank, err := pipelinepkg.LoadQuestionBank()
	if err != nil {
		t.Fatalf("LoadQuestionBank: %v", err)
	}

	manifest := loadContentManifest(t)

	// The refine phase ships under the doc-refine command with the
	// doc-refinement skill, so the mapping cannot be derived from the phase id.
	skillFor := map[string]string{
		"idea": "doc-idea", "rec": "doc-rec", "prd": "doc-prd",
		"refine": "doc-refinement", "tech": "doc-tech",
		"ddd": "doc-ddd", "pti": "doc-pti",
	}
	commandFor := map[string]string{
		"idea": "doc-idea", "rec": "doc-rec", "prd": "doc-prd",
		"refine": "doc-refine", "tech": "doc-tech",
		"ddd": "doc-ddd", "pti": "doc-pti",
	}

	for _, spec := range bank.Phases {
		phase := string(spec.Phase)
		t.Run(phase, func(t *testing.T) {
			skill, ok := skillFor[phase]
			if !ok {
				t.Fatalf("phase %q has no skill mapping in this guard", phase)
			}
			if !containsStr(manifest.Skills, skill) {
				t.Errorf("phase %q needs skill %q but the manifest does not ship it", phase, skill)
			}
			if _, err := os.Stat(filepath.Join("skills", skill, "SKILL.md")); err != nil {
				t.Errorf("skill file for phase %q is missing: %v", phase, err)
			}

			command := commandFor[phase]
			path := filepath.Join("src", "content", "commands", command+".md")
			if _, err := os.Stat(path); err != nil {
				t.Errorf("phase %q names command %q but %s is missing: %v", phase, command, path, err)
			}
		})
	}
}

// citation matches the evidence comments in questionbank.yaml, such as
// "# skills/doc-rec/SKILL.md:86,93" or "# src/content/roles/doc-prd.md:76-84".
var citation = regexp.MustCompile(`([A-Za-z0-9_./-]+\.md):([0-9]+(?:[,\-][0-9]+)*)`)

func TestQuestionBankCitationsStillResolve(t *testing.T) {
	// Every topic in the bank cites the prose that requires it. Those citations
	// are the traceability that makes the bank reviewable, so a stale line number
	// is a real defect: it means the prose moved and nobody checked the bank.
	raw, err := os.ReadFile(filepath.Join("internal", "pipeline", "questionbank.yaml"))
	if err != nil {
		t.Fatalf("reading question bank: %v", err)
	}

	matches := citation.FindAllStringSubmatch(string(raw), -1)
	if len(matches) == 0 {
		t.Fatal("the question bank cites no evidence; every topic should point at the prose requiring it")
	}

	lineCounts := map[string]int{}

	for _, match := range matches {
		file, lineSpec := match[1], match[2]

		count, known := lineCounts[file]
		if !known {
			content, err := os.ReadFile(file)
			if err != nil {
				t.Errorf("citation %s does not resolve: %v", file, err)
				lineCounts[file] = -1
				continue
			}
			count = len(strings.Split(strings.TrimRight(string(content), "\n"), "\n"))
			lineCounts[file] = count
		}
		if count < 0 {
			continue
		}

		for _, part := range regexp.MustCompile(`[,\-]`).Split(lineSpec, -1) {
			line, err := strconv.Atoi(part)
			if err != nil {
				t.Errorf("citation %s:%s has an unparseable line number %q", file, lineSpec, part)
				continue
			}
			if line < 1 || line > count {
				t.Errorf("citation %s:%s points at line %d but the file has %d lines",
					file, lineSpec, line, count)
			}
		}
	}
}

func TestNoInterviewPhaseCanCompleteWithoutTopics(t *testing.T) {
	// A phase with zero required topics would report complete the moment its
	// artifact exists, which is the fabricated-completeness hole the bank closes.
	bank, err := pipelinepkg.LoadQuestionBank()
	if err != nil {
		t.Fatalf("LoadQuestionBank: %v", err)
	}

	for _, nodeType := range []pipelinepkg.NodeType{
		pipelinepkg.NodeSystem, pipelinepkg.NodeModule, pipelinepkg.NodeSubmodule,
	} {
		for _, spec := range bank.Phases {
			if spec.Kind != pipelinepkg.KindInterview {
				continue
			}
			topics := bank.RequiredTopics(spec.Phase, nodeType)
			if len(topics) == 0 {
				t.Errorf("phase %q requires no topics for node type %q", spec.Phase, nodeType)
			}
		}
	}
}

func TestTopicIdsAreGloballyUnique(t *testing.T) {
	// Each topic belongs to exactly one phase, the one that owns its altitude.
	// A id appearing in two phases means two phases ask for the same thing, which
	// is how the documentation turns repetitive and tedious to produce. The bank
	// header records which phase owns which altitude.
	bank, err := pipelinepkg.LoadQuestionBank()
	if err != nil {
		t.Fatalf("LoadQuestionBank: %v", err)
	}

	owner := map[string]pipelinepkg.PhaseID{}
	for _, spec := range bank.Phases {
		for _, topic := range spec.RequiredTopics {
			if prior, taken := owner[topic.ID]; taken {
				t.Errorf("topic %q is declared by both %q and %q; only one phase may own it",
					topic.ID, prior, spec.Phase)
				continue
			}
			owner[topic.ID] = spec.Phase
		}
	}
}

func TestSectionTitlesAreCanonicalEnglish(t *testing.T) {
	// The heading is the language-independent part of every artifact: prose is
	// written in the user's language, structure is not. A non-ASCII or accented
	// title would mean a translated heading leaked into the canonical vocabulary
	// and the validator would start depending on the documentation language again.
	bank, err := pipelinepkg.LoadQuestionBank()
	if err != nil {
		t.Fatalf("LoadQuestionBank: %v", err)
	}

	for _, spec := range bank.Phases {
		if spec.Kind != pipelinepkg.KindInterview {
			continue
		}
		t.Run(string(spec.Phase), func(t *testing.T) {
			if !isASCII(spec.DocumentTitle) {
				t.Errorf("document title %q is not plain ASCII English", spec.DocumentTitle)
			}
			for _, topic := range spec.RequiredTopics {
				if topic.Title == "" {
					t.Errorf("topic %q declares no section title", topic.ID)
					continue
				}
				if !isASCII(topic.Title) {
					t.Errorf("topic %q has a non-ASCII title %q", topic.ID, topic.Title)
				}
			}
		})
	}
}

func isASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return false
		}
	}
	return s != ""
}

func containsStr(haystack []string, needle string) bool {
	for _, candidate := range haystack {
		if candidate == needle {
			return true
		}
	}
	return false
}

// overlappingTopics are the pairs that collided in the first full live run. Each
// side must keep a note distinguishing it, because `topics` returns an id and a
// title and nothing else told the model these were different questions.
//
// The evidence: status reported the same recorded span for idea/problem-solved
// and rec/current-process, and again for idea/success-definition and
// prd/success-metrics. prd/security-privacy and ddd/constraints were asked with
// an identical prompt.
var overlappingTopics = map[string]pipelinepkg.PhaseID{
	"problem-solved":     pipelinepkg.PhaseIdea,
	"current-process":    pipelinepkg.PhaseRec,
	"success-definition": pipelinepkg.PhaseIdea,
	"success-metrics":    pipelinepkg.PhasePRD,
	"security-privacy":   pipelinepkg.PhasePRD,
	"constraints":        pipelinepkg.PhaseDDD,
}

func TestTopicsThatCollidedInPracticeCarryADistinguishingNote(t *testing.T) {
	bank, err := pipelinepkg.LoadQuestionBank()
	if err != nil {
		t.Fatalf("LoadQuestionBank: %v", err)
	}

	for topicID, phase := range overlappingTopics {
		t.Run(topicID, func(t *testing.T) {
			topic, ok := bank.Topic(phase, topicID)
			if !ok {
				t.Fatalf("phase %q no longer declares topic %q", phase, topicID)
			}
			if strings.TrimSpace(topic.Note) == "" {
				t.Errorf("topic %q lost its note; it was asked twice in practice without one", topicID)
			}
			// A note that does not name the other side explains nothing.
			if !strings.Contains(topic.Note, "/") {
				t.Errorf("note for %q does not name the sibling topic that owns the other side: %q",
					topicID, topic.Note)
			}
		})
	}
}

func TestTopicTitlesAreGloballyUniqueToo(t *testing.T) {
	// A heading is what a reader sees. Two phases owning the same one is the same
	// hazard as two phases owning the same id: "Constraints" read as restrictions
	// in general, and the privacy question got asked in ddd as well as prd.
	bank, err := pipelinepkg.LoadQuestionBank()
	if err != nil {
		t.Fatalf("LoadQuestionBank: %v", err)
	}

	owner := map[string]pipelinepkg.PhaseID{}
	for _, spec := range bank.Phases {
		for _, topic := range spec.RequiredTopics {
			if topic.Title == "" {
				continue
			}
			if prior, taken := owner[topic.Title]; taken {
				t.Errorf("title %q is used by both %q and %q", topic.Title, prior, spec.Phase)
				continue
			}
			owner[topic.Title] = spec.Phase
		}
	}
}
