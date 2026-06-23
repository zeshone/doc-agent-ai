package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// executeInstall tests (T2a-2)
// ---------------------------------------------------------------------------

// setupExecuteInstallFixture creates the minimal filesystem state needed for
// executeInstall tests: temp HOME, an opencode platform, generated dist.
// Returns: tmpHome, bundle, manifest, opencodePlatform.
func setupExecuteInstallFixture(t *testing.T) (string, Bundle, DistManifest, Platform) {
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
	bundle, err := bundleFromDistDir(manifest, distDir)
	if err != nil {
		t.Fatalf("bundleFromDistDir: %v", err)
	}

	opencodeHome := filepath.Join(tmpHome, ".config", "opencode")
	if err := os.MkdirAll(opencodeHome, 0755); err != nil {
		t.Fatalf("create opencode home: %v", err)
	}
	cfgData, _ := json.Marshal(map[string]any{})
	if err := os.WriteFile(filepath.Join(opencodeHome, "opencode.json"), cfgData, 0644); err != nil {
		t.Fatalf("write opencode.json: %v", err)
	}
	plat := newPlatformForTest(t, "opencode", opencodeHome)
	return tmpHome, bundle, manifest, plat
}

// TestExecuteInstall_VaultMode_InstallsAndPersistsConfig verifies that
// executeInstall runs a vault-mode install and writes AppConfig afterward.
func TestExecuteInstall_VaultMode_InstallsAndPersistsConfig(t *testing.T) {
	tmpHome, bundle, _, plat := setupExecuteInstallFixture(t)

	basePath := filepath.ToSlash(filepath.Join(tmpHome, "projects")) + "/"
	plan := InstallPlan{
		Platforms: []string{"opencode"},
		Mode:      ModeVault,
		BasePath:  basePath,
		PrevMode:  ModeVault,
	}

	r := newBufferReporter()
	if err := ExecuteInstall(bundle, plan, []Platform{plat}, r); err != nil {
		t.Fatalf("executeInstall: %v", err)
	}

	// Config must be written.
	cfgPath, _ := configPath()
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Fatalf("AppConfig not written after executeInstall: %s", cfgPath)
	}

	// Config must reflect chosen mode and base path.
	cfg, existed, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if !existed {
		t.Fatal("config file does not exist after executeInstall")
	}
	if cfg.Mode != string(ModeVault) {
		t.Errorf("config.Mode = %q; want %q", cfg.Mode, ModeVault)
	}
	if cfg.Path != basePath {
		t.Errorf("config.Path = %q; want %q", cfg.Path, basePath)
	}
}

// TestExecuteInstall_InProjectMode_InstallsAndPersistsConfig verifies
// in-project mode sets config.Mode and leaves Path empty.
func TestExecuteInstall_InProjectMode_InstallsAndPersistsConfig(t *testing.T) {
	_, bundle, _, plat := setupExecuteInstallFixture(t)

	plan := InstallPlan{
		Platforms: []string{"opencode"},
		Mode:      ModeInProject,
		BasePath:  "", // in-project has no base path
		PrevMode:  ModeVault,
	}

	r := newBufferReporter()
	if err := ExecuteInstall(bundle, plan, []Platform{plat}, r); err != nil {
		t.Fatalf("executeInstall: %v", err)
	}

	cfg, existed, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if !existed {
		t.Fatal("config file does not exist after executeInstall (in-project)")
	}
	if cfg.Mode != string(ModeInProject) {
		t.Errorf("config.Mode = %q; want %q", cfg.Mode, ModeInProject)
	}
}

