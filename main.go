package main

import (
	"fmt"
	"os"
)

// version is set at build time via ldflags: -ldflags "-X main.version=3.0.0"
var version = "dev"

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
				copilotPathOverride = args[i+1]
			case "--pi-path":
				piPathOverride = args[i+1]
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
	installFlags, args := parseInstallFlags(args)

	if len(args) == 0 {
		printHelp()
		os.Exit(0)
	}

	switch args[0] {
	case "generate":
		if err := generate("dist"); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("dist generated from src canonical content")

	case "install":
		// Decision order (spec F1): explicit install flags > TTY-interactive > error.
		// --yes alone also triggers the headless path (skips confirmation).
		if hasInstallFlags(installFlags) {
			if err := runHeadlessInstall(installFlags, ""); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		} else if isTerminal() {
			// No flags, TTY present: launch the Bubbletea install wizard.
			if err := runInstallTUI(); err != nil {
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
		if isTerminal() {
			if err := runUninstallTUI(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		} else {
			if err := uninstallInteractive(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}

	case "--version":
		fmt.Printf("doc-agent-ai %s\n", version)

	case "--help":
		printHelp()

	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", args[0])
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`doc-agent-ai — Multi-platform documentation workflow agent installer

Usage:
  doc-agent-ai [flags] <subcommand>

Subcommands:
  generate    Generate dist/ from embedded src/ + skills/
  install     Install doc-agent-ai to detected platforms
  uninstall   Remove doc-agent-ai from detected platforms

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
