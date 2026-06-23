package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// ---------------------------------------------------------------------------
// AppConfig — persistent global configuration at ~/.doc-agent-ai.json
// ---------------------------------------------------------------------------

// AppConfig holds the user's persistent installer settings.
// The file uses additive JSON: unknown fields are silently ignored so older
// binaries can read configs written by newer versions without crashing.
type AppConfig struct {
	// Version is a schema version integer; absent or 0 is treated as v1.
	Version int `json:"version,omitempty"`
	// Mode is the global doc storage mode: "vault" or "in-project".
	// Absent in file defaults to "vault" (preserves pre-v4 behaviour).
	Mode string `json:"mode,omitempty"`
	// Path is the vault base path (absolute, trailing slash normalized).
	// Ignored and may be empty when Mode == "in-project".
	Path string `json:"path,omitempty"`
	// Platforms is the last selected platform list, used to pre-fill TUI on
	// reinstall. Absence means "prompt/all".
	Platforms []string `json:"platforms,omitempty"`
}

// configPath returns the absolute path to the persistent config file:
// $HOME/.doc-agent-ai.json on all OSes.
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".doc-agent-ai.json"), nil
}

// loadConfig reads and parses ~/.doc-agent-ai.json.
// Returns:
//   - cfg: the parsed config (with safe defaults applied for EXISTING configs)
//   - existed: true if the file was found and successfully parsed
//   - err: non-nil only on I/O error or JSON parse failure; a missing file is
//     NOT an error — it returns (AppConfig{}, false, nil)
//
// Contract: a MISSING or UNREADABLE config returns the zero value AppConfig{}
// (Mode=="") to signal "no prior state". Callers (e.g. parsePlanFromFlags) apply
// their own resolution order and must not treat Mode=="" as a prior "vault"
// install. The Mode=="" → "vault" normalisation is applied ONLY to an existing
// parsed config whose Mode field is absent (old-config backward compat).
func Load() (AppConfig, bool, error) {
	path, err := ConfigPath()
	if err != nil {
		return AppConfig{}, false, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return AppConfig{}, false, nil
		}
		return AppConfig{}, false, err
	}

	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return AppConfig{}, false, err
	}

	// Apply safe defaults for fields that may be absent in old configs.
	// This normalisation applies only to an existing file — it preserves
	// backward compat with pre-v4 configs that omit the "mode" field.
	if cfg.Mode == "" {
		cfg.Mode = "vault"
	}

	return cfg, true, nil
}

// saveConfig writes cfg to ~/.doc-agent-ai.json with file permissions 0644.
// Output is indented JSON followed by a trailing newline (matches repo convention).
func Save(cfg AppConfig) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	return os.WriteFile(path, data, 0644)
}
