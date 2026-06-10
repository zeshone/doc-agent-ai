package main

import (
	"testing"
)

// ---------------------------------------------------------------------------
// T1a-5: Failing tests for plan.go (RED phase)
// ---------------------------------------------------------------------------

// TestParsePlan_PlatformsCSV verifies that a comma-separated platforms string
// is split into the expected slice.
func TestParsePlan_PlatformsCSV(t *testing.T) {
	flags := FlagSet{Platforms: "opencode,claude"}
	cfg := AppConfig{Mode: "vault", Path: "/docs/"}
	plan, err := parsePlanFromFlags(flags, cfg)
	if err != nil {
		t.Fatalf("parsePlanFromFlags() error: %v", err)
	}
	if len(plan.Platforms) != 2 {
		t.Fatalf("Platforms length = %d, want 2", len(plan.Platforms))
	}
	if plan.Platforms[0] != "opencode" {
		t.Errorf("Platforms[0] = %q, want %q", plan.Platforms[0], "opencode")
	}
	if plan.Platforms[1] != "claude" {
		t.Errorf("Platforms[1] = %q, want %q", plan.Platforms[1], "claude")
	}
}

// TestParsePlan_InvalidPlatform verifies that an unknown platform ID causes
// parsePlanFromFlags to return an error.
func TestParsePlan_InvalidPlatform(t *testing.T) {
	flags := FlagSet{Platforms: "opencode,unknown-plat"}
	cfg := AppConfig{Mode: "vault", Path: "/docs/"}
	_, err := parsePlanFromFlags(flags, cfg)
	if err == nil {
		t.Fatal("expected error for unknown platform, got nil")
	}
}

// TestParsePlan_DocsModeValidation verifies that an invalid docs-mode value
// causes parsePlanFromFlags to return an error.
func TestParsePlan_DocsModeValidation(t *testing.T) {
	flags := FlagSet{
		Platforms: "opencode",
		DocsMode:  "invalid-mode",
		Path:      "/docs/",
	}
	cfg := AppConfig{Mode: "vault"}
	_, err := parsePlanFromFlags(flags, cfg)
	if err == nil {
		t.Fatal("expected error for invalid docs-mode, got nil")
	}
}

// TestParsePlan_InProjectDropsPathRequirement verifies that in-project mode
// does not require a --path flag (BasePath stays empty).
func TestParsePlan_InProjectDropsPathRequirement(t *testing.T) {
	flags := FlagSet{
		Platforms: "opencode",
		DocsMode:  "in-project",
		// No Path provided
	}
	cfg := AppConfig{Mode: "in-project"}
	plan, err := parsePlanFromFlags(flags, cfg)
	if err != nil {
		t.Fatalf("parsePlanFromFlags() error: %v", err)
	}
	if plan.Mode != ModeInProject {
		t.Errorf("Mode = %q, want %q", plan.Mode, ModeInProject)
	}
	if plan.BasePath != "" {
		t.Errorf("BasePath = %q, want empty for in-project mode", plan.BasePath)
	}
}

// TestParsePlan_VaultRequiresPath verifies that vault mode with no --path and
// no config default causes parsePlanFromFlags to return an error.
func TestParsePlan_VaultRequiresPath(t *testing.T) {
	flags := FlagSet{
		Platforms: "opencode",
		DocsMode:  "vault",
		// No Path flag
	}
	cfg := AppConfig{Mode: "vault", Path: ""} // no config default either
	_, err := parsePlanFromFlags(flags, cfg)
	if err == nil {
		t.Fatal("expected error: vault mode requires a path when no config default exists")
	}
}

// TestParsePlan_YesFlag verifies that YesFlag=true is reflected on the plan.
func TestParsePlan_YesFlag(t *testing.T) {
	flags := FlagSet{
		Platforms: "opencode",
		DocsMode:  "vault",
		Path:      "/docs/",
		Yes:       true,
	}
	cfg := AppConfig{Mode: "vault"}
	plan, err := parsePlanFromFlags(flags, cfg)
	if err != nil {
		t.Fatalf("parsePlanFromFlags() error: %v", err)
	}
	if !plan.Yes {
		t.Error("plan.Yes should be true when --yes flag is set")
	}
}

// TestParsePlan_ConfigDefaults verifies that when no flags are provided, the
// plan is pre-filled from the AppConfig.
func TestParsePlan_ConfigDefaults(t *testing.T) {
	flags := FlagSet{} // no flags
	cfg := AppConfig{
		Version:   1,
		Mode:      "vault",
		Path:      "/home/user/docs/",
		Platforms: []string{"opencode", "claude"},
	}
	plan, err := parsePlanFromFlags(flags, cfg)
	if err != nil {
		t.Fatalf("parsePlanFromFlags() error: %v", err)
	}
	if plan.Mode != ModeVault {
		t.Errorf("Mode = %q, want %q", plan.Mode, ModeVault)
	}
	if plan.BasePath != cfg.Path {
		t.Errorf("BasePath = %q, want %q", plan.BasePath, cfg.Path)
	}
	if len(plan.Platforms) != 2 || plan.Platforms[0] != "opencode" || plan.Platforms[1] != "claude" {
		t.Errorf("Platforms = %v, want config default [opencode claude]", plan.Platforms)
	}
}

// TestParsePlan_FlagOverridesConfig verifies that explicit flags take
// precedence over the AppConfig defaults.
func TestParsePlan_FlagOverridesConfig(t *testing.T) {
	flags := FlagSet{
		Platforms: "opencode",
		DocsMode:  "in-project",
		Path:      "", // in-project: no path needed
	}
	cfg := AppConfig{
		Mode: "vault",
		Path: "/old/vault/path/",
	}
	plan, err := parsePlanFromFlags(flags, cfg)
	if err != nil {
		t.Fatalf("parsePlanFromFlags() error: %v", err)
	}
	if plan.Mode != ModeInProject {
		t.Errorf("Mode = %q, want %q", plan.Mode, ModeInProject)
	}
	// Flag overrides config path — in-project has no base path.
	if plan.BasePath != "" {
		t.Errorf("BasePath = %q, want empty (flag in-project mode overrides config vault path)", plan.BasePath)
	}
}
