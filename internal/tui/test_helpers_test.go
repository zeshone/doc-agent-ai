package tui

import (
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
	plat := installpkg.NewPlatformForTest(id, homeDir)
	if plat == nil {
		t.Fatalf("unknown platform: %s", id)
	}
	return plat
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
