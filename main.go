package main

import (
	"fmt"
	"os"
)

// version is set at build time via ldflags: -ldflags "-X main.version=3.0.0"
var version = "dev"

func main() {
	// Pre-scan os.Args for --copilot-path <path> before the subcommand switch.
	// We do this manually to stay consistent with the rest of the arg parsing.
	args := os.Args[1:]
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--copilot-path" {
			copilotPathOverride = args[i+1]
			// Remove the flag and its value so the switch below sees clean args.
			args = append(args[:i], args[i+2:]...)
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
  --version              Print version and exit
  --help                 Print this help and exit`)
}
