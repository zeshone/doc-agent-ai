package main

// ContentManifest matches the schema of src/manifests/content.json.
type ContentManifest struct {
	PlaceholderBasePath string          `json:"placeholderBasePath"`
	Skills              []string        `json:"skills"`
	ConditionalSkills   []string        `json:"conditionalSkills,omitempty"`
	Roles               []RoleConfig    `json:"roles"`
	Commands            []CommandConfig `json:"commands"`
	LegacyCommandIds    []string        `json:"legacyCommandIds,omitempty"`
}

// RoleConfig represents a single role entry in content.json.
type RoleConfig struct {
	ID              string          `json:"id"`
	Content         string          `json:"content"`
	Skill           string          `json:"skill"`
	RulesSkill      string          `json:"rulesSkill"`
	Description     string          `json:"description"`
	UserInvocable   bool            `json:"userInvocable"`
	Hidden          bool            `json:"hidden"`
	Mode            string          `json:"mode"`
	OpenCodeTools   map[string]bool `json:"opencodeTools,omitempty"`
	CopilotChildren []string        `json:"copilotChildren,omitempty"`
}

// CommandConfig represents a single command entry in content.json.
type CommandConfig struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Agent       string `json:"agent"`
	Content     string `json:"content"`
}
