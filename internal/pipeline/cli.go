package pipeline

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
)

// Exit codes. The distinction matters for scripting: a rejected submission is a
// successful run that produced a negative verdict, not a broken invocation.
const (
	// ExitOK means the command ran and its verdict is affirmative.
	ExitOK = 0
	// ExitUsage means the invocation or the environment was wrong.
	ExitUsage = 1
	// ExitVerdict means the command ran and refused: rejected, or undetermined.
	ExitVerdict = 2
)

// emit writes a contract as indented JSON. Every boundary-crossing value leaves
// this package as versioned JSON carrying its own schema name.
func emit(out io.Writer, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "%s\n", raw)
	return err
}

func usageError(errOut io.Writer, format string, args ...any) int {
	fmt.Fprintf(errOut, "Error: "+format+"\n", args...)
	return ExitUsage
}

// RunTopics prints the question bank, optionally narrowed to one phase and node
// type. This is how the model learns which topics a phase requires without the
// list living in skill prose.
func RunTopics(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("topics", flag.ContinueOnError)
	fs.SetOutput(errOut)
	phaseArg := fs.String("phase", "", "phase id; omit for the whole bank")
	nodeTypeArg := fs.String("node-type", "", "system, module or submodule; narrows conditional topics")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}

	bank, err := LoadQuestionBank()
	if err != nil {
		return usageError(errOut, "%v", err)
	}

	if *phaseArg == "" {
		if err := emit(out, bank); err != nil {
			return usageError(errOut, "%v", err)
		}
		return ExitOK
	}

	phase, err := ParsePhase(*phaseArg)
	if err != nil {
		return usageError(errOut, "%v", err)
	}
	spec, ok := bank.Phase(phase)
	if !ok {
		return usageError(errOut, "phase %q is not in the question bank", phase)
	}

	payload := struct {
		SchemaName     string     `json:"schemaName"`
		Phase          PhaseID    `json:"phase"`
		Kind           Kind       `json:"kind"`
		Optional       bool       `json:"optional"`
		Artifact       string     `json:"artifact"`
		DocumentTitle  string     `json:"documentTitle,omitempty"`
		RequiredTopics []Topic    `json:"requiredTopics"`
		AuditRule      *AuditRule `json:"auditRule,omitempty"`
	}{
		SchemaName:    QuestionBankSchema,
		Phase:         phase,
		Kind:          spec.Kind,
		Optional:      spec.Optional,
		Artifact:      spec.Artifact,
		DocumentTitle: spec.DocumentTitle,
		AuditRule:     spec.AuditRule,
	}

	if *nodeTypeArg == "" {
		// With no node type, report every declared topic rather than guessing.
		payload.RequiredTopics = spec.RequiredTopics
	} else {
		nodeType, err := parseNodeType(*nodeTypeArg)
		if err != nil {
			return usageError(errOut, "%v", err)
		}
		payload.RequiredTopics = bank.TopicsFor(phase, nodeType)
	}
	if payload.RequiredTopics == nil {
		payload.RequiredTopics = []Topic{}
	}

	if err := emit(out, payload); err != nil {
		return usageError(errOut, "%v", err)
	}
	return ExitOK
}

// RunStatus prints the node's computed status.
//
// It exits ExitOK whenever it produced a status, including a blocked one: a
// blocked status is a correct answer, not a failed invocation. Only an
// undetermined result raises the exit code, because then the program is telling
// the caller it could not decide.
func RunStatus(args []string, env Environment, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(errOut)
	nodeArg := fs.String("node", "", "node identifier, e.g. acme-hr or acme-hr/payroll")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	if *nodeArg == "" {
		return usageError(errOut, "status needs --node <system[/module[/submodule]]>")
	}

	node, err := ParseNode(*nodeArg)
	if err != nil {
		return usageError(errOut, "%v", err)
	}
	bank, err := LoadQuestionBank()
	if err != nil {
		return usageError(errOut, "%v", err)
	}

	status := ComputeStatus(node, env, bank)
	if err := emit(out, status); err != nil {
		return usageError(errOut, "%v", err)
	}
	if status.NextAction.Kind == ActionCannotDetermine {
		return ExitVerdict
	}
	return ExitOK
}

