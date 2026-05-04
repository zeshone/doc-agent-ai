package main

// ContentManifest matches the schema of src/manifests/content.json.
type ContentManifest struct {
	PlaceholderBasePath string          `json:"placeholderBasePath"`
	Skills              []string        `json:"skills"`
	Roles               []RoleConfig    `json:"roles"`
	Commands            []CommandConfig `json:"commands"`
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

// PlatformManifest matches the schema of src/manifests/platforms.json.
type PlatformManifest struct {
	OpenCode PlatformConfig `json:"opencode"`
	Copilot  PlatformConfig `json:"copilot"`
	Claude   PlatformConfig `json:"claude"`
	Qwen     PlatformConfig `json:"qwen"`
}

// PlatformConfig holds configuration for a single AI platform.
type PlatformConfig struct {
	SkillRoot                string   `json:"skillRoot"`
	PromptDir                string   `json:"promptDir"`
	CommandDir               string   `json:"commandDir,omitempty"`
	AgentDir                 string   `json:"agentDir,omitempty"`
	AgentExtension           string   `json:"agentExtension,omitempty"`
	AgentTemplate            string   `json:"agentTemplate,omitempty"`
	AgentTools               []string `json:"agentTools,omitempty"`
	OrchestratorTools        []string `json:"orchestratorTools,omitempty"`
	ApprovalMode             string   `json:"approvalMode,omitempty"`
	OrchestratorApprovalMode string   `json:"orchestratorApprovalMode,omitempty"`
}

// DistManifest is written to dist/manifest.json by the generate subcommand.
type DistManifest struct {
	GeneratedAt         string           `json:"generatedAt"`
	PlaceholderBasePath string           `json:"placeholderBasePath"`
	Skills              []string         `json:"skills"`
	Roles               []DistRole       `json:"roles"`
	Commands            []DistCommand    `json:"commands"`
	Platforms           PlatformManifest `json:"platforms"`
}

// DistRole represents a role in the output manifest.
type DistRole struct {
	ID            string          `json:"id"`
	Description   string          `json:"description"`
	Hidden        bool            `json:"hidden"`
	Mode          string          `json:"mode"`
	OpenCodeTools map[string]bool `json:"opencodeTools,omitempty"`
	PromptFiles   PromptFileMap   `json:"promptFiles"`
	AgentFiles    AgentFileMap    `json:"agentFiles"`
}

// PromptFileMap maps platform IDs to prompt file paths. Field order matches
// the v2.0.0 JS Object.fromEntries insertion order for byte-level parity.
type PromptFileMap struct {
	OpenCode string `json:"opencode"`
	Copilot  string `json:"copilot"`
	Claude   string `json:"claude"`
	Qwen     string `json:"qwen"`
}

// AgentFileMap maps platform IDs to agent file paths. Only platforms with
// agent support are included (opencode is excluded). Field order matches
// the v2.0.0 JS Object.fromEntries insertion order.
type AgentFileMap struct {
	Copilot string `json:"copilot"`
	Claude  string `json:"claude"`
	Qwen    string `json:"qwen"`
}

// DistCommand represents a command in the output manifest.
type DistCommand struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Agent       string `json:"agent"`
	File        string `json:"file"`
}
