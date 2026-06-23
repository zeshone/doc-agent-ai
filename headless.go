package main

import "fmt"

// ---------------------------------------------------------------------------
// Headless install flags — parsed in the main pre-scan before subcommand dispatch
// ---------------------------------------------------------------------------

// parseInstallFlags extracts the four install-specific flags from args:
//
//	--platforms <csv>   comma-separated platform IDs
//	--docs-mode <mode>  vault | in-project
//	--path <path>       vault base path
//	--yes               skip interactive confirmation
//
// Returns the populated FlagSet and the remaining args (flags consumed, subcommand
// and unrecognised args left intact). The function is a pure scan-and-consume loop
// so it can be called before os.Args is otherwise parsed.
func parseInstallFlags(args []string) (FlagSet, []string) {
	var flags FlagSet
	remaining := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--platforms":
			if i+1 < len(args) {
				flags.Platforms = args[i+1]
				i++ // consume value
			}
		case "--docs-mode":
			if i+1 < len(args) {
				flags.DocsMode = args[i+1]
				i++ // consume value
			}
		case "--path":
			if i+1 < len(args) {
				flags.Path = args[i+1]
				i++ // consume value
			}
		case "--yes":
			flags.Yes = true
		default:
			remaining = append(remaining, args[i])
		}
	}

	return flags, remaining
}

// hasInstallFlags reports whether any of the four install-specific flags are set.
// Decision order: explicit flags > TTY-interactive > error.
// When this returns true the main dispatcher routes directly to runHeadlessInstall,
// bypassing the interactive/TUI flow entirely.
func hasInstallFlags(f FlagSet) bool {
	return f.Platforms != "" || f.DocsMode != "" || f.Path != "" || f.Yes
}

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
