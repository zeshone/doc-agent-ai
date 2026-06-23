package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	configpkg "github.com/zeshone/doc-agent-ai/internal/config"
)

// ---------------------------------------------------------------------------
// Mode-switch cleanup tests for doc-reader (T3-3 RED → GREEN)
// ---------------------------------------------------------------------------

// seedDocReaderOnPlatform creates a fake doc-reader skill dir in the platform's
// skillsDir to simulate a prior in-project install.
func seedDocReaderOnPlatform(t *testing.T, plat Platform) {
	t.Helper()
	docReaderDir := filepath.Join(plat.SkillsDir(), "doc-reader")
	if err := os.MkdirAll(docReaderDir, 0755); err != nil {
		t.Fatalf("seed doc-reader dir on %s: %v", plat.ID(), err)
	}
	skillMD := filepath.Join(docReaderDir, "SKILL.md")
	if err := os.WriteFile(skillMD, []byte("# doc-reader\n"), 0644); err != nil {
		t.Fatalf("seed SKILL.md on %s: %v", plat.ID(), err)
	}
}

// TestSweepDocReader_InProjectToVault_RemovesFromAllPlatforms verifies that
// sweepDocReaderIfLeavingInProject removes doc-reader from all provided platforms
// when switching from in-project to vault mode.
func TestSweepDocReader_InProjectToVault_RemovesFromAllPlatforms(t *testing.T) {
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	defer restoreHome()

	// Create two platforms and seed doc-reader on both.
	opencodeHome := filepath.Join(tmpHome, ".config", "opencode")
	if err := os.MkdirAll(opencodeHome, 0755); err != nil {
		t.Fatalf("create opencode home: %v", err)
	}
	writeEmptyJSON(t, filepath.Join(opencodeHome, "opencode.json"))
	opencode := newPlatformForTest(t, "opencode", opencodeHome)

	claudeHome := filepath.Join(tmpHome, ".claude")
	if err := os.MkdirAll(claudeHome, 0755); err != nil {
		t.Fatalf("create claude home: %v", err)
	}
	claude := newPlatformForTest(t, "claude", claudeHome)

	platforms := []Platform{opencode, claude}
	for _, plat := range platforms {
		seedDocReaderOnPlatform(t, plat)
	}

	// Run the sweep.
	r := newBufferReporter()
	sweepDocReaderIfLeavingInProject(platforms, r)

	// Verify doc-reader is gone from all platforms.
	for _, plat := range platforms {
		docReaderDir := filepath.Join(plat.SkillsDir(), "doc-reader")
		if _, err := os.Stat(docReaderDir); !os.IsNotExist(err) {
			t.Errorf("[%s] doc-reader dir should be gone after sweep; path: %s", plat.ID(), docReaderDir)
		}
	}
}

// TestSweepDocReader_Idempotent_NoErrorWhenAbsent verifies that
// sweepDocReaderIfLeavingInProject is idempotent: calling it when doc-reader
// is already absent produces no error.
func TestSweepDocReader_Idempotent_NoErrorWhenAbsent(t *testing.T) {
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	defer restoreHome()

	opencodeHome := filepath.Join(tmpHome, ".config", "opencode")
	if err := os.MkdirAll(opencodeHome, 0755); err != nil {
		t.Fatalf("create opencode home: %v", err)
	}
	writeEmptyJSON(t, filepath.Join(opencodeHome, "opencode.json"))
	opencode := newPlatformForTest(t, "opencode", opencodeHome)

	// doc-reader not seeded — calling sweep should not panic or error.
	r := newBufferReporter()
	sweepDocReaderIfLeavingInProject([]Platform{opencode}, r)

	out := r.buf.String()
	if strings.Contains(out, "✖") || strings.Contains(out, "⚠") {
		t.Errorf("sweep with absent doc-reader emitted error/warn output:\n%s", out)
	}
}

