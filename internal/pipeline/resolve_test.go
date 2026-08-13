package pipeline

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveVaultLayoutByNodeDepth(t *testing.T) {
	projectRoot := t.TempDir()
	base := t.TempDir()

	tests := []struct {
		name         string
		node         string
		wantRelDir   string
		wantArtifact string
		wantIndex    string
	}{
		{
			// skills/doc-arch/SKILL.md:23-25 — the system root is flat.
			name:         "system sits at the vault root",
			node:         "acme-hr",
			wantRelDir:   "acme-hr",
			wantArtifact: "acme-hr_prd.md",
			wantIndex:    "acme-hr.md",
		},
		{
			// skills/doc-arch/SKILL.md:27 — modules live under modules/<module>/.
			name:         "module nests under modules",
			node:         "acme-hr/payroll",
			wantRelDir:   filepath.Join("acme-hr", "modules", "payroll"),
			wantArtifact: "payroll_prd.md",
			wantIndex:    "payroll.md",
		},
		{
			name:         "submodule nests one level further",
			node:         "acme-hr/payroll/tax",
			wantRelDir:   filepath.Join("acme-hr", "modules", "payroll", "modules", "tax"),
			wantArtifact: "tax_prd.md",
			wantIndex:    "tax.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := ParseNode(tt.node)
			if err != nil {
				t.Fatalf("ParseNode: %v", err)
			}

			res, err := Resolve(node, Environment{
				ProjectRoot:    projectRoot,
				GlobalMode:     ModeVault,
				GlobalBasePath: base,
			})
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}

			if want := filepath.Join(base, tt.wantRelDir); res.DocsRoot != want {
				t.Errorf("docs root = %q, want %q", res.DocsRoot, want)
			}
			if got := filepath.Base(res.ArtifactPath(PhasePRD, mustLoadBank(t))); got != tt.wantArtifact {
				t.Errorf("prd artifact = %q, want %q", got, tt.wantArtifact)
			}
			if got := filepath.Base(res.IndexPath); got != tt.wantIndex {
				t.Errorf("index = %q, want %q", got, tt.wantIndex)
			}
		})
	}
}

func TestResolveInProjectLayoutDropsTheSystemFolderAndFilePrefix(t *testing.T) {
	projectRoot := t.TempDir()

	tests := []struct {
		name         string
		node         string
		wantRelDir   string
		wantArtifact string
	}{
		{
			// src/templates/path-resolution.md.tmpl:5 — in-project writes
			// docs/doc-agent/_prd.md: no <system> folder and a bare filename.
			name:         "system maps to the docs root itself",
			node:         "acme-hr",
			wantRelDir:   filepath.Join("docs", "doc-agent"),
			wantArtifact: "_prd.md",
		},
		{
			name:         "module gets a plain subfolder with no modules segment",
			node:         "acme-hr/payroll",
			wantRelDir:   filepath.Join("docs", "doc-agent", "payroll"),
			wantArtifact: "_prd.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := ParseNode(tt.node)
			if err != nil {
				t.Fatalf("ParseNode: %v", err)
			}

			res, err := Resolve(node, Environment{
				ProjectRoot: projectRoot,
				GlobalMode:  ModeInProject,
			})
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}

			if want := filepath.Join(projectRoot, tt.wantRelDir); res.DocsRoot != want {
				t.Errorf("docs root = %q, want %q", res.DocsRoot, want)
			}
			if got := filepath.Base(res.ArtifactPath(PhasePRD, mustLoadBank(t))); got != tt.wantArtifact {
				t.Errorf("prd artifact = %q, want %q", got, tt.wantArtifact)
			}
		})
	}
}

