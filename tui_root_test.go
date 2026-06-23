package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRootModel_StartsOnHome(t *testing.T) {
	m := newRootModelForTest(testBundle())
	if m.screen != screenHome {
		t.Fatalf("screen = %v, want %v", m.screen, screenHome)
	}
	if m.menuCursor != 0 {
		t.Fatalf("menuCursor = %d, want 0", m.menuCursor)
	}
}

func TestRootModel_HomeCursorWraps(t *testing.T) {
	m := newRootModelForTest(testBundle())
	m = updateRootModel(t, m, tea.KeyMsg{Type: tea.KeyUp})
	if m.menuCursor != 2 {
		t.Fatalf("menuCursor after up wrap = %d, want 2", m.menuCursor)
	}
	m = updateRootModel(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if m.menuCursor != 0 {
		t.Fatalf("menuCursor after down wrap = %d, want 0", m.menuCursor)
	}
}

func TestRootModel_EnterOnQuitReturnsQuit(t *testing.T) {
	m := newRootModelForTest(testBundle())
	m.menuCursor = 2
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected tea.Quit command")
	}
}

func TestRootModel_EnterOnInstallTransitionsToInstall(t *testing.T) {
	home := t.TempDir()
	restoreHome := mockHomeEnv(t, home)
	defer restoreHome()

	opencodeHome := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(opencodeHome, 0755); err != nil {
		t.Fatalf("create opencode home: %v", err)
	}
	cfgData, _ := json.Marshal(map[string]any{})
	if err := os.WriteFile(filepath.Join(opencodeHome, "opencode.json"), cfgData, 0644); err != nil {
		t.Fatalf("write opencode.json: %v", err)
	}

	bundle := Bundle{Manifest: DistManifest{Platforms: PlatformManifest{OpenCode: PlatformConfig{SkillRoot: "~/.config/opencode/skills", PromptDir: "prompts", CommandDir: "commands"}}}, Files: map[string][]byte{}}
	m := newRootModelForTest(bundle)

	m = updateRootModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != screenInstall {
		t.Fatalf("screen = %v, want %v", m.screen, screenInstall)
	}
	if m.install == nil {
		t.Fatal("install model not initialized")
	}
}

func TestRootModel_DoneKeyReturnsHome(t *testing.T) {
	m := newRootModelForTest(testBundle())
	m.screen = screenInstall
	install := newInstallModelForTest(AppConfig{}, false, []Platform{})
	install.step = stepDone
	m.install = &install

	m = updateRootModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != screenHome {
		t.Fatalf("screen = %v, want %v", m.screen, screenHome)
	}
	if m.install != nil {
		t.Fatal("install model should be cleared after returning home")
	}
}

func TestGolden_HomeScreen(t *testing.T) {
	m := newRootModelForTest(testBundle())
	view := m.View()
	assertGolden(t, view)
	if !strings.Contains(view, "Install") {
		t.Fatalf("home view should contain Install option; got:\n%s", view)
	}
}

func newRootModelForTest(bundle Bundle) RootModel {
	m := RootModel{screen: screenHome, bundle: bundle, styles: NoColor(), width: 80, height: 24}
	return m
}

func updateRootModel(t *testing.T, m RootModel, msg tea.Msg) RootModel {
	t.Helper()
	next, _ := m.Update(msg)
	updated, ok := next.(RootModel)
	if !ok {
		t.Fatalf("Update returned %T, want RootModel", next)
	}
	return updated
}
