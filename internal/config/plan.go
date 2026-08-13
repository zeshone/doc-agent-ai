package config

import (
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// InstallPlan — the seam that decouples the TUI/headless layer from the engine
// ---------------------------------------------------------------------------

// InstallPlan is a plain value object that captures all decisions needed to
// execute an install. Both the TUI wizard and the headless flags parser build
// an InstallPlan; the engine (installPlatforms, executeInstall) consumes only
// this struct and never imports any charm/TUI packages.
type InstallPlan struct {
	// Platforms is the resolved list of platform IDs to install (e.g. "opencode").
	Platforms []string

	// Mode is the documentation storage mode selected for this install.
	Mode DocsMode

	// BasePath is the normalized vault base path (e.g. "/home/user/docs/").
	// Only meaningful when Mode == ModeVault; empty for ModeInProject.
	BasePath string

	// Overwrite is a per-platform flag that controls whether existing agents
	// are overwritten. True = overwrite confirmed.
	Overwrite map[string]bool

	// PrevMode is the mode stored in the AppConfig BEFORE this install; used
	// to detect a mode-switch (e.g. in-project → vault) so cleanup can run.
	PrevMode DocsMode

	// Yes reflects the --yes flag: when true, the TUI confirmation step is
	// skipped and the install proceeds without interactive confirmation.
	Yes bool
}

// ---------------------------------------------------------------------------
// FlagSet — the headless command-line inputs parsed before the TUI is launched
// ---------------------------------------------------------------------------

// FlagSet captures the values of the headless install flags extracted from
// os.Args by the main pre-scan loop. Empty strings / false booleans mean "not
// provided" (fall through to TUI prompts or AppConfig defaults).
type FlagSet struct {
	// Platforms is a comma-separated list of platform IDs, e.g. "opencode,claude".
	Platforms string

	// DocsMode is the docs mode flag value: "vault" or "in-project".
	DocsMode string

	// Path is the vault base path flag value (absolute path string).
	Path string

	// Yes is the --yes flag: skip interactive confirmation.
	Yes bool
}

// validPlatformIDs is the canonical set of platform identifiers accepted by the
// install engine. Flags with unknown IDs are rejected.
var validPlatformIDs = map[string]bool{
	"opencode": true,
	"claude":   true,
	"copilot":  true,
	"qwen":     true,
	"pi":       true,
}

// parsePlanFromFlags builds an InstallPlan from explicit flags + AppConfig defaults.
// Flag values take precedence over config defaults; see resolution rules below.
//
// Resolution rules (per field):
//  1. Platforms: flag (CSV split) → config.Platforms → all (empty = prompt/all)
//  2. Mode: flag DocsMode → config.Mode → "vault"
//  3. BasePath: flag Path → config.Path → error (required for vault)
//
// Validation:
//   - Unknown platform IDs in --platforms cause an error.
//   - An invalid --docs-mode value causes an error.
//   - Vault mode with no path and no config default causes an error.
func ParsePlanFromFlags(flags FlagSet, cfg AppConfig) (InstallPlan, error) {
	var plan InstallPlan

	// --- Resolve platforms ---
	if flags.Platforms != "" {
		parts := strings.Split(flags.Platforms, ",")
		for i, p := range parts {
			parts[i] = strings.TrimSpace(p)
		}
		for _, id := range parts {
			if !validPlatformIDs[id] {
				return InstallPlan{}, fmt.Errorf("unknown platform %q: valid values are opencode, claude, copilot, qwen, pi", id)
			}
		}
		plan.Platforms = parts
	} else if len(cfg.Platforms) > 0 {
		plan.Platforms = cfg.Platforms
	}
	// else: plan.Platforms remains nil → "all / prompt" behaviour

	// --- Resolve mode ---
	rawMode := flags.DocsMode
	if rawMode == "" {
		rawMode = cfg.Mode
	}
	if rawMode == "" {
		rawMode = string(ModeVault) // built-in default
	}
	switch DocsMode(rawMode) {
	case ModeVault, ModeInProject:
		plan.Mode = DocsMode(rawMode)
	default:
		return InstallPlan{}, fmt.Errorf("invalid docs-mode %q: must be \"vault\" or \"in-project\"", rawMode)
	}

	// --- Resolve base path ---
	if plan.Mode == ModeVault {
		rawPath := flags.Path
		if rawPath == "" {
			rawPath = cfg.Path
		}
		if rawPath == "" {
			return InstallPlan{}, fmt.Errorf("vault mode requires a documentation base path; provide --path or run without --docs-mode to use the TUI")
		}
		plan.BasePath = ExpandUserPath(rawPath)
	}
	// in-project: BasePath stays empty (docs/doc-agent/ is implicit)

	// --- Carry over remaining fields ---
	plan.PrevMode = DocsMode(cfg.Mode)
	plan.Yes = flags.Yes

	return plan, nil
}
