package main

import "fmt"

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
func runHeadlessInstall(flags FlagSet, _ string) error {
	bundle, err := BuildBundle()
	if err != nil {
		return fmt.Errorf("build content: %w", err)
	}
	return runHeadlessInstallWithBundle(flags, bundle)
}

func runHeadlessInstallWithBundle(flags FlagSet, bundle Bundle) error {
	r := defaultReporter

	// --- Step 1: Load config for defaults ---
	cfg, _, cfgErr := loadConfig()
	if cfgErr != nil {
		// Non-fatal; fall through with empty defaults and let flag validation decide.
		r.Warn("could not read config: " + cfgErr.Error())
	}

	// --- Step 2: Parse and validate InstallPlan ---
	plan, err := parsePlanFromFlags(flags, cfg)
	if err != nil {
		return fmt.Errorf("invalid flags: %w", err)
	}

	manifest := bundle.Manifest
	if missing := ValidateBundle(bundle); len(missing) > 0 {
		return fmt.Errorf("incomplete bundle: %d missing artifacts", len(missing))
	}

	// --- Step 3: Detect platforms ---
	// Platform filtering to plan.Platforms is executeInstall's responsibility —
	// pass the full detected universe and let the engine resolve.
	allDetected := detectAllPlatforms(manifest)

	// --- Step 4: Execute install ---
	return ExecuteInstall(bundle, plan, allDetected, r)
}
