package install

import (
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// Conditional install tests for doc-reader (T3-2 RED → GREEN)
// ---------------------------------------------------------------------------

// setupMultiPlatformFixture creates a tmpHome with opencode + claude platforms
// for testing conditional skill install across skillsDir platforms.
func setupMultiPlatformFixture(t *testing.T) (string, Bundle, DistManifest, []Platform) {
	t.Helper()
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	t.Cleanup(restoreHome)

	bundle := testBundle()
	manifest := bundle.Manifest

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

	return tmpHome, bundle, manifest, []Platform{opencodeP, claudeP}
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
	_, bundle, manifest, platforms := setupMultiPlatformFixture(t)

	for _, plat := range platforms {
		r := newBufferReporter()
		if err := InstallToPlatformWithReporter(manifest, bundle, plat, "", r, string(ModeInProject)); err != nil {
			t.Fatalf("InstallToPlatformWithReporter [%s] in-project: %v", plat.ID(), err)
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
	_, bundle, manifest, platforms := setupMultiPlatformFixture(t)

	for _, plat := range platforms {
		r := newBufferReporter()
		if err := InstallToPlatformWithReporter(manifest, bundle, plat, "/base/path/", r, string(ModeVault)); err != nil {
			t.Fatalf("InstallToPlatformWithReporter [%s] vault: %v", plat.ID(), err)
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
	_, bundle, manifest, platforms := setupMultiPlatformFixture(t)

	plat := platforms[0] // opencode
	r := newBufferReporter()
	if err := InstallToPlatformWithReporter(manifest, bundle, plat, "/base/path/", r); err != nil {
		t.Fatalf("InstallToPlatformWithReporter (default mode): %v", err)
	}

	skillsDir := plat.SkillsDir()
	docReaderDir := filepath.Join(skillsDir, "doc-reader")
	if _, err := os.Stat(docReaderDir); !os.IsNotExist(err) {
		t.Errorf("doc-reader must not be installed in default (vault) mode; path: %s", docReaderDir)
	}
}
