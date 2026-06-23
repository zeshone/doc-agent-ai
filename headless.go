package docagent

import (
	"fmt"

	configpkg "github.com/zeshone/doc-agent-ai/internal/config"
	installpkg "github.com/zeshone/doc-agent-ai/internal/install"
)

// ---------------------------------------------------------------------------
// Headless install flags — parsed in the main pre-scan before subcommand dispatch
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// runHeadlessInstall — non-interactive install driven by flags
// ---------------------------------------------------------------------------

// runHeadlessInstall executes a full install without a TTY or TUI.
// It is called from main when any install flag is present.
//
// Steps:
//  1. Load AppConfig for defaults.
//  2. parsePlanFromFlags to validate flags and build InstallPlan.
//  3. Build the in-memory bundle.
//  4. Detect platforms; filter to plan.Platforms.
//  5. ExecuteInstall with a stdout Reporter.
func runHeadlessInstall(flags FlagSet) error {
	bundle, err := BuildBundle()
	if err != nil {
		return fmt.Errorf("build content: %w", err)
	}
	return runHeadlessInstallWithBundle(flags, bundle)
}

func RunHeadlessInstall(flags FlagSet) error {
	return runHeadlessInstall(flags)
}

func runHeadlessInstallWithBundle(flags FlagSet, bundle Bundle) error {
	r := installpkg.NewStdoutReporter()

	// --- Step 1: Load config for defaults ---
	cfg, _, cfgErr := configpkg.Load()
	if cfgErr != nil {
		// Non-fatal; fall through with empty defaults and let flag validation decide.
		r.Warn("could not read config: " + cfgErr.Error())
	}

	// --- Step 2: Parse and validate InstallPlan ---
	plan, err := configpkg.ParsePlanFromFlags(flags, cfg)
	if err != nil {
		return fmt.Errorf("invalid flags: %w", err)
	}

	manifest := bundle.Manifest
	if missing := installpkg.ValidateBundleExport(bundle); len(missing) > 0 {
		return fmt.Errorf("incomplete bundle: %s", installpkg.SummarizeMissingArtifacts(missing))
	}

	// --- Step 3: Detect platforms ---
	// Platform filtering to plan.Platforms is executeInstall's responsibility —
	// pass the full detected universe and let the engine resolve.
	allDetected := installpkg.DetectAllPlatforms(manifest)

	// --- Step 4: Execute install ---
	return installpkg.ExecuteInstallExport(bundle, plan, allDetected, r)
}