// TestExecuteInstall_PlatformListNilUsesDetected verifies that when plan.Platforms
// is nil, all provided platforms are installed (nil → use provided list as "all").
func TestExecuteInstall_PlatformListNilUsesDetected(t *testing.T) {
	tmpHome, bundle, _, plat := setupExecuteInstallFixture(t)

	basePath := filepath.ToSlash(filepath.Join(tmpHome, "projects")) + "/"
	plan := InstallPlan{
		Platforms: nil, // nil → install all provided platforms
		Mode:      ModeVault,
		BasePath:  basePath,
		PrevMode:  ModeVault,
	}

	r := newBufferReporter()
	// Provide opencode as the "all detected" list.
	if err := ExecuteInstall(bundle, plan, []Platform{plat}, r); err != nil {
		t.Fatalf("executeInstall with nil Platforms: %v", err)
	}

	// Reporter must have captured output proving install ran.
	if !strings.Contains(r.buf.String(), "opencode") &&
		!strings.Contains(r.buf.String(), "skill:") &&
		!strings.Contains(r.buf.String(), "prompt:") {
		t.Errorf("expected install output for opencode, got:\n%s", r.buf.String())
	}
}

// TestExecuteInstall_PassesModeToInstallToPlatform verifies that the resolved
// plan.Mode is passed through to file substitution (no __DOC_AGENT_GLOBAL_MODE__
// leaks after install).
func TestExecuteInstall_PassesModeToInstallToPlatform(t *testing.T) {
	tmpHome, bundle, _, plat := setupExecuteInstallFixture(t)

	basePath := filepath.ToSlash(filepath.Join(tmpHome, "projects")) + "/"
	plan := InstallPlan{
		Platforms: []string{"opencode"},
		Mode:      ModeVault,
		BasePath:  basePath,
		PrevMode:  ModeVault,
	}

	r := newBufferReporter()
	if err := ExecuteInstall(bundle, plan, []Platform{plat}, r); err != nil {
		t.Fatalf("executeInstall: %v", err)
	}

	// Walk installed prompts and commands — no raw token must survive.
	const barePrefix = "__DOC_AGENT_"
	var leaks []string
	walkForLeaks := func(dir string) {
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			p := filepath.Join(dir, e.Name())
			c, _ := os.ReadFile(p)
			if strings.Contains(string(c), barePrefix) {
				leaks = append(leaks, p)
			}
		}
	}
	walkForLeaks(plat.PromptsDir())
	walkForLeaks(filepath.Join(plat.HomeDir(), "commands"))
	if len(leaks) > 0 {
		t.Errorf("executeInstall left %d unresolved token(s): %v", len(leaks), leaks)
	}
}

// TestExecuteInstall_ModeSwitchHook_NoopWhenSameMode verifies that the mode-switch
// hook seam fires without error when prevMode == plan.Mode (no mode change).
func TestExecuteInstall_ModeSwitchHook_NoopWhenSameMode(t *testing.T) {
	tmpHome, bundle, _, plat := setupExecuteInstallFixture(t)

	basePath := filepath.ToSlash(filepath.Join(tmpHome, "projects")) + "/"
	plan := InstallPlan{
		Platforms: []string{"opencode"},
		Mode:      ModeVault,
		BasePath:  basePath,
		PrevMode:  ModeVault, // same mode — no switch
	}

	r := newBufferReporter()
	if err := ExecuteInstall(bundle, plan, []Platform{plat}, r); err != nil {
		t.Fatalf("executeInstall (no mode switch): %v", err)
	}
	// No mode-switch notice should appear — assert against the exact strings
	// runModeSwitchHook emits, or this guard is a false negative.
	if strings.Contains(r.buf.String(), "Mode changed") ||
		strings.Contains(r.buf.String(), "not automatically migrated") {
		t.Errorf("unexpected mode-switch notice when mode did not change; output:\n%s", r.buf.String())
	}
}