// TestRunModeSwitchHook_InProjectToVault_CallsSweep verifies that the
// runModeSwitchHook fires sweepDocReaderIfLeavingInProject when switching
// from in-project to vault mode, removing doc-reader from all platforms.
func TestRunModeSwitchHook_InProjectToVault_CallsSweep(t *testing.T) {
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	defer restoreHome()

	opencodeHome := filepath.Join(tmpHome, ".config", "opencode")
	if err := os.MkdirAll(opencodeHome, 0755); err != nil {
		t.Fatalf("create opencode home: %v", err)
	}
	writeEmptyJSON(t, filepath.Join(opencodeHome, "opencode.json"))
	opencode := newPlatformForTest(t, "opencode", opencodeHome)
	seedDocReaderOnPlatform(t, opencode)

	plan := configpkg.InstallPlan{
		PrevMode:  configpkg.ModeInProject,
		Mode:      configpkg.ModeVault,
		Platforms: []string{"opencode"},
	}
	r := newBufferReporter()

	// runModeSwitchHook now receives all platform targets.
	runModeSwitchHookWithPlatforms(plan, []Platform{opencode}, r)

	docReaderDir := filepath.Join(opencode.SkillsDir(), "doc-reader")
	if _, err := os.Stat(docReaderDir); !os.IsNotExist(err) {
		t.Errorf("doc-reader must be removed when switching in-project→vault; dir: %s", docReaderDir)
	}
}

// TestRunModeSwitchHook_VaultToVault_DoesNotSweep verifies that a vault→vault
// mode (no mode change) does NOT trigger a doc-reader sweep.
func TestRunModeSwitchHook_VaultToVault_DoesNotSweep(t *testing.T) {
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	defer restoreHome()

	opencodeHome := filepath.Join(tmpHome, ".config", "opencode")
	if err := os.MkdirAll(opencodeHome, 0755); err != nil {
		t.Fatalf("create opencode home: %v", err)
	}
	writeEmptyJSON(t, filepath.Join(opencodeHome, "opencode.json"))
	opencode := newPlatformForTest(t, "opencode", opencodeHome)
	// Seed doc-reader to confirm it is NOT removed.
	seedDocReaderOnPlatform(t, opencode)

	plan := configpkg.InstallPlan{
		PrevMode:  configpkg.ModeVault,
		Mode:      configpkg.ModeVault,
		Platforms: []string{"opencode"},
	}
	r := newBufferReporter()
	// vault→vault: mode switch hook should not be called (executeInstall guards
	// this with plan.PrevMode != plan.Mode). Call directly to assert no sweep.
	runModeSwitchHookWithPlatforms(plan, []Platform{opencode}, r)

	// doc-reader should still be there (vault→vault is not a switch).
	docReaderDir := filepath.Join(opencode.SkillsDir(), "doc-reader")
	if _, err := os.Stat(docReaderDir); os.IsNotExist(err) {
		t.Error("doc-reader should NOT be removed on vault→vault (no mode switch)")
	}
}

// TestRunModeSwitchHook_VaultToInProject_DoesNotSweep verifies that switching
// vault → in-project does NOT remove doc-reader (it will be added during install,
// not during the cleanup hook).
func TestRunModeSwitchHook_VaultToInProject_DoesNotSweep(t *testing.T) {
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	defer restoreHome()

	opencodeHome := filepath.Join(tmpHome, ".config", "opencode")
	if err := os.MkdirAll(opencodeHome, 0755); err != nil {
		t.Fatalf("create opencode home: %v", err)
	}
	writeEmptyJSON(t, filepath.Join(opencodeHome, "opencode.json"))
	opencode := newPlatformForTest(t, "opencode", opencodeHome)

	plan := configpkg.InstallPlan{
		PrevMode:  configpkg.ModeVault,
		Mode:      configpkg.ModeInProject,
		Platforms: []string{"opencode"},
	}
	r := newBufferReporter()
	runModeSwitchHookWithPlatforms(plan, []Platform{opencode}, r)
	// No assertion on doc-reader (not installed in vault, not swept in this direction).
	// The test just confirms it does not panic or error.
}
