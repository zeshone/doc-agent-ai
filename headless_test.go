package docagent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Headless flags path tests (T2a-3)
// ---------------------------------------------------------------------------

// TestParseInstallFlags_NoFlags verifies that no install flags returns an empty
// FlagSet and hasInstallFlags returns false.
func TestParseInstallFlags_NoFlags(t *testing.T) {
	args := []string{"install"}
	flags, remaining := parseInstallFlags(args)

	if hasInstallFlags(flags) {
		t.Error("expected hasInstallFlags=false for args with no install flags")
	}
	if len(remaining) != 1 || remaining[0] != "install" {
		t.Errorf("unexpected remaining args: %v", remaining)
	}
}

// TestParseInstallFlags_AllFlags verifies all four install flags are parsed.
func TestParseInstallFlags_AllFlags(t *testing.T) {
	args := []string{
		"install",
		"--platforms", "opencode,claude",
		"--docs-mode", "vault",
		"--path", "/my/docs",
		"--yes",
	}
	flags, remaining := parseInstallFlags(args)

	if flags.Platforms != "opencode,claude" {
		t.Errorf("Platforms = %q; want %q", flags.Platforms, "opencode,claude")
	}
	if flags.DocsMode != "vault" {
		t.Errorf("DocsMode = %q; want %q", flags.DocsMode, "vault")
	}
	if flags.Path != "/my/docs" {
		t.Errorf("Path = %q; want %q", flags.Path, "/my/docs")
	}
	if !flags.Yes {
		t.Error("Yes should be true")
	}
	if hasInstallFlags(flags) == false {
		t.Error("hasInstallFlags must be true when any flag is set")
	}
	// "install" subcommand must remain in remaining.
	if len(remaining) != 1 || remaining[0] != "install" {
		t.Errorf("unexpected remaining args: %v", remaining)
	}
}

// TestParseInstallFlags_YesAlone verifies --yes alone sets hasInstallFlags=true.
func TestParseInstallFlags_YesAlone(t *testing.T) {
	args := []string{"install", "--yes"}
	flags, _ := parseInstallFlags(args)
	if !flags.Yes {
		t.Error("Yes should be true")
	}
	if !hasInstallFlags(flags) {
		t.Error("hasInstallFlags must be true when --yes is set")
	}
}

// TestParseInstallFlags_PlatformsOnly verifies --platforms alone sets hasInstallFlags.
func TestParseInstallFlags_PlatformsOnly(t *testing.T) {
	args := []string{"install", "--platforms", "opencode"}
	flags, _ := parseInstallFlags(args)
	if flags.Platforms != "opencode" {
		t.Errorf("Platforms = %q; want %q", flags.Platforms, "opencode")
	}
	if !hasInstallFlags(flags) {
		t.Error("hasInstallFlags must be true when --platforms is set")
	}
}

// TestParseInstallFlags_DocsModeOnly verifies --docs-mode alone sets hasInstallFlags.
func TestParseInstallFlags_DocsModeOnly(t *testing.T) {
	args := []string{"install", "--docs-mode", "in-project"}
	flags, _ := parseInstallFlags(args)
	if flags.DocsMode != "in-project" {
		t.Errorf("DocsMode = %q; want %q", flags.DocsMode, "in-project")
	}
	if !hasInstallFlags(flags) {
		t.Error("hasInstallFlags must be true when --docs-mode is set")
	}
}

// TestParseInstallFlags_PathOnly verifies --path alone sets hasInstallFlags.
func TestParseInstallFlags_PathOnly(t *testing.T) {
	args := []string{"install", "--path", "/some/path"}
	flags, _ := parseInstallFlags(args)
	if flags.Path != "/some/path" {
		t.Errorf("Path = %q; want %q", flags.Path, "/some/path")
	}
	if !hasInstallFlags(flags) {
		t.Error("hasInstallFlags must be true when --path is set")
	}
}

