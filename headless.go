package main

import (
	"fmt"
	"os"
)

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
// distDirOverride allows tests to inject a pre-generated dist directory
// instead of the default "dist/" path used in production.
//
// Steps:
//  1. Load AppConfig for defaults.
//  2. parsePlanFromFlags to validate flags and build InstallPlan.
//  3. Auto-generate dist if missing (same as installInteractive).
//  4. Read and validate manifest.
//  5. Detect platforms; filter to plan.Platforms.
//  6. executeInstall with a stdout Reporter.
func runHeadlessInstall(flags FlagSet, distDirOverride string) error {
	// --- Step 1: Load config for defaults ---
	cfg, _, cfgErr := loadConfig()
	if cfgErr != nil {
		// Non-fatal; fall through with empty defaults and let flag validation decide.
		warn("could not read config: " + cfgErr.Error())
	}

	// --- Step 2: Parse and validate InstallPlan ---
	plan, err := parsePlanFromFlags(flags, cfg)
	if err != nil {
		return fmt.Errorf("invalid flags: %w", err)
	}

	// --- Step 3: Resolve dist directory ---
	distDir := distDirOverride
	if distDir == "" {
		distDir = "dist"
	}

	// Auto-generate dist if missing.
	if _, err := os.Stat(distDir + "/manifest.json"); os.IsNotExist(err) {
		info("Auto-generating dist/ from embedded source...")
		if err := generate(distDir); err != nil {
			return fmt.Errorf("auto-generate dist: %w", err)
		}
		ok("dist/ generated")
	}

	// --- Step 4: Read and validate manifest ---
	manifest, err := readManifestFrom(distDir)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}
	if missing := validateDist(manifest, distDir); len(missing) > 0 {
		return fmt.Errorf("incomplete dist: %d missing artifacts", len(missing))
	}

	// --- Step 5: Detect platforms + filter to requested list ---
	allDetected := detectAllPlatforms(manifest)

	// If plan.Platforms is non-nil, filter to only those requested.
	// Unknown IDs were already rejected by parsePlanFromFlags.
	targetPlatforms := resolvePlatformTargets(plan.Platforms, allDetected)
	if len(targetPlatforms) == 0 {
		return fmt.Errorf("none of the requested platforms were detected on this system")
	}

	// --- Step 6: Execute install ---
	r := defaultReporter
	return executeInstall(manifest, plan, distDir, targetPlatforms, r)
}
