package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	configpkg "github.com/zeshone/doc-agent-ai/internal/config"
)

func TestFreshInstall_NoModeChangedNotice(t *testing.T) {
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	defer restoreHome()

	bundle := testBundle()

	opencodeHome := filepath.Join(tmpHome, ".config", "opencode")
	if err := os.MkdirAll(opencodeHome, 0755); err != nil {
		t.Fatalf("create opencode home: %v", err)
	}
	cfgData, _ := json.Marshal(map[string]any{})
	if err := os.WriteFile(filepath.Join(opencodeHome, "opencode.json"), cfgData, 0644); err != nil {
		t.Fatalf("write opencode.json: %v", err)
	}
	plat := newPlatformForTest(t, "opencode", opencodeHome)

	cfg, _, _ := configpkg.Load()
	flags := configpkg.FlagSet{Platforms: "opencode", DocsMode: "in-project", Yes: true}
	plan, err := configpkg.ParsePlanFromFlags(flags, cfg)
	if err != nil {
		t.Fatalf("ParsePlanFromFlags: %v", err)
	}

	r := newBufferReporter()
	if err := ExecuteInstall(bundle, plan, []Platform{plat}, r); err != nil {
		t.Fatalf("executeInstall: %v", err)
	}

	out := r.buf.String()
	if strings.Contains(out, "Mode changed") {
		t.Errorf("fresh install must NOT emit 'Mode changed' notice; got output:\n%s", out)
	}
}
