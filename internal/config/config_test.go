package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// T1a-1: Failing tests for config.go (RED phase)
// ---------------------------------------------------------------------------

// TestConfigPath verifies the config file path resolves to ~/.doc-agent-ai.json.
func TestConfigPath(t *testing.T) {
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	defer restoreHome()

	got, err := ConfigPath()
	if err != nil {
		t.Fatalf("configPath() error: %v", err)
	}
	want := filepath.Join(tmpHome, ".doc-agent-ai.json")
	if got != want {
		t.Errorf("configPath() = %q, want %q", got, want)
	}
}

// TestLoadConfig_MissingFile verifies that a missing config file returns
// (AppConfig{}, existed=false, nil) — no error, zero-value config (Mode=="").
//
// Contract: a MISSING config signals "no prior state" to callers. Mode=="" lets
// parsePlanFromFlags apply its own resolution order (flags → config → built-in
// default) without an invented "vault" poisoning the PrevMode field. The
// Mode=="" → "vault" normalisation is applied ONLY to an existing parsed config
// with an empty Mode field (old-config compat, config.go loadConfig lines 67-69).
func TestLoadConfig_MissingFile(t *testing.T) {
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	defer restoreHome()

	cfg, existed, err := Load()
	if err != nil {
		t.Fatalf("loadConfig() with missing file returned error: %v", err)
	}
	if existed {
		t.Error("existed should be false for missing file")
	}
	// A missing config must return a zero-value AppConfig (Mode == "").
	// Callers that need a default must apply their own fallback.
	if cfg.Mode != "" {
		t.Errorf("missing-file config.Mode = %q, want %q (zero value)", cfg.Mode, "")
	}
}

// TestLoadConfig_MalformedJSON verifies that malformed JSON returns an error
// and does not panic.
func TestLoadConfig_MalformedJSON(t *testing.T) {
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	defer restoreHome()

	cfgFile := filepath.Join(tmpHome, ".doc-agent-ai.json")
	if err := os.WriteFile(cfgFile, []byte("{this is not json"), 0644); err != nil {
		t.Fatalf("write malformed file: %v", err)
	}

	_, _, err := Load()
	if err == nil {
		t.Fatal("loadConfig() with malformed JSON should return error, got nil")
	}
}

// TestLoadConfig_RoundTrip verifies that saveConfig then loadConfig preserves
// mode, path, and platforms across vault and in-project variants.
func TestLoadConfig_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		cfg  AppConfig
	}{
		{
			name: "vault mode",
			cfg: AppConfig{
				Version:   1,
				Mode:      "vault",
				Path:      "/home/user/docs/",
				Platforms: []string{"opencode", "claude"},
			},
		},
		{
			name: "in-project mode",
			cfg: AppConfig{
				Version:   1,
				Mode:      "in-project",
				Path:      "",
				Platforms: []string{"opencode"},
			},
		},
		{
			name: "no platforms field",
			cfg: AppConfig{
				Version:   1,
				Mode:      "vault",
				Path:      "/tmp/",
				Platforms: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpHome := t.TempDir()
			restoreHome := mockHomeEnv(t, tmpHome)
			defer restoreHome()

			if err := Save(tt.cfg); err != nil {
				t.Fatalf("saveConfig() error: %v", err)
			}

			loaded, existed, err := Load()
			if err != nil {
				t.Fatalf("loadConfig() error: %v", err)
			}
			if !existed {
				t.Error("existed should be true after save")
			}
			if loaded.Mode != tt.cfg.Mode {
				t.Errorf("Mode = %q, want %q", loaded.Mode, tt.cfg.Mode)
			}
			if loaded.Path != tt.cfg.Path {
				t.Errorf("Path = %q, want %q", loaded.Path, tt.cfg.Path)
			}
			if len(loaded.Platforms) != len(tt.cfg.Platforms) {
				t.Errorf("Platforms length = %d, want %d", len(loaded.Platforms), len(tt.cfg.Platforms))
			}
			for i, p := range tt.cfg.Platforms {
				if loaded.Platforms[i] != p {
					t.Errorf("Platforms[%d] = %q, want %q", i, loaded.Platforms[i], p)
				}
			}
		})
	}
}

// TestLoadConfig_VersionTolerance verifies that a missing "version" field in
// the JSON is treated as version 1 (forward-compat: no hard crash).
func TestLoadConfig_VersionTolerance(t *testing.T) {
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	defer restoreHome()

	// Write a config without a version field.
	noVersion := map[string]string{"mode": "vault", "path": "/docs/"}
	data, _ := json.Marshal(noVersion)
	cfgFile := filepath.Join(tmpHome, ".doc-agent-ai.json")
	if err := os.WriteFile(cfgFile, data, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, existed, err := Load()
	if err != nil {
		t.Fatalf("loadConfig() error: %v", err)
	}
	if !existed {
		t.Error("existed should be true")
	}
	// Version absent in JSON => zero value => treated as v1 implicitly.
	if cfg.Mode != "vault" {
		t.Errorf("Mode = %q, want %q", cfg.Mode, "vault")
	}
}

// TestSaveConfig_FilePermissions verifies that saveConfig writes with mode 0644.
func TestSaveConfig_FilePermissions(t *testing.T) {
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	defer restoreHome()

	cfg := AppConfig{Version: 1, Mode: "vault", Path: "/docs/"}
	if err := Save(cfg); err != nil {
		t.Fatalf("saveConfig() error: %v", err)
	}

	cfgFile := filepath.Join(tmpHome, ".doc-agent-ai.json")
	info, err := os.Stat(cfgFile)
	if err != nil {
		t.Fatalf("stat config file: %v", err)
	}
	// On Windows permissions are simplified; only check on POSIX-like systems.
	mode := info.Mode().Perm()
	// Accept 0644 exactly on POSIX; on Windows mode checks are best-effort.
	if mode != 0644 {
		// Windows reports 0666 regardless; the os.PathSeparator check below is
		// what actually skips the assertion there.
		if !filepath.IsAbs(tmpHome) || os.PathSeparator != '\\' {
			t.Errorf("file permissions = %o, want 0644", mode)
		}
	}
}


// TestSaveConfig_TrailingNewline verifies that saveConfig output ends with '\n'.
func TestSaveConfig_TrailingNewline(t *testing.T) {
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	defer restoreHome()

	cfg := AppConfig{Version: 1, Mode: "vault", Path: "/docs/"}
	if err := Save(cfg); err != nil {
		t.Fatalf("saveConfig() error: %v", err)
	}

	cfgFile := filepath.Join(tmpHome, ".doc-agent-ai.json")
	data, err := os.ReadFile(cfgFile)
	if err != nil {
		t.Fatalf("read config file: %v", err)
	}
	if len(data) == 0 || data[len(data)-1] != '\n' {
		t.Errorf("saved config does not end with newline; last byte = %q", data[len(data)-1])
	}
}
