package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// T1a-3: Failing tests for resolve.go (RED phase)
// ---------------------------------------------------------------------------

// TestResolveMode verifies the precedence rule:
//   marker.mode (cwd) > global config.mode > built-in default "vault"
//
// Cases:
//  1. marker present with in-project + global vault     → in-project
//  2. no marker + global vault                         → vault
//  3. no marker + global in-project                    → in-project
//  4. marker present with vault + global in-project    → vault
func TestResolveMode(t *testing.T) {
	tests := []struct {
		name        string
		markerMode  DocsMode
		markerFound bool
		globalMode  DocsMode
		want        DocsMode
	}{
		{
			name:        "marker in-project overrides global vault",
			markerMode:  ModeInProject,
			markerFound: true,
			globalMode:  ModeVault,
			want:        ModeInProject,
		},
		{
			name:        "no marker falls back to global vault",
			markerMode:  "",
			markerFound: false,
			globalMode:  ModeVault,
			want:        ModeVault,
		},
		{
			name:        "no marker falls back to global in-project",
			markerMode:  "",
			markerFound: false,
			globalMode:  ModeInProject,
			want:        ModeInProject,
		},
		{
			name:        "marker vault overrides global in-project",
			markerMode:  ModeVault,
			markerFound: true,
			globalMode:  ModeInProject,
			want:        ModeVault,
		},
		{
			name:        "invalid marker mode degrades to global mode",
			markerMode:  "bogus",
			markerFound: true,
			globalMode:  ModeInProject,
			want:        ModeInProject,
		},
		{
			name:        "invalid global mode degrades to vault default",
			markerMode:  "",
			markerFound: false,
			globalMode:  "bogus",
			want:        ModeVault,
		},
		{
			name:        "invalid marker and invalid global degrade to vault default",
			markerMode:  "bogus",
			markerFound: true,
			globalMode:  "also-bogus",
			want:        ModeVault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveMode(tt.markerMode, tt.markerFound, tt.globalMode)
			if got != tt.want {
				t.Errorf("resolveMode(%q, %v, %q) = %q, want %q",
					tt.markerMode, tt.markerFound, tt.globalMode, got, tt.want)
			}
		})
	}
}

// TestReadMarker_Present verifies readMarker correctly parses a .doc-agent.json file.
func TestReadMarker_Present(t *testing.T) {
	dir := t.TempDir()
	markerData := map[string]string{"mode": "in-project"}
	data, _ := json.Marshal(markerData)
	if err := os.WriteFile(filepath.Join(dir, ".doc-agent.json"), data, 0644); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	marker, found, err := readMarker(dir)
	if err != nil {
		t.Fatalf("readMarker() error: %v", err)
	}
	if !found {
		t.Fatal("readMarker() found=false, want true")
	}
	if marker.Mode != ModeInProject {
		t.Errorf("marker.Mode = %q, want %q", marker.Mode, ModeInProject)
	}
}

// TestReadMarker_Missing verifies readMarker returns (zero, false, nil) when absent.
func TestReadMarker_Missing(t *testing.T) {
	dir := t.TempDir()

	marker, found, err := readMarker(dir)
	if err != nil {
		t.Fatalf("readMarker() error on missing file: %v", err)
	}
	if found {
		t.Error("readMarker() found=true, want false for missing file")
	}
	if marker.Mode != "" {
		t.Errorf("marker.Mode = %q, want empty for missing file", marker.Mode)
	}
}

// TestReadMarker_Malformed verifies readMarker returns an error on invalid JSON.
func TestReadMarker_Malformed(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".doc-agent.json"), []byte("{bad json"), 0644); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	_, _, err := readMarker(dir)
	if err == nil {
		t.Fatal("readMarker() with malformed JSON should return error, got nil")
	}
}

// TestReadMarker_VaultMode verifies the vault mode is parsed correctly.
func TestReadMarker_VaultMode(t *testing.T) {
	dir := t.TempDir()
	markerData := map[string]string{"mode": "vault"}
	data, _ := json.Marshal(markerData)
	if err := os.WriteFile(filepath.Join(dir, ".doc-agent.json"), data, 0644); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	marker, found, err := readMarker(dir)
	if err != nil {
		t.Fatalf("readMarker() error: %v", err)
	}
	if !found {
		t.Fatal("readMarker() found=false, want true")
	}
	if marker.Mode != ModeVault {
		t.Errorf("marker.Mode = %q, want %q", marker.Mode, ModeVault)
	}
}
