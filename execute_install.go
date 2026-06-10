package main

import (
	"fmt"
)

// ---------------------------------------------------------------------------
// executeInstall — thin orchestrator consuming InstallPlan
// ---------------------------------------------------------------------------

// executeInstall is the engine-side orchestrator that drives a full install
// from a resolved InstallPlan. Both the headless flags path and (in slice 2b)
// the Bubbletea TUI will call executeInstall after building an InstallPlan.
//
// Responsibilities:
//  1. Platform resolution — if plan.Platforms is nil, treat allPlatforms as
//     the full list (headless "install to all detected").
//  2. Mode wiring — passes plan.Mode (as string) to installToPlatformWithReporter
//     so the __DOC_AGENT_GLOBAL_MODE__ placeholder is resolved correctly.
//  3. Config persistence — writes AppConfig after a successful install so
//     subsequent runs pre-fill mode/path/platforms.
//  4. Mode-switch hook — when plan.PrevMode != plan.Mode, emits the
//     non-migration notice. Slice 3 will plug doc-reader removal here; for
//     now the seam exists and the notice is the only side-effect.
//
// The engine never reads AppConfig directly — it receives resolved values via
// InstallPlan. This keeps the engine pure and unit-testable without filesystem
// state.
func executeInstall(manifest DistManifest, plan InstallPlan, distDir string, allPlatforms []Platform, r Reporter) error {
	// --- Step 1: Resolve platform list ---
	// nil Platforms → install to all provided platforms (headless "all" behaviour).
	targets := resolvePlatformTargets(plan.Platforms, allPlatforms)
	if len(targets) == 0 {
		if len(plan.Platforms) > 0 {
			return fmt.Errorf("none of the requested platforms were detected on this system")
		}
		return fmt.Errorf("no platforms detected on this system")
	}

	// --- Step 2: Install to each target platform (with overwrite gate) ---
	// When a platform already has agents installed, it requires explicit overwrite
	// consent. Consent is given either via plan.Overwrite[platID]=true (TUI/wizard)
	// or via plan.Yes=true (headless --yes flag). Absent consent causes an error
	// so the user is informed to either re-run with --yes or use the TUI.
	globalMode := string(plan.Mode)
	for _, plat := range targets {
		platID := plat.ID()
		existing := checkAlreadyInstalled(manifest, plat)
		if len(existing) > 0 {
			// Check consent: plan.Yes (headless --yes) overrides everything.
			if !plan.Yes && !plan.Overwrite[platID] {
				return fmt.Errorf("%s already has agents installed — pass --yes to overwrite, or use the TUI to confirm per-platform", platformDisplayName(platID))
			}
			if plan.Yes {
				r.Info("Overwriting existing installation for " + platformDisplayName(platID) + " (--yes provided).")
			}
		}

		r.Head("Installing for " + platformDisplayName(platID) + "...")
		if err := installToPlatformWithReporter(manifest, plat, plan.BasePath, distDir, r, globalMode); err != nil {
			return fmt.Errorf("install to %s: %w", platID, err)
		}
	}

	// --- Step 3: Persist config ---
	// Build an updated AppConfig reflecting the completed install.
	installedIDs := make([]string, 0, len(targets))
	for _, p := range targets {
		installedIDs = append(installedIDs, p.ID())
	}
	newCfg := AppConfig{
		Version:   1,
		Mode:      globalMode,
		Path:      plan.BasePath,
		Platforms: installedIDs,
	}
	if err := saveConfig(newCfg); err != nil {
		// Config write failure is non-fatal: the install succeeded; we just
		// cannot pre-fill defaults next time. Emit a warning and continue.
		r.Warn("could not save config: " + err.Error())
	}

	// --- Step 4: Mode-switch hook (seam for slice 3 doc-reader cleanup) ---
	if plan.PrevMode != "" && plan.PrevMode != plan.Mode {
		runModeSwitchHook(plan, r)
	}

	return nil
}

// resolvePlatformTargets returns the Platform instances to install to.
// If requestedIDs is nil or empty, all platforms in allPlatforms are returned.
// If requestedIDs is non-nil, only those IDs present in allPlatforms are returned.
func resolvePlatformTargets(requestedIDs []string, allPlatforms []Platform) []Platform {
	if len(requestedIDs) == 0 {
		// nil / empty = "all detected"
		return allPlatforms
	}

	// Build a lookup map from the available platforms.
	byID := make(map[string]Platform, len(allPlatforms))
	for _, p := range allPlatforms {
		byID[p.ID()] = p
	}

	var result []Platform
	for _, id := range requestedIDs {
		if p, ok := byID[id]; ok {
			result = append(result, p)
		}
		// Unknown IDs are silently skipped here; parsePlanFromFlags validates them
		// at parse time so by the time we reach executeInstall they are known-good.
	}
	return result
}

// runModeSwitchHook is the seam for mode-switch side-effects.
// Currently it emits the non-migration notice required by spec F1.
//
// Slice 3 will extend this function to call sweepDocReaderIfLeavingInProject
// when switching from in-project → vault. The function signature is intentionally
// kept stable so slice 3 can add behaviour without touching executeInstall.
func runModeSwitchHook(plan InstallPlan, r Reporter) {
	// Always emit the non-migration notice (spec F1, mode-switch cleanup notice).
	r.Info("Mode changed from " + string(plan.PrevMode) + " to " + string(plan.Mode) + ".")
	r.Warn("Existing documentation files are not automatically migrated.")
	r.Info("See the path-resolution preamble in your installed prompts for the new layout.")

	// TODO(slice-3): if plan.PrevMode == ModeInProject && plan.Mode == ModeVault {
	//     sweepDocReaderIfLeavingInProject(plan, r)
	// }
}
