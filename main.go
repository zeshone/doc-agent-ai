package main

import (
	"fmt"
	"os"
)

// version is set at build time via ldflags: -ldflags "-X main.version=3.0.0"
var version = "dev"

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(0)
	}

	switch os.Args[1] {
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

	case "--version":
		fmt.Printf("doc-agent-ai %s\n", version)

	case "--help":
		printHelp()

	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", os.Args[1])
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`doc-agent-ai — Multi-platform documentation workflow agent installer

Usage:
  doc-agent-ai <subcommand>

Subcommands:
  generate    Generate dist/ from embedded src/ + skills/
  install     Install doc-agent-ai to detected platforms
  uninstall   Remove doc-agent-ai from detected platforms

Flags:
  --version   Print version and exit
  --help      Print this help and exit`)
}
