package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// doc-reader parity tests (T3-5 RED → GREEN)
// Registry parity goes RED when doc-reader lands in content.json and GREEN
// when the registryTemplate row is also added. Both changes are in one commit.
// ---------------------------------------------------------------------------

// TestDocReaderRegistration_SkillListed verifies doc-reader is in the skills
// array of content.json.
func TestDocReaderRegistration_SkillListed(t *testing.T) {
	data, err := embedded.ReadFile("src/manifests/content.json")
	if err != nil {
		t.Fatalf("cannot read content.json: %v", err)
	}
	var cm ContentManifest
	if err := json.Unmarshal(data, &cm); err != nil {
		t.Fatalf("cannot unmarshal content.json: %v", err)
	}

	found := false
	for _, s := range cm.Skills {
		if s == "doc-reader" {
			found = true
			break
		}
	}
	if !found {
		t.Error("doc-reader not found in content.json skills array — add it to trigger conditional install")
	}
}

// TestDocReaderRegistration_RegistryRow verifies registryTemplate contains a
// doc-reader row in the User Skills table.
func TestDocReaderRegistration_RegistryRow(t *testing.T) {
	output := registryTemplate("/base", "/skills", "opencode")

	if !strings.Contains(output, "doc-reader") {
		t.Fatal("registryTemplate missing doc-reader in User Skills table")
	}
	// Normalize path to forward slashes for cross-platform comparison.
	normalOutput := filepath.ToSlash(output)
	if !strings.Contains(normalOutput, "/skills/doc-reader/SKILL.md") {
		t.Errorf("registryTemplate doc-reader row missing SKILL.md path")
	}
}

// TestDocReaderRegistration_CompactRules verifies registryTemplate contains a
// compact rules block for doc-reader with the required content.
func TestDocReaderRegistration_CompactRules(t *testing.T) {
	output := registryTemplate("/base", "/skills", "opencode")

	if !strings.Contains(output, "### doc-reader") {
		t.Fatal("registryTemplate missing ### doc-reader compact rules section")
	}

	requiredRules := []string{
		"agent_sdd_context_project",
		"in-project",
		"_sdd-context.md",
		"_sdd-tech-context.md",
		"doc-to-sdd",
	}
	for _, rule := range requiredRules {
		if !strings.Contains(output, rule) {
			t.Errorf("doc-reader compact rules missing required text: %q", rule)
		}
	}
}

// TestDocReaderRegistration_ParityAllManifestSkillsInRegistry is the forward
// assertion: if doc-reader is in content.json, it must also be in registryTemplate.
// This complements TestRegistryParity_AllManifestSkillsInRegistry.
func TestDocReaderRegistration_ParityAllManifestSkillsInRegistry(t *testing.T) {
	data, err := embedded.ReadFile("src/manifests/content.json")
	if err != nil {
		t.Fatalf("cannot read content.json: %v", err)
	}
	var cm ContentManifest
	if err := json.Unmarshal(data, &cm); err != nil {
		t.Fatalf("cannot unmarshal content.json: %v", err)
	}

	const testBase = "/base"
	const testSkillsDir = "/skills"
	output := registryTemplate(testBase, testSkillsDir, "opencode")
	inRegistry := registrySkillIDs(output, testSkillsDir)

	for _, skillID := range cm.Skills {
		if skillID == "doc-reader" && !inRegistry[skillID] {
			t.Errorf("doc-reader is in content.json but missing from registryTemplate — atomicity violation")
		}
	}
}
