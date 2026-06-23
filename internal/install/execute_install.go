package install

import (
	"fmt"
	"os"
	"path/filepath"

	configpkg "github.com/zeshone/doc-agent-ai/internal/config"
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
func ExecuteInstall(bundle Bundle, plan configpkg.InstallPlan, allPlatforms []Platform, r Reporter) error {
	manifest := bundle.Manifest
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
		if err := InstallToPlatformWithReporter(manifest, bundle, plat, plan.BasePath, r, globalMode); err != nil {
			return fmt.Errorf("install to %s: %w", platID, err)
		}
	}

	// --- Step 3: Persist config ---
	// Build an updated AppConfig reflecting the completed install.
	installedIDs := make([]string, 0, len(targets))
	for _, p := range targets {
		installedIDs = append(installedIDs, p.ID())
	}
	newCfg := configpkg.AppConfig{
		Version:   1,
		Mode:      globalMode,
		Path:      plan.BasePath,
		Platforms: installedIDs,
	}
	if err := configpkg.Save(newCfg); err != nil {
		// Config write failure is non-fatal: the install succeeded; we just
		// cannot pre-fill defaults next time. Emit a warning and continue.
		r.Warn("could not save config: " + err.Error())
	}

	// --- Step 4: Mode-switch hook (seam for slice 3 doc-reader cleanup) ---
	if plan.PrevMode != "" && plan.PrevMode != plan.Mode {
		runModeSwitchHookWithPlatforms(plan, targets, r)
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

// runModeSwitchHook is kept for backward compatibility with existing tests that
// call it without a platform list. It delegates to runModeSwitchHookWithPlatforms
// with a nil platform list (notice only, no sweep).
func runModeSwitchHook(plan configpkg.InstallPlan, r Reporter) {
	runModeSwitchHookWithPlatforms(plan, nil, r)
}

// runModeSwitchHookWithPlatforms is the canonical mode-switch side-effect handler.
// It always emits the non-migration notice (spec F1) and, when switching from
// in-project → vault, sweeps the doc-reader skill from all provided platforms.
//
// platforms is the resolved install target list (from executeInstall). When nil
// (legacy / test call path), the sweep is skipped.
func runModeSwitchHookWithPlatforms(plan configpkg.InstallPlan, platforms []Platform, r Reporter) {
	// Always emit the non-migration notice (spec F1, mode-switch cleanup notice).
	r.Info("Mode changed from " + string(plan.PrevMode) + " to " + string(plan.Mode) + ".")
	r.Warn("Existing documentation files are not automatically migrated.")
	r.Info("See the path-resolution preamble in your installed prompts for the new layout.")

	// When switching from in-project → vault, remove the doc-reader skill from
	// all platforms that have a skillsDir. The sweep is idempotent: absent dirs
	// are silently skipped (reusing the removeDirIfExists pattern from uninstall).
	if plan.PrevMode == configpkg.ModeInProject && plan.Mode == configpkg.ModeVault && len(platforms) > 0 {
		sweepDocReaderIfLeavingInProject(platforms, r)
	}
}

// sweepDocReaderIfLeavingInProject removes the doc-reader skill directory from
// every platform that has a non-empty SkillsDir. This is called during a
// mode-switch from in-project → vault so that stale conditional-skill files
// are cleaned up automatically. Idempotent: platforms where doc-reader was
// never installed (or was already removed) are silently skipped.
func sweepDocReaderIfLeavingInProject(platforms []Platform, r Reporter) {
	for _, plat := range platforms {
		skillsDir := plat.SkillsDir()
		if skillsDir == "" {
			continue
		}
		docReaderDir := filepath.Join(skillsDir, "doc-reader")
		if _, err := os.Stat(docReaderDir); os.IsNotExist(err) {
			// Already absent — idempotent skip.
			continue
		}
		if err := os.RemoveAll(docReaderDir); err != nil {
			r.Warn("could not remove doc-reader from " + plat.ID() + ": " + err.Error())
			continue
		}
		r.Ok("removed conditional skill doc-reader from " + plat.ID())
	}
}
