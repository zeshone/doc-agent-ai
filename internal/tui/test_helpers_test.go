package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	installpkg "github.com/zeshone/doc-agent-ai/internal/install"
)

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

func newPlatformForTest(t *testing.T, id string, homeDir string) Platform {
	t.Helper()
	switch id {
	case "opencode":
		p, err := installpkg.NewOpenCodePlatform(installpkg.PlatformConfig{SkillRoot: homeDir + "/skills", PromptDir: "prompts", CommandDir: "commands"})
		if err != nil {
			t.Fatalf("newOpenCodePlatform: %v", err)
		}
		return p
	case "claude":
		return &tuiTestPlatform{id: id, homeDir: homeDir, skillsDir: filepath.Join(homeDir, "skills"), promptsDir: filepath.Join(homeDir, "prompts", "doc"), agentsDir: filepath.Join(homeDir, "agents")}
	default:
		t.Fatalf("unknown platform: %s", id)
		return nil
	}
}

type tuiTestPlatform struct {
	id        string
	homeDir   string
	skillsDir string
	promptsDir string
	agentsDir string
}

func (p *tuiTestPlatform) ID() string      { return p.id }
func (p *tuiTestPlatform) HomeDir() string { return p.homeDir }
func (p *tuiTestPlatform) Detect() bool    { return true }
func (p *tuiTestPlatform) SkillsDir() string { return p.skillsDir }
func (p *tuiTestPlatform) PromptsDir() string { return p.promptsDir }
func (p *tuiTestPlatform) AgentsDir() string  { return p.agentsDir }
func (p *tuiTestPlatform) GetAgentIDs(manifest installpkg.DistManifest, installed map[string]bool) ([]string, error) {
	var ids []string
	for _, role := range manifest.Roles {
		path := filepath.Join(p.agentsDir, role.ID+".md")
		if _, err := os.Stat(path); err == nil {
			ids = append(ids, role.ID)
			if installed != nil {
				installed[role.ID] = true
			}
		}
	}
	return ids, nil
}
func (p *tuiTestPlatform) GetPromptIDs(manifest installpkg.DistManifest, installed map[string]bool) ([]string, error) {
	var ids []string
	for _, role := range manifest.Roles {
		path := filepath.Join(p.promptsDir, role.ID+".md")
		if _, err := os.Stat(path); err == nil {
			ids = append(ids, role.ID)
			if installed != nil {
				installed[role.ID] = true
			}
		}
	}
	return ids, nil
}
func (p *tuiTestPlatform) GetSkillIDs(manifest installpkg.DistManifest) []string {
	var ids []string
	for _, skill := range manifest.Skills {
		if _, err := os.Stat(filepath.Join(p.skillsDir, skill)); err == nil {
			ids = append(ids, skill)
		}
	}
	return ids
}
func (p *tuiTestPlatform) GetCommandIDs(manifest installpkg.DistManifest) ([]string, error) { return nil, nil }
func (p *tuiTestPlatform) PatchConfig(manifest installpkg.DistManifest, basePath string, roleIDs []string) error { return nil }
func (p *tuiTestPlatform) RemoveConfig(manifest installpkg.DistManifest, roleIDs []string) error { return nil }
func (p *tuiTestPlatform) SkillRegistryTrigger() string { return "" }
func (p *tuiTestPlatform) WriteSkillRegistry(basePath string) error { return nil }

func writeEmptyJSON(t *testing.T, path string) {
	t.Helper()
	data, _ := json.Marshal(map[string]any{})
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write json %s: %v", path, err)
	}
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
