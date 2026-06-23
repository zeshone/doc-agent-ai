package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallToPlatformWithReporter_UsesBundleFiles(t *testing.T) {
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	defer restoreHome()

	opencodeHome := filepath.Join(tmpHome, ".config", "opencode")
	if err := os.MkdirAll(opencodeHome, 0755); err != nil {
		t.Fatalf("create opencode home: %v", err)
	}
	cfgData, _ := json.Marshal(map[string]any{})
	if err := os.WriteFile(filepath.Join(opencodeHome, "opencode.json"), cfgData, 0644); err != nil {
		t.Fatalf("write opencode.json: %v", err)
	}

	bundle := Bundle{
		Manifest: DistManifest{
			PlaceholderBasePath: "__DOC_AGENT_BASE_PATH__/",
			Skills:              []string{"doc-arch"},
			Roles: []DistRole{{
				ID: "doc-arch",
				PromptFiles: PromptFileMap{
					OpenCode: "prompts/opencode/doc-arch.md",
				},
			}},
			Commands: []DistCommand{{
				ID:   "doc-arch",
				File: "commands/doc-arch.md",
			}},
		},
		Files: map[string][]byte{
			"skills/doc-arch/SKILL.md":           []byte("skill body\n"),
			"prompts/opencode/doc-arch.md":       []byte("path=__DOC_AGENT_BASE_PATH__/\nmode=__DOC_AGENT_GLOBAL_MODE__\n"),
			"commands/doc-arch.md":               []byte("command=__DOC_AGENT_GLOBAL_BASE__\n"),
		},
	}

	basePath := filepath.ToSlash(filepath.Join(tmpHome, "vault")) + "/"
	plat := newPlatformForTest(t, "opencode", opencodeHome)
	r := newBufferReporter()

	if err := InstallToPlatformWithReporter(bundle.Manifest, bundle, plat, basePath, r, string(ModeVault)); err != nil {
		t.Fatalf("InstallToPlatformWithReporter: %v", err)
	}

	skillPath := filepath.Join(plat.SkillsDir(), "doc-arch", "SKILL.md")
	if data, err := os.ReadFile(skillPath); err != nil {
		t.Fatalf("read installed skill: %v", err)
	} else if string(data) != "skill body\n" {
		t.Fatalf("skill body mismatch: %q", string(data))
	}

	promptPath := filepath.Join(plat.PromptsDir(), "doc-arch.md")
	promptData, err := os.ReadFile(promptPath)
	if err != nil {
		t.Fatalf("read installed prompt: %v", err)
	}
	promptText := string(promptData)
	if strings.Contains(promptText, "__DOC_AGENT_") {
		t.Fatalf("prompt still contains placeholders: %q", promptText)
	}
	if !strings.Contains(promptText, "path="+basePath) {
		t.Fatalf("prompt missing base path substitution: %q", promptText)
	}
	if !strings.Contains(promptText, "mode=vault") {
		t.Fatalf("prompt missing mode substitution: %q", promptText)
	}

	commandPath := filepath.Join(plat.HomeDir(), "commands", "doc-arch.md")
	commandData, err := os.ReadFile(commandPath)
	if err != nil {
		t.Fatalf("read installed command: %v", err)
	}
	if got := string(commandData); !strings.Contains(got, strings.TrimSuffix(basePath, "/")) {
		t.Fatalf("command missing global base substitution: %q", got)
	}
}
