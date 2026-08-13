package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandUserPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home directory available: %v", err)
	}

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			// The trap this exists for: stored verbatim, resolved without error,
			// and a directory literally named "~" appears beside the cwd.
			name: "tilde slash expands to home",
			in:   "~/src/Obsidian/DevZeshOne",
			want: filepath.Join(home, "src", "Obsidian", "DevZeshOne"),
		},
		{
			name: "bare tilde is the home directory",
			in:   "~",
			want: home,
		},
		{
			name: "surrounding whitespace is trimmed before expanding",
			in:   "  ~/vault  ",
			want: filepath.Join(home, "vault"),
		},
		{
			name: "an absolute path is untouched",
			in:   "/home/someone/vault",
			want: "/home/someone/vault",
		},
		{
			name: "a relative path is untouched",
			in:   "docs/vault",
			want: "docs/vault",
		},
		{
			// Resolving another account's home is not portable, and guessing the
			// current user's would be a different wrong answer rather than none.
			name: "another user's home is left alone",
			in:   "~someone/vault",
			want: "~someone/vault",
		},
		{
			name: "a tilde inside the path is not a home reference",
			in:   "/home/me/back~up/vault",
			want: "/home/me/back~up/vault",
		},
		{
			name: "empty stays empty",
			in:   "",
			want: "",
		},
		{
			name: "whitespace only collapses to empty",
			in:   "   ",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExpandUserPath(tt.in); got != tt.want {
				t.Errorf("ExpandUserPath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParsePlanExpandsTheVaultPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home directory available: %v", err)
	}

	t.Run("from the flag", func(t *testing.T) {
		plan, err := ParsePlanFromFlags(FlagSet{DocsMode: "vault", Path: "~/vault"}, AppConfig{})
		if err != nil {
			t.Fatalf("ParsePlanFromFlags: %v", err)
		}
		if want := filepath.Join(home, "vault"); plan.BasePath != want {
			t.Errorf("BasePath = %q, want %q", plan.BasePath, want)
		}
	})

	t.Run("from a config written before expansion existed", func(t *testing.T) {
		plan, err := ParsePlanFromFlags(FlagSet{DocsMode: "vault"}, AppConfig{Path: "~/legacy-vault"})
		if err != nil {
			t.Fatalf("ParsePlanFromFlags: %v", err)
		}
		if want := filepath.Join(home, "legacy-vault"); plan.BasePath != want {
			t.Errorf("BasePath = %q, want %q", plan.BasePath, want)
		}
	})
}
