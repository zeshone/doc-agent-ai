package config

import (
	"os"
	"path/filepath"
	"testing"
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
