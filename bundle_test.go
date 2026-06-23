package main

import (
	"strings"
	"testing"
)

func TestBuildBundle_IncludesManifestAndFiles(t *testing.T) {
	bundle, err := BuildBundle()
	if err != nil {
		t.Fatalf("BuildBundle: %v", err)
	}

	if len(bundle.Manifest.Roles) == 0 {
		t.Fatal("BuildBundle returned an empty manifest role list")
	}
	if len(bundle.Files) == 0 {
		t.Fatal("BuildBundle returned no rendered files")
	}
	if _, ok := bundle.Files["skills/doc-arch/SKILL.md"]; !ok {
		t.Fatal("bundle missing skills/doc-arch/SKILL.md")
	}

	foundPrompt := false
	for rel := range bundle.Files {
		if strings.HasPrefix(rel, bundle.Manifest.Platforms.OpenCode.PromptDir+"/") && strings.HasSuffix(rel, ".md") {
			foundPrompt = true
			break
		}
	}
	if !foundPrompt {
		t.Fatal("bundle missing an opencode prompt file")
	}

	for _, role := range bundle.Manifest.Roles {
		for _, rel := range []string{
			role.PromptFiles.OpenCode,
			role.PromptFiles.Copilot,
			role.PromptFiles.Claude,
			role.PromptFiles.Qwen,
			role.PromptFiles.Pi,
			role.AgentFiles.Copilot,
			role.AgentFiles.Claude,
			role.AgentFiles.Qwen,
		} {
			assertBundleHasPath(t, bundle, rel)
		}
	}

	for _, cmd := range bundle.Manifest.Commands {
		assertBundleHasPath(t, bundle, cmd.File)
	}

	for _, skill := range bundle.Manifest.Skills {
		prefix := "skills/" + skill + "/"
		found := false
		for rel := range bundle.Files {
			if strings.HasPrefix(rel, prefix) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("bundle missing skill files under %s", prefix)
		}
	}
}

func assertBundleHasPath(t *testing.T, bundle Bundle, rel string) {
	t.Helper()
	if rel == "" {
		return
	}
	if _, ok := bundle.Files[rel]; !ok {
		t.Fatalf("bundle missing manifest-referenced path %q", rel)
	}
}
