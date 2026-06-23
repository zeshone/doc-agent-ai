package install

import (
	configpkg "github.com/zeshone/doc-agent-ai/internal/config"
	"os"
	"path/filepath"
	"testing"
)

const (
	ModeVault     = configpkg.ModeVault
	ModeInProject = configpkg.ModeInProject
)

func loadConfig() (configpkg.AppConfig, bool, error) { return configpkg.Load() }
func configPath() (string, error)                    { return configpkg.ConfigPath() }

func newPlatformForTest(t *testing.T, id string, homeDir string) Platform {
	t.Helper()
	cfg := PlatformConfig{SkillRoot: homeDir + "/skills", PromptDir: "prompts"}
	switch id {
	case "opencode":
		cfg.CommandDir = "commands"
		return &opencodePlatform{basePlatform{id: id, homeDir: homeDir, cfg: cfg}}
	case "qwen":
		cfg.AgentDir = "agents"
		return &qwenPlatform{basePlatform{id: id, homeDir: homeDir, cfg: cfg}}
	case "copilot":
		cfg.AgentDir = "agents"
		return &copilotPlatform{basePlatform{id: id, homeDir: homeDir, cfg: cfg}}
	case "claude":
		cfg.AgentDir = "agents"
		return &claudePlatform{basePlatform{id: id, homeDir: homeDir, cfg: cfg}}
	case "pi":
		return &piPlatform{basePlatform{id: id, homeDir: homeDir, cfg: cfg}}
	default:
		t.Fatalf("unknown platform: %s", id)
		return nil
	}
}

func mockHomeEnv(t *testing.T, tmpDir string) func() {
	t.Helper()
	restore := make(map[string]string)
	for _, env := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME"} {
		if old, ok := os.LookupEnv(env); ok {
			restore[env] = old
		}
	}
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))
	return func() {
		for k, v := range restore {
			os.Setenv(k, v)
		}
	}
}