func TestResolveModePrecedence(t *testing.T) {
	// src/templates/path-resolution.md.tmpl:1-3 — marker.mode > global.mode > vault.
	tests := []struct {
		name           string
		marker         string
		globalMode     string
		wantMode       string
		wantResolvedBy string
	}{
		{
			name:           "marker wins over global",
			marker:         `{"mode":"in-project"}`,
			globalMode:     ModeVault,
			wantMode:       ModeInProject,
			wantResolvedBy: "marker",
		},
		{
			name:           "global applies when there is no marker",
			globalMode:     ModeInProject,
			wantMode:       ModeInProject,
			wantResolvedBy: "global",
		},
		{
			name:           "vault is the default when nothing is set",
			wantMode:       ModeVault,
			wantResolvedBy: "default",
		},
		{
			name:           "marker without a mode field falls through to global",
			marker:         `{"somethingElse":"value"}`,
			globalMode:     ModeInProject,
			wantMode:       ModeInProject,
			wantResolvedBy: "global",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectRoot := t.TempDir()
			if tt.marker != "" {
				markerPath := filepath.Join(projectRoot, markerFileName)
				if err := os.WriteFile(markerPath, []byte(tt.marker), 0o644); err != nil {
					t.Fatalf("writing marker: %v", err)
				}
			}

			node, err := ParseNode("acme-hr")
			if err != nil {
				t.Fatalf("ParseNode: %v", err)
			}

			res, err := Resolve(node, Environment{
				ProjectRoot:    projectRoot,
				GlobalMode:     tt.globalMode,
				GlobalBasePath: t.TempDir(),
			})
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}

			if res.Mode != tt.wantMode {
				t.Errorf("mode = %q, want %q", res.Mode, tt.wantMode)
			}
			if res.ModeResolvedBy != tt.wantResolvedBy {
				t.Errorf("resolved by = %q, want %q", res.ModeResolvedBy, tt.wantResolvedBy)
			}
		})
	}
}

func TestResolveFailsRatherThanGuess(t *testing.T) {
	node, err := ParseNode("acme-hr")
	if err != nil {
		t.Fatalf("ParseNode: %v", err)
	}

	t.Run("vault mode without a base path cannot be resolved", func(t *testing.T) {
		if _, err := Resolve(node, Environment{
			ProjectRoot: t.TempDir(),
			GlobalMode:  ModeVault,
		}); err == nil {
			t.Fatal("Resolve succeeded with no vault base path, want error")
		}
	})

	t.Run("marker with an unrecognised mode is an error not a fallback", func(t *testing.T) {
		// Falling back here would silently write documentation somewhere the user
		// did not ask for, which is worse than refusing.
		projectRoot := t.TempDir()
		markerPath := filepath.Join(projectRoot, markerFileName)
		if err := os.WriteFile(markerPath, []byte(`{"mode":"somewhere-else"}`), 0o644); err != nil {
			t.Fatalf("writing marker: %v", err)
		}

		if _, err := Resolve(node, Environment{
			ProjectRoot:    projectRoot,
			GlobalMode:     ModeVault,
			GlobalBasePath: t.TempDir(),
		}); err == nil {
			t.Fatal("Resolve succeeded with an invalid marker mode, want error")
		}
	})

	t.Run("unparseable marker is an error", func(t *testing.T) {
		projectRoot := t.TempDir()
		markerPath := filepath.Join(projectRoot, markerFileName)
		if err := os.WriteFile(markerPath, []byte(`{ not json`), 0o644); err != nil {
			t.Fatalf("writing marker: %v", err)
		}

		if _, err := Resolve(node, Environment{
			ProjectRoot:    projectRoot,
			GlobalMode:     ModeVault,
			GlobalBasePath: t.TempDir(),
		}); err == nil {
			t.Fatal("Resolve succeeded with a corrupt marker, want error")
		}
	})
}

func TestStateDirLivesBesideTheArtifacts(t *testing.T) {
	node, err := ParseNode("acme-hr/payroll")
	if err != nil {
		t.Fatalf("ParseNode: %v", err)
	}

	base := t.TempDir()
	res, err := Resolve(node, Environment{
		ProjectRoot:    t.TempDir(),
		GlobalMode:     ModeVault,
		GlobalBasePath: base,
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	wantDir := filepath.Join(res.DocsRoot, stateDirName, "answers")
	got := res.AnswerRecordPath(PhasePRD)
	if filepath.Dir(got) != wantDir {
		t.Errorf("answer record dir = %q, want %q", filepath.Dir(got), wantDir)
	}
	if want := "payroll.prd.json"; filepath.Base(got) != want {
		t.Errorf("answer record file = %q, want %q", filepath.Base(got), want)
	}
}