// TestRunHeadlessInstall_VaultSuccess verifies that runHeadlessInstall completes
// a full install into a temp HOME when valid vault flags are provided.
func TestRunHeadlessInstall_VaultSuccess(t *testing.T) {
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	defer restoreHome()

	distDir := filepath.Join(t.TempDir(), "dist")
	if err := generate(distDir); err != nil {
		t.Fatalf("generate dist: %v", err)
	}

	// Set up opencode platform.
	opencodeHome := filepath.Join(tmpHome, ".config", "opencode")
	if err := os.MkdirAll(opencodeHome, 0755); err != nil {
		t.Fatalf("create opencode home: %v", err)
	}
	cfgData, _ := json.Marshal(map[string]any{})
	if err := os.WriteFile(filepath.Join(opencodeHome, "opencode.json"), cfgData, 0644); err != nil {
		t.Fatalf("write opencode.json: %v", err)
	}

	basePath := filepath.ToSlash(filepath.Join(tmpHome, "projects")) + "/"
	flags := FlagSet{
		Platforms: "opencode",
		DocsMode:  "vault",
		Path:      basePath,
		Yes:       true,
	}

	// Provide a custom distDir override to avoid touching the real dist/.
	if err := runHeadlessInstall(flags, distDir); err != nil {
		t.Fatalf("runHeadlessInstall: %v", err)
	}

	// Config must have been written.
	cfgPath, _ := configPath()
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Fatalf("AppConfig not written after runHeadlessInstall: %s", cfgPath)
	}
	cfg, _, _ := loadConfig()
	if cfg.Mode != "vault" {
		t.Errorf("config.Mode = %q; want %q", cfg.Mode, "vault")
	}
}

// TestRunHeadlessInstall_InProjectSuccess verifies in-project mode headless install.
func TestRunHeadlessInstall_InProjectSuccess(t *testing.T) {
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	defer restoreHome()

	distDir := filepath.Join(t.TempDir(), "dist")
	if err := generate(distDir); err != nil {
		t.Fatalf("generate dist: %v", err)
	}

	opencodeHome := filepath.Join(tmpHome, ".config", "opencode")
	if err := os.MkdirAll(opencodeHome, 0755); err != nil {
		t.Fatalf("create opencode home: %v", err)
	}
	cfgData, _ := json.Marshal(map[string]any{})
	if err := os.WriteFile(filepath.Join(opencodeHome, "opencode.json"), cfgData, 0644); err != nil {
		t.Fatalf("write opencode.json: %v", err)
	}

	flags := FlagSet{
		Platforms: "opencode",
		DocsMode:  "in-project",
		// No path needed for in-project mode.
		Yes: true,
	}

	if err := runHeadlessInstall(flags, distDir); err != nil {
		t.Fatalf("runHeadlessInstall in-project: %v", err)
	}

	cfg, _, _ := loadConfig()
	if cfg.Mode != "in-project" {
		t.Errorf("config.Mode = %q; want %q", cfg.Mode, "in-project")
	}
}

// TestRunHeadlessInstall_VaultMissingPath verifies an error is returned when
// vault mode is used without a path (neither flag nor config default).
func TestRunHeadlessInstall_VaultMissingPath(t *testing.T) {
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	defer restoreHome()

	distDir := filepath.Join(t.TempDir(), "dist")
	if err := generate(distDir); err != nil {
		t.Fatalf("generate dist: %v", err)
	}

	flags := FlagSet{
		Platforms: "opencode",
		DocsMode:  "vault",
		// Path intentionally missing.
		Yes: true,
	}

	err := runHeadlessInstall(flags, distDir)
	if err == nil {
		t.Error("expected error for vault mode with no path; got nil")
	}
	if !strings.Contains(err.Error(), "path") && !strings.Contains(err.Error(), "vault") {
		t.Errorf("expected error to mention path/vault; got: %v", err)
	}
}

// TestRunHeadlessInstall_InvalidPlatform verifies an error is returned for an
// unknown platform ID.
func TestRunHeadlessInstall_InvalidPlatform(t *testing.T) {
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	defer restoreHome()

	distDir := filepath.Join(t.TempDir(), "dist")
	if err := generate(distDir); err != nil {
		t.Fatalf("generate dist: %v", err)
	}

	basePath := filepath.ToSlash(filepath.Join(tmpHome, "projects")) + "/"
	flags := FlagSet{
		Platforms: "unknown-platform",
		DocsMode:  "vault",
		Path:      basePath,
		Yes:       true,
	}

	err := runHeadlessInstall(flags, distDir)
	if err == nil {
		t.Error("expected error for unknown platform; got nil")
	}
}

// TestDecisionOrder_FlagsOverTTY verifies that when install flags are present,
// hasInstallFlags returns true, meaning the flags path is chosen over TTY.
// This is the "flags > TTY" unit test from the spec.
func TestDecisionOrder_FlagsOverTTY(t *testing.T) {
	// Simulate having --docs-mode set — any install flag triggers headless path.
	flags := FlagSet{DocsMode: "vault"}
	if !hasInstallFlags(flags) {
		t.Error("hasInstallFlags must be true when --docs-mode is provided, regardless of TTY state")
	}

	// Simulate no flags — falls through to TTY check.
	empty := FlagSet{}
	if hasInstallFlags(empty) {
		t.Error("hasInstallFlags must be false when no install flags are set")
	}
}
