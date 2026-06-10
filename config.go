package main

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
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".doc-agent-ai.json"), nil
}

// loadConfig reads and parses ~/.doc-agent-ai.json.
// Returns:
//   - cfg: the parsed config (with safe defaults applied)
//   - existed: true if the file was found and successfully parsed
//   - err: non-nil only on I/O error or JSON parse failure; a missing file is
//     NOT an error — it returns (defaults, false, nil)
func loadConfig() (AppConfig, bool, error) {
	path, err := configPath()
	if err != nil {
		return AppConfig{Mode: "vault"}, false, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return AppConfig{Mode: "vault"}, false, nil
		}
		return AppConfig{Mode: "vault"}, false, err
	}

	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return AppConfig{Mode: "vault"}, false, err
	}

	// Apply safe defaults for fields that may be absent in old configs.
	if cfg.Mode == "" {
		cfg.Mode = "vault"
	}

	return cfg, true, nil
}

// saveConfig writes cfg to ~/.doc-agent-ai.json with file permissions 0644.
// Output is indented JSON followed by a trailing newline (matches repo convention).
func saveConfig(cfg AppConfig) error {
	path, err := configPath()
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
