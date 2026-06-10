package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Reporter seam tests (T2a-1)
// ---------------------------------------------------------------------------

// TestBufferReporter_CapturesOk verifies that bufferReporter captures ok messages.
func TestBufferReporter_CapturesOk(t *testing.T) {
	r := newBufferReporter()
	r.Ok("skill installed")
	got := r.buf.String()
	if !strings.Contains(got, "skill installed") {
		t.Errorf("expected 'skill installed' in captured output, got: %q", got)
	}
}

// TestBufferReporter_CapturesAllMethods verifies all six Reporter methods produce output.
func TestBufferReporter_CapturesAllMethods(t *testing.T) {
	tests := []struct {
		name   string
		call   func(r *bufferReporter)
		expect string
	}{
		{"Ok", func(r *bufferReporter) { r.Ok("msg-ok") }, "msg-ok"},
		{"Warn", func(r *bufferReporter) { r.Warn("msg-warn") }, "msg-warn"},
		{"ErrOut", func(r *bufferReporter) { r.ErrOut("msg-err") }, "msg-err"},
		{"Info", func(r *bufferReporter) { r.Info("msg-info") }, "msg-info"},
		{"Dim", func(r *bufferReporter) { r.Dim("msg-dim") }, "msg-dim"},
		{"Head", func(r *bufferReporter) { r.Head("msg-head") }, "msg-head"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newBufferReporter()
			tt.call(r)
			if !strings.Contains(r.buf.String(), tt.expect) {
				t.Errorf("Reporter.%s did not capture %q; got: %q", tt.name, tt.expect, r.buf.String())
			}
		})
	}
}

// TestStdoutReporter_Implements verifies that stdoutReporter satisfies the Reporter interface.
func TestStdoutReporter_Implements(t *testing.T) {
	var _ Reporter = &stdoutReporter{}
}

// TestBufferReporter_Implements verifies that bufferReporter satisfies the Reporter interface.
func TestBufferReporter_Implements(t *testing.T) {
	var _ Reporter = &bufferReporter{}
}

// TestInstallToPlatformWithReporter_CapturesOutput verifies that
// installToPlatformWithReporter routes install output through the supplied
// Reporter rather than writing directly to stdout.
func TestInstallToPlatformWithReporter_CapturesOutput(t *testing.T) {
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	defer restoreHome()

	distDir := filepath.Join(t.TempDir(), "dist")
	if err := generate(distDir); err != nil {
		t.Fatalf("generate dist: %v", err)
	}
	manifest, err := readManifestFrom(distDir)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	opencodeHome := filepath.Join(tmpHome, ".config", "opencode")
	if err := os.MkdirAll(opencodeHome, 0755); err != nil {
		t.Fatalf("create opencode dir: %v", err)
	}
	cfgData, _ := json.Marshal(map[string]any{})
	if err := os.WriteFile(filepath.Join(opencodeHome, "opencode.json"), cfgData, 0644); err != nil {
		t.Fatalf("write opencode.json: %v", err)
	}
	plat := newPlatformForTest(t, "opencode", opencodeHome)
	basePath := filepath.ToSlash(filepath.Join(tmpHome, "projects")) + "/"

	r := newBufferReporter()
	if err := installToPlatformWithReporter(manifest, plat, basePath, distDir, r, "vault"); err != nil {
		t.Fatalf("installToPlatformWithReporter: %v", err)
	}

	// The reporter must have captured some output (skills, prompts, commands).
	if r.buf.Len() == 0 {
		t.Error("reporter captured no output — expected install messages")
	}
	// Must mention at least one installed artifact.
	out := r.buf.String()
	if !strings.Contains(out, "skill:") && !strings.Contains(out, "prompt:") && !strings.Contains(out, "command:") {
		t.Errorf("reporter output contains no artifact lines; got:\n%s", out)
	}
}

// TestInstallToPlatform_BackwardCompatible verifies the original 5-arg signature
// still compiles and functions correctly (backward-compat wrapper).
func TestInstallToPlatform_BackwardCompatible(t *testing.T) {
	tmpHome := t.TempDir()
	restoreHome := mockHomeEnv(t, tmpHome)
	defer restoreHome()

	distDir := filepath.Join(t.TempDir(), "dist")
	if err := generate(distDir); err != nil {
		t.Fatalf("generate dist: %v", err)
	}
	manifest, err := readManifestFrom(distDir)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	opencodeHome := filepath.Join(tmpHome, ".config", "opencode")
	if err := os.MkdirAll(opencodeHome, 0755); err != nil {
		t.Fatalf("create opencode dir: %v", err)
	}
	cfgData, _ := json.Marshal(map[string]any{})
	if err := os.WriteFile(filepath.Join(opencodeHome, "opencode.json"), cfgData, 0644); err != nil {
		t.Fatalf("write opencode.json: %v", err)
	}
	plat := newPlatformForTest(t, "opencode", opencodeHome)
	basePath := filepath.ToSlash(filepath.Join(tmpHome, "projects")) + "/"

	// Original call signature must still compile and succeed.
	if err := installToPlatform(manifest, plat, basePath, distDir, "vault"); err != nil {
		t.Fatalf("installToPlatform (backward-compat): %v", err)
	}
}
