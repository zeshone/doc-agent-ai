package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// registrySkillIDs extracts the skill IDs referenced in a registryTemplate
// output by scanning table rows for cells that contain a path under skillsDir
// and end with SKILL.md. Works on both Unix and Windows because it normalises
// slashes before comparing.
func registrySkillIDs(registryContent, skillsDir string) map[string]bool {
	// Normalise to forward slashes so the comparison works on Windows too.
	normalDir := filepath.ToSlash(skillsDir)

	ids := make(map[string]bool)
	for _, line := range strings.Split(registryContent, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			continue
		}
		// Split the Markdown table row into cells.
		cells := strings.Split(line, "|")
		for _, cell := range cells {
			cell = strings.TrimSpace(cell)
			// Normalise any backslashes in the cell value.
			normalCell := filepath.ToSlash(cell)
			// A skill path cell contains skillsDir and ends with /SKILL.md.
			if strings.Contains(normalCell, normalDir) && strings.HasSuffix(normalCell, "SKILL.md") {
				// Derive the skill ID from the directory component after skillsDir.
				rel := strings.TrimPrefix(normalCell, normalDir)
				rel = strings.TrimPrefix(rel, "/")
				parts := strings.SplitN(rel, "/", 2)
				if len(parts) >= 1 && parts[0] != "" {
					ids[parts[0]] = true
				}
			}
		}
	}
	return ids
}

// TestRegistryParity_AllManifestSkillsInRegistry verifies that every skill ID
// listed in content.json appears as a skill path entry in registryTemplate output.
func TestRegistryParity_AllManifestSkillsInRegistry(t *testing.T) {
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

	var missing []string
	for _, skillID := range cm.Skills {
		if !inRegistry[skillID] {
			missing = append(missing, skillID)
		}
	}

	if len(missing) > 0 {
		t.Errorf("skills listed in content.json but missing from registryTemplate: %v\n"+
			"Add the missing skills to registryTemplate() in platform.go.", missing)
	}
}

// TestRegistryParity_NoOrphanEntriesInRegistry verifies that every skill path
// in registryTemplate output corresponds to a skill ID listed in content.json.
func TestRegistryParity_NoOrphanEntriesInRegistry(t *testing.T) {
	data, err := embedded.ReadFile("src/manifests/content.json")
	if err != nil {
		t.Fatalf("cannot read content.json: %v", err)
	}

	var cm ContentManifest
	if err := json.Unmarshal(data, &cm); err != nil {
		t.Fatalf("cannot unmarshal content.json: %v", err)
	}

	manifestSkills := make(map[string]bool, len(cm.Skills))
	for _, id := range cm.Skills {
		manifestSkills[id] = true
	}

	const testBase = "/base"
	const testSkillsDir = "/skills"

	output := registryTemplate(testBase, testSkillsDir, "opencode")
	inRegistry := registrySkillIDs(output, testSkillsDir)

	var orphans []string
	for id := range inRegistry {
		if !manifestSkills[id] {
			orphans = append(orphans, id)
		}
	}

	if len(orphans) > 0 {
		t.Errorf("skills in registryTemplate but NOT in content.json: %v\n"+
			"Remove or rename these entries in registryTemplate() in platform.go.", orphans)
	}
}