// RunValidate checks a submission without writing anything.
func RunValidate(args []string, env Environment, out, errOut io.Writer) int {
	sub, bank, code := parseSubmission("validate", args, errOut)
	if code != ExitOK {
		return code
	}

	result := Validate(sub, bank)
	if err := emit(out, result); err != nil {
		return usageError(errOut, "%v", err)
	}
	if !result.Accepted() {
		return ExitVerdict
	}
	return ExitOK
}

// RunCommitPhase validates a submission and writes it only if it passes.
func RunCommitPhase(args []string, env Environment, out, errOut io.Writer) int {
	sub, bank, code := parseSubmission("commit-phase", args, errOut)
	if code != ExitOK {
		return code
	}

	result := Commit(sub, env, bank)
	if err := emit(out, result); err != nil {
		return usageError(errOut, "%v", err)
	}
	if result.Result != CommitWritten {
		return ExitVerdict
	}
	return ExitOK
}

// RunDecidePhase records the user's choice about an optional phase so it is not
// re-asked in a later session.
func RunDecidePhase(args []string, env Environment, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("decide-phase", flag.ContinueOnError)
	fs.SetOutput(errOut)
	nodeArg := fs.String("node", "", "node identifier")
	phaseArg := fs.String("phase", "", "optional phase id")
	decisionArg := fs.String("decision", "", "accepted or declined")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	if *nodeArg == "" || *phaseArg == "" || *decisionArg == "" {
		return usageError(errOut, "decide-phase needs --node, --phase and --decision")
	}

	node, err := ParseNode(*nodeArg)
	if err != nil {
		return usageError(errOut, "%v", err)
	}
	phase, err := ParsePhase(*phaseArg)
	if err != nil {
		return usageError(errOut, "%v", err)
	}
	bank, err := LoadQuestionBank()
	if err != nil {
		return usageError(errOut, "%v", err)
	}

	if err := RecordDecision(node, env, bank, phase, OptionalPhaseDecision(*decisionArg)); err != nil {
		return usageError(errOut, "%v", err)
	}

	status := ComputeStatus(node, env, bank)
	if err := emit(out, status); err != nil {
		return usageError(errOut, "%v", err)
	}
	return ExitOK
}

// RunDoctor aligns pre-existing documentation with the pipeline.
//
// Reporting is the default and writing is opt-in, because this command touches
// documentation the user wrote by hand.
func RunDoctor(args []string, env Environment, now string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(errOut)
	nodeArg := fs.String("node", "", "node identifier")
	applyArg := fs.Bool("apply", false, "write the plan; omit to report only")
	checkArg := fs.Bool("check", false, "report only (the default)")
	archetypeArg := fs.String("archetype", "", "bounded or evolving, when it cannot be determined")
	recursiveArg := fs.Bool("recursive", false, "also adopt nodes nested under this one")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	if *nodeArg == "" {
		return usageError(errOut, "doctor needs --node <system[/module[/submodule]]>")
	}
	if *applyArg && *checkArg {
		return usageError(errOut, "--apply and --check contradict each other; pick one")
	}
	if *archetypeArg != "" && *archetypeArg != ArchetypeBounded && *archetypeArg != ArchetypeEvolving {
		return usageError(errOut, "--archetype must be %q or %q", ArchetypeBounded, ArchetypeEvolving)
	}

	node, err := ParseNode(*nodeArg)
	if err != nil {
		return usageError(errOut, "%v", err)
	}
	bank, err := LoadQuestionBank()
	if err != nil {
		return usageError(errOut, "%v", err)
	}

	report := Doctor(node, env, bank, DoctorOptions{
		Apply:     *applyArg,
		Archetype: *archetypeArg,
		Recursive: *recursiveArg,
		Now:       now,
	})
	if err := emit(out, report); err != nil {
		return usageError(errOut, "%v", err)
	}
	if report.HasBlockers() {
		return ExitVerdict
	}
	return ExitOK
}