// TestExecuteInstall_ModeSwitchHook_NoticeWhenModeChanges verifies that the
// mode-switch hook emits a config notice when prevMode != plan.Mode.
func TestExecuteInstall_ModeSwitchHook_NoticeWhenModeChanges(t *testing.T) {
	tmpHome, bundle, _, plat := setupExecuteInstallFixture(t)

	basePath := filepath.ToSlash(filepath.Join(tmpHome, "projects")) + "/"
	plan := InstallPlan{
		Platforms: []string{"opencode"},
		Mode:      ModeVault,
		BasePath:  basePath,
		PrevMode:  ModeInProject, // mode switch: in-project → vault
	}

	r := newBufferReporter()
	if err := ExecuteInstall(bundle, plan, []Platform{plat}, r); err != nil {
		t.Fatalf("executeInstall (mode switch): %v", err)
	}
	// A non-migration notice must be emitted.
	out := r.buf.String()
	if !strings.Contains(out, "not migrated") && !strings.Contains(out, "not automatically migrated") {
		t.Errorf("expected non-migration notice on mode switch; output:\n%s", out)
	}
}

// TestExecuteInstall_FailurePath_NoConfigWritten verifies that when a platform
// install fails, executeInstall returns an error and does NOT write AppConfig.
//
// Guard for execute_install.go:41-43: the early return on install error must
// fire BEFORE the saveConfig call at execute_install.go:58.
func TestExecuteInstall_FailurePath_NoConfigWritten(t *testing.T) {
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
	bundle, err := bundleFromDistDir(manifest, distDir)
	if err != nil {
		t.Fatalf("bundleFromDistDir: %v", err)
	}

	// Create a platform whose SkillsDir() path is blocked by a file so that
	// copyDir → ensureDir → os.MkdirAll fails with "not a directory".
	opencodeHome := filepath.Join(tmpHome, ".config", "opencode")
	if err := os.MkdirAll(opencodeHome, 0755); err != nil {
		t.Fatalf("create opencode home: %v", err)
	}
	// Write a regular FILE at the path where the skills dir would be created.
	// os.MkdirAll cannot create a directory when a path component is a file.
	skillsBlocker := filepath.Join(opencodeHome, "skills")
	if err := os.WriteFile(skillsBlocker, []byte("blocker"), 0644); err != nil {
		t.Fatalf("write skills blocker file: %v", err)
	}

	plat := newPlatformForTest(t, "opencode", opencodeHome)

	basePath := filepath.ToSlash(filepath.Join(tmpHome, "projects")) + "/"
	plan := InstallPlan{
		Platforms: []string{"opencode"},
		Mode:      ModeVault,
		BasePath:  basePath,
		PrevMode:  ModeVault,
	}

	r := newBufferReporter()
	installErr := ExecuteInstall(bundle, plan, []Platform{plat}, r)
	if installErr == nil {
		t.Fatal("expected executeInstall to fail when platform install is blocked; got nil error")
	}

	// Config MUST NOT have been written.
	cfgPath, _ := configPath()
	if _, statErr := os.Stat(cfgPath); !os.IsNotExist(statErr) {
		t.Errorf("AppConfig must not be written after a failed install; file exists at %s", cfgPath)
	}
}

// TestExecuteInstall_ConfigPersistsSelectedPlatforms verifies that the
// platforms list from the plan is saved in AppConfig after a successful install.
func TestExecuteInstall_ConfigPersistsSelectedPlatforms(t *testing.T) {
	tmpHome, bundle, _, plat := setupExecuteInstallFixture(t)

	basePath := filepath.ToSlash(filepath.Join(tmpHome, "projects")) + "/"
	plan := InstallPlan{
		Platforms: []string{"opencode"},
		Mode:      ModeVault,
		BasePath:  basePath,
		PrevMode:  ModeVault,
	}

	r := newBufferReporter()
	if err := ExecuteInstall(bundle, plan, []Platform{plat}, r); err != nil {
		t.Fatalf("executeInstall: %v", err)
	}

	cfg, _, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if len(cfg.Platforms) == 0 {
		t.Error("config.Platforms not persisted after executeInstall")
	}
	found := false
	for _, id := range cfg.Platforms {
		if id == "opencode" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'opencode' in config.Platforms; got %v", cfg.Platforms)
	}
}
