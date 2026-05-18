package main

import (
	"fmt"
	"os"
)

// version is set at build time via ldflags: -ldflags "-X main.version=3.0.0"
var version = "dev"

func main() {
	// Pre-scan os.Args for path-override flags before the subcommand switch.
	// We do this manually to stay consistent with the rest of the arg parsing.
	// The loop restarts after each match so flags can appear in any order.
	args := os.Args[1:]
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
		if err := installInteractive(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "uninstall":
		if err := uninstallInteractive(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
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

Flags:
  --copilot-path <path>  Override the GitHub Copilot home directory used during
                         install/uninstall (bypasses all auto-detection).
  --pi-path <path>       Override the Pi agent home directory used during
                         install/uninstall (bypasses all auto-detection and
                         the PI_CODING_AGENT_DIR environment variable).
  --version              Print version and exit
  --help                 Print this help and exit`)
}
