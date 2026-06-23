package docagent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

func setupExecuteInstallFixture(t *testing.T) (string, Bundle, DistManifest, Platform) {
	t.Helper()
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	t.Cleanup(restoreHome)

	bundle, err := BuildBundle()
	if err != nil {
		t.Fatalf("BuildBundle: %v", err)
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
	return tmpHome, bundle, bundle.Manifest, plat
}

type bufferReporter struct{ buf bytes.Buffer }

func newBufferReporter() *bufferReporter { return &bufferReporter{} }
func (b *bufferReporter) Ok(msg string)     { fmt.Fprintf(&b.buf, "  ✔ %s\n", msg) }
func (b *bufferReporter) Warn(msg string)   { fmt.Fprintf(&b.buf, "  ⚠  %s\n", msg) }
func (b *bufferReporter) ErrOut(msg string) { fmt.Fprintf(&b.buf, "  ✖ %s\n", msg) }
func (b *bufferReporter) Info(msg string)   { fmt.Fprintf(&b.buf, "  → %s\n", msg) }
func (b *bufferReporter) Dim(msg string)    { fmt.Fprintf(&b.buf, "  %s\n", msg) }
func (b *bufferReporter) Head(msg string)   { fmt.Fprintf(&b.buf, "\n  %s\n", msg) }
