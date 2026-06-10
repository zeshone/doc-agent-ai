package main

import (
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// Conditional install tests for doc-reader (T3-2 RED → GREEN)
// ---------------------------------------------------------------------------

// setupMultiPlatformFixture creates a tmpHome with opencode + claude + pi platforms
// for testing conditional skill install across all skillsDir platforms.
func setupMultiPlatformFixture(t *testing.T) (string, string, DistManifest, []Platform) {
	t.Helper()
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	t.Cleanup(restoreHome)

	distDir := filepath.Join(t.TempDir(), "dist")
	if err := generate(distDir); err != nil {
		t.Fatalf("generate dist: %v", err)
	}
	manifest, err := readManifestFrom(distDir)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	// opencode platform
	opencodeHome := filepath.Join(tmpHome, ".config", "opencode")
	if err := os.MkdirAll(opencodeHome, 0755); err != nil {
		t.Fatalf("create opencode home: %v", err)
	}
	writeEmptyJSON(t, filepath.Join(opencodeHome, "opencode.json"))
	opencodeP := newPlatformForTest(t, "opencode", opencodeHome)

	// claude platform
	claudeHome := filepath.Join(tmpHome, ".claude")
	if err := os.MkdirAll(claudeHome, 0755); err != nil {
		t.Fatalf("create claude home: %v", err)
	}
	claudeP := newPlatformForTest(t, "claude", claudeHome)

	return tmpHome, distDir, manifest, []Platform{opencodeP, claudeP}
}

// writeEmptyJSON writes an empty JSON object to path.
func writeEmptyJSON(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
		t.Fatalf("writeEmptyJSON %s: %v", path, err)
	}
}

// TestInstall_InProject_InstallsDocReaderOnAllSkillsDirPlatforms verifies that
// executing an install in in-project mode copies doc-reader to every platform
// with a non-empty SkillsDir.
func TestInstall_InProject_InstallsDocReaderOnAllSkillsDirPlatforms(t *testing.T) {
	_, distDir, manifest, platforms := setupMultiPlatformFixture(t)

	for _, plat := range platforms {
		r := newBufferReporter()
		if err := installToPlatformWithReporter(manifest, plat, "", distDir, r, string(ModeInProject)); err != nil {
			t.Fatalf("installToPlatformWithReporter [%s] in-project: %v", plat.ID(), err)
		}

		skillsDir := plat.SkillsDir()
		docReaderDir := filepath.Join(skillsDir, "doc-reader")
		if _, err := os.Stat(docReaderDir); os.IsNotExist(err) {
			t.Errorf("[%s] doc-reader skill dir should exist in in-project mode; path: %s", plat.ID(), docReaderDir)
		}
	}
}

// TestInstall_Vault_DoesNotInstallDocReader verifies that vault mode installs
// do NOT copy doc-reader to any platform's skillsDir.
func TestInstall_Vault_DoesNotInstallDocReader(t *testing.T) {
	_, distDir, manifest, platforms := setupMultiPlatformFixture(t)

	for _, plat := range platforms {
		r := newBufferReporter()
		if err := installToPlatformWithReporter(manifest, plat, "/base/path/", distDir, r, string(ModeVault)); err != nil {
			t.Fatalf("installToPlatformWithReporter [%s] vault: %v", plat.ID(), err)
		}

		skillsDir := plat.SkillsDir()
		docReaderDir := filepath.Join(skillsDir, "doc-reader")
		if _, err := os.Stat(docReaderDir); !os.IsNotExist(err) {
			t.Errorf("[%s] doc-reader skill dir must NOT exist in vault mode; path: %s", plat.ID(), docReaderDir)
		}
	}
}

// TestInstall_Vault_DefaultMode_DoesNotInstallDocReader verifies that the
// default mode (no explicit mode argument = vault) also skips doc-reader.
func TestInstall_Vault_DefaultMode_DoesNotInstallDocReader(t *testing.T) {
	_, distDir, manifest, platforms := setupMultiPlatformFixture(t)

	plat := platforms[0] // opencode
	if err := installToPlatform(manifest, plat, "/base/path/", distDir); err != nil {
		t.Fatalf("installToPlatform (default mode): %v", err)
	}

	skillsDir := plat.SkillsDir()
	docReaderDir := filepath.Join(skillsDir, "doc-reader")
	if _, err := os.Stat(docReaderDir); !os.IsNotExist(err) {
		t.Errorf("doc-reader must not be installed in default (vault) mode; path: %s", docReaderDir)
	}
}
