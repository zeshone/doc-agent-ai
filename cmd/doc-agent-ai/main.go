package main

import (
	"fmt"
	"os"

	docagent "github.com/zeshone/doc-agent-ai"
	buildpkg "github.com/zeshone/doc-agent-ai/internal/build"
	configpkg "github.com/zeshone/doc-agent-ai/internal/config"
	installpkg "github.com/zeshone/doc-agent-ai/internal/install"
	pipelinepkg "github.com/zeshone/doc-agent-ai/internal/pipeline"
	tuipkg "github.com/zeshone/doc-agent-ai/internal/tui"
)

func main() {
	// Pre-scan os.Args for platform-path overrides and install flags before
	// the subcommand switch. The platform-path loop restarts after each match
	// so flags can appear in any order. Install flags are extracted in a single
	// pass via parseInstallFlags.
	args := os.Args[1:]

	// Pass 1: consume --copilot-path and --pi-path (platform overrides).
	for {
		matched := false
		for i := 0; i < len(args)-1; i++ {
			switch args[i] {
			case "--copilot-path":
				installpkg.SetCopilotPathOverride(args[i+1])
			case "--pi-path":
				installpkg.SetPiPathOverride(args[i+1])
			default:
				continue
			}
			args = append(args[:i], args[i+2:]...)
			matched = true
			break
		}
		if !matched {
			break
		}
	}

	// Pass 2: extract install-specific flags (--platforms, --docs-mode, --path, --yes).
	// These are consumed from args; the subcommand and unrecognised tokens remain.
	installFlags, args := configpkg.ParseInstallFlags(args)

	if len(args) == 0 {
		if tuipkg.IsTerminal() {
			bundle, err := docagent.BuildBundle()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if err := tuipkg.RunApp(bundle); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		}
		fmt.Fprintln(os.Stderr, "Error: no TTY detected. Run with a TTY, or use `install --platforms ...`.")
		os.Exit(1)
	}

	switch args[0] {
	case "generate":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: doc-agent-ai generate <dir>")
			os.Exit(1)
		}
		if err := docagent.Generate(args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("bundle generated at %s\n", args[1])

	case "install":
		// Decision order (spec F1): explicit install flags > TTY-interactive > error.
		// --yes alone also triggers the headless path (skips confirmation).
		if configpkg.HasInstallFlags(installFlags) {
			if err := docagent.RunHeadlessInstall(installFlags); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		} else if tuipkg.IsTerminal() {
			// No flags, TTY present: launch the Bubbletea install wizard.
			bundle, err := docagent.BuildBundle()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if err := tuipkg.RunInstallTUI(bundle); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		} else {
			// No flags, no TTY: non-interactive context; guide the user.
			fmt.Fprintln(os.Stderr, "Error: no TTY detected and no install flags provided.")
			fmt.Fprintln(os.Stderr, "Use flags for non-interactive install, e.g.:")
			fmt.Fprintln(os.Stderr, "  doc-agent-ai install --platforms opencode --docs-mode vault --path /docs --yes")
			fmt.Fprintln(os.Stderr, "Run --help for all available flags.")
			os.Exit(1)
		}

	case "uninstall":
		// Decision order: TTY present → Bubbletea TUI; no TTY → bufio fallback.
		//
		// Asymmetry with install: install errors on no-TTY + no-flags because it
		// needs user input (mode, path) that cannot be reasonably defaulted.
		// Uninstall only needs a yes/no confirmation, so the bufio fallback is safe
		// and avoids breaking automated environments that call uninstall directly.
		if tuipkg.IsTerminal() {
			bundle, err := docagent.BuildBundle()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if err := tuipkg.RunUninstallTUI(bundle); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		} else {
			bundle, err := docagent.BuildBundle()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if err := installpkg.UninstallInteractive(bundle.Manifest); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}

	case "topics":
		os.Exit(pipelinepkg.RunTopics(args[1:], os.Stdout, os.Stderr))

	case "status":
		os.Exit(pipelinepkg.RunStatus(args[1:], pipelineEnvironment(), os.Stdout, os.Stderr))

	case "validate":
		os.Exit(pipelinepkg.RunValidate(args[1:], pipelineEnvironment(), os.Stdout, os.Stderr))

	case "commit-phase":
		os.Exit(pipelinepkg.RunCommitPhase(args[1:], pipelineEnvironment(), os.Stdout, os.Stderr))

	case "decide-phase":
		os.Exit(pipelinepkg.RunDecidePhase(args[1:], pipelineEnvironment(), os.Stdout, os.Stderr))

	case "--version":
		fmt.Printf("doc-agent-ai %s\n", buildpkg.Version)

	case "--help":
		printHelp()

	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", args[0])
		printHelp()
		os.Exit(1)
	}
}

// pipelineEnvironment builds the ambient input the pipeline commands need.
//
// A missing or unreadable global config is not fatal here: the pipeline reports
// an unresolvable destination as a typed undetermined status, which is more
// useful to the caller than a bare CLI error.
func pipelineEnvironment() pipelinepkg.Environment {
	env := pipelinepkg.Environment{}
	if cwd, err := os.Getwd(); err == nil {
		env.ProjectRoot = cwd
	}
	if cfg, _, err := configpkg.Load(); err == nil {
		env.GlobalMode = cfg.Mode
		env.GlobalBasePath = cfg.Path
	}
	return env
}

func printHelp() {
	fmt.Println(`doc-agent-ai — Multi-platform documentation workflow agent installer

Usage:
  doc-agent-ai [flags] <subcommand>

Subcommands:
  generate    Generate bundle output to an explicit directory
  install     Install doc-agent-ai to detected platforms
  uninstall   Remove doc-agent-ai from detected platforms

Pipeline subcommands (typed JSON on stdout; the orchestrator routes on these):
  topics        Print required interview topics
                  --phase <id> --node-type <system|module|submodule>
  status        Print a node's computed phase status
                  --node <system[/module[/submodule]]>
  validate      Check a phase submission without writing anything
                  --node <n> --phase <p> --answers <f> --sections <f>
  commit-phase  Validate a submission and write it only if it passes
                  --node <n> --phase <p> --answers <f> --sections <f>
                  --audit <f> instead of --answers for audit phases
                  Section headings are rendered by the program from the
                  question bank; --sections carries prose per topic id.
  decide-phase  Record a decision about an optional phase
                  --node <n> --phase <p> --decision <accepted|declined>

  Exit codes: 0 affirmative, 1 usage or environment error, 2 refused verdict.

Install flags (bypass TUI; trigger headless mode when any is present):
  --platforms <ids>      Comma-separated platform IDs (opencode, claude, copilot, qwen, pi)
  --docs-mode <mode>     Documentation mode: vault (default) or in-project
  --path <path>          Vault base path (required for --docs-mode vault)
  --yes                  Skip interactive confirmation prompts

Platform flags:
  --copilot-path <path>  Override the GitHub Copilot home directory used during
                         install/uninstall (bypasses all auto-detection).
  --pi-path <path>       Override the Pi agent home directory used during
                         install/uninstall (bypasses all auto-detection and
                         the PI_CODING_AGENT_DIR environment variable).

General flags:
  --version              Print version and exit
  --help                 Print this help and exit`)
}