// parseSubmission builds a Submission from the shared validate/commit flags.
func parseSubmission(name string, args []string, errOut io.Writer) (Submission, QuestionBank, int) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(errOut)
	nodeArg := fs.String("node", "", "node identifier")
	phaseArg := fs.String("phase", "", "phase id")
	answersArg := fs.String("answers", "", "path to a docagent.answers/v1 file")
	auditArg := fs.String("audit", "", "path to a docagent.audit/v1 file")
	sectionsArg := fs.String("sections", "", "path to a docagent.sections/v1 file")
	if err := fs.Parse(args); err != nil {
		return Submission{}, QuestionBank{}, ExitUsage
	}
	if *nodeArg == "" || *phaseArg == "" {
		return Submission{}, QuestionBank{}, usageError(errOut, "%s needs --node and --phase", name)
	}

	node, err := ParseNode(*nodeArg)
	if err != nil {
		return Submission{}, QuestionBank{}, usageError(errOut, "%v", err)
	}
	phase, err := ParsePhase(*phaseArg)
	if err != nil {
		return Submission{}, QuestionBank{}, usageError(errOut, "%v", err)
	}
	bank, err := LoadQuestionBank()
	if err != nil {
		return Submission{}, QuestionBank{}, usageError(errOut, "%v", err)
	}

	spec, ok := bank.Phase(phase)
	if !ok {
		return Submission{}, QuestionBank{}, usageError(errOut, "phase %q is not in the question bank", phase)
	}

	sub := Submission{Node: node, Phase: phase}

	if *answersArg != "" {
		raw, err := os.ReadFile(*answersArg)
		if err != nil {
			return Submission{}, QuestionBank{}, usageError(errOut, "reading answers: %v", err)
		}
		var record AnswerRecord
		if err := json.Unmarshal(raw, &record); err != nil {
			return Submission{}, QuestionBank{}, usageError(errOut, "parsing answers: %v", err)
		}
		sub.Answers = &record
	}

	if *auditArg != "" {
		raw, err := os.ReadFile(*auditArg)
		if err != nil {
			return Submission{}, QuestionBank{}, usageError(errOut, "reading audit: %v", err)
		}
		var record AuditRecord
		if err := json.Unmarshal(raw, &record); err != nil {
			return Submission{}, QuestionBank{}, usageError(errOut, "parsing audit: %v", err)
		}
		sub.Audit = &record
	}

	if *sectionsArg != "" {
		raw, err := os.ReadFile(*sectionsArg)
		if err != nil {
			return Submission{}, QuestionBank{}, usageError(errOut, "reading sections: %v", err)
		}
		var input SectionInput
		if err := json.Unmarshal(raw, &input); err != nil {
			return Submission{}, QuestionBank{}, usageError(errOut, "parsing sections: %v", err)
		}
		if input.SchemaName != SectionsSchema {
			return Submission{}, QuestionBank{}, usageError(errOut,
				"sections file declares schema %q, expected %q", input.SchemaName, SectionsSchema)
		}
		sub.Content = input
	}

	// Guide the caller before the validator does, so a missing flag reads as a
	// usage problem rather than a content rejection.
	switch spec.Kind {
	case KindAudit:
		if sub.Audit == nil {
			return Submission{}, QuestionBank{}, usageError(errOut,
				"phase %q is an audit and needs --audit <file>", phase)
		}
	default:
		if sub.Answers == nil {
			return Submission{}, QuestionBank{}, usageError(errOut,
				"phase %q is an interview and needs --answers <file>", phase)
		}
		if *sectionsArg == "" {
			return Submission{}, QuestionBank{}, usageError(errOut,
				"phase %q writes an artifact and needs --sections <file>", phase)
		}
	}

	return sub, bank, ExitOK
}

func parseNodeType(raw string) (NodeType, error) {
	switch NodeType(raw) {
	case NodeSystem:
		return NodeSystem, nil
	case NodeModule:
		return NodeModule, nil
	case NodeSubmodule:
		return NodeSubmodule, nil
	}
	return "", fmt.Errorf("unknown node type %q: expected %q, %q or %q",
		raw, NodeSystem, NodeModule, NodeSubmodule)
}
