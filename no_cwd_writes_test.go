package docagent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestRunHeadlessInstall_DoesNotWriteToCurrentWorkingDirectory(t *testing.T) {
	cwd := t.TempDir()
	home := t.TempDir()
	restoreHome := mockHomeEnv(t, home)
	defer restoreHome()

	before := dirEntries(t, cwd)

	opencodeHome := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(opencodeHome, 0755); err != nil {
		t.Fatalf("create opencode home: %v", err)
	}
	cfgData, _ := json.Marshal(map[string]any{})
	if err := os.WriteFile(filepath.Join(opencodeHome, "opencode.json"), cfgData, 0644); err != nil {
		t.Fatalf("write opencode.json: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	flags := FlagSet{
		Platforms: "opencode",
		DocsMode:  "vault",
		Path:      filepath.ToSlash(filepath.Join(home, "projects")) + "/",
		Yes:       true,
	}

	if err := runHeadlessInstall(flags, ""); err != nil {
		t.Fatalf("runHeadlessInstall: %v", err)
	}

	after := dirEntries(t, cwd)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("current working directory changed: before=%v after=%v", before, after)
	}
}

func dirEntries(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir %s: %v", dir, err)
	}
	result := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}
