package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// copilotPathOverride holds the value of --copilot-path when provided on the
// command line.  Set in main() before any platform construction occurs.
var copilotPathOverride string

// ---------------------------------------------------------------------------
// Platform interface
// ---------------------------------------------------------------------------

// Platform abstracts platform-specific paths, detection, and config patching.
// An empty AgentsDir() return means the platform does not use file-based agents
// (e.g. opencode stores agent definitions in opencode.json).
type Platform interface {
	ID() string
	HomeDir() string
	Detect() bool
	SkillsDir() string
	PromptsDir() string
	AgentsDir() string // "" when platform uses JSON-based agent registry (opencode)

	// GetAgentIDs returns role IDs that have agent definitions installed.
	// installed, if non-nil, acts as a candidate filter: only roles whose IDs
	// are keys are checked.  The map is also populated with results.
	GetAgentIDs(manifest DistManifest, installed map[string]bool) ([]string, error)

	// GetPromptIDs returns role IDs that have prompt files installed.
	// installed has the same filter+accumulate semantics as GetAgentIDs.
	GetPromptIDs(manifest DistManifest, installed map[string]bool) ([]string, error)

	// GetSkillIDs returns skill IDs installed on this platform.
	GetSkillIDs(manifest DistManifest) []string

	// GetCommandIDs returns command IDs installed on this platform.
	// Only opencode has command support; other platforms return (nil, nil).
	GetCommandIDs(manifest DistManifest) ([]string, error)

	// PatchConfig merges agent entries into the platform's config file
	// (opencode.json for opencode). roleIDs filters which roles to patch;
	// empty or nil means all roles.  Other platforms return nil.
	PatchConfig(manifest DistManifest, basePath string, roleIDs []string) error

	// RemoveConfig removes doc-agent-ai agent entries from the platform's config.
	// roleIDs filters which roles to remove; empty or nil means all roles.
	// Other platforms return nil.
	RemoveConfig(manifest DistManifest, roleIDs []string) error

	// SkillRegistryTrigger returns the platform-specific trigger text for the
	// doc-arch skill in the skill registry ("opencode" or "claude" style).
	SkillRegistryTrigger() string

	// WriteSkillRegistry writes .atl/skill-registry.md to the platform home.
	// Only opencode and claude write a registry; other platforms return nil.
	WriteSkillRegistry(basePath string) error
}

// ---------------------------------------------------------------------------
// basePlatform — common fields and shared method implementations
// ---------------------------------------------------------------------------

type basePlatform struct {
	id      string
	homeDir string
	cfg     PlatformConfig
}

func (b *basePlatform) ID() string               { return b.id }
func (b *basePlatform) HomeDir() string           { return b.homeDir }
func (b *basePlatform) SkillsDir() string         { return filepath.Join(b.homeDir, "skills") }
func (b *basePlatform) PromptsDir() string        { return filepath.Join(b.homeDir, "prompts", "doc") }

// skillRegistryPath returns the path to .atl/skill-registry.md for this platform.
func (b *basePlatform) skillRegistryPath() string {
	return filepath.Join(b.homeDir, ".atl", "skill-registry.md")
}

// ---------------------------------------------------------------------------
// opencodePlatform
// ---------------------------------------------------------------------------

type opencodePlatform struct {
	basePlatform
}

func (p *opencodePlatform) AgentsDir() string { return "" } // JSON-based agent registry

func (p *opencodePlatform) Detect() bool {
	configPath := filepath.Join(p.homeDir, "opencode.json")
	_, err := os.Stat(configPath)
	return err == nil
}

func (p *opencodePlatform) GetAgentIDs(manifest DistManifest, installed map[string]bool) ([]string, error) {
	configPath := filepath.Join(p.homeDir, "opencode.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read opencode.json: %w", err)
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse opencode.json: %w", err)
	}

	agents, _ := config["agent"].(map[string]any)
	if agents == nil {
		return nil, nil
	}

	roleIDs := candidateRoleIDs(manifest, installed)
	var result []string
	for _, id := range roleIDs {
		if _, ok := agents[id]; ok {
			result = append(result, id)
			if installed != nil {
				installed[id] = true
			}
		}
	}
	return result, nil
}

func (p *opencodePlatform) GetPromptIDs(manifest DistManifest, installed map[string]bool) ([]string, error) {
	return getPromptIDsFromDir(p.PromptsDir(), manifest, installed), nil
}

func (p *opencodePlatform) GetSkillIDs(manifest DistManifest) []string {
	return getSkillIDsFromDir(p.SkillsDir(), manifest)
}

func (p *opencodePlatform) GetCommandIDs(manifest DistManifest) ([]string, error) {
	cmdsDir := filepath.Join(p.homeDir, "commands")
	var result []string
	for _, cmd := range manifest.Commands {
		cmdPath := filepath.Join(cmdsDir, cmd.ID+".md")
		if _, err := os.Stat(cmdPath); err == nil {
			result = append(result, cmd.ID)
		}
	}
	return result, nil
}

func (p *opencodePlatform) PatchConfig(manifest DistManifest, basePath string, roleIDs []string) error {
	configPath := filepath.Join(p.homeDir, "opencode.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read opencode.json: %w", err)
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("parse opencode.json: %w", err)
	}

	if config["agent"] == nil {
		config["agent"] = make(map[string]any)
	}
	agents, ok := config["agent"].(map[string]any)
	if !ok {
		agents = make(map[string]any)
		config["agent"] = agents
	}

	promptsBase := p.PromptsDir()
	filter := toSet(roleIDs)

	for _, role := range manifest.Roles {
		if len(filter) > 0 && !filter[role.ID] {
			continue
		}

		entry := map[string]any{
			"description": role.Description,
			"mode":        role.Mode,
			"prompt":      fmt.Sprintf("{file:%s}", filepath.Join(promptsBase, role.ID+".md")),
			"tools":       role.OpenCodeTools,
		}
		if role.Hidden {
			entry["hidden"] = true
		}
		agents[role.ID] = entry
	}

	out, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal opencode.json: %w", err)
	}
	out = append(out, '\n')

	if err := os.WriteFile(configPath, out, 0644); err != nil {
		return fmt.Errorf("write opencode.json: %w", err)
	}
	return nil
}

func (p *opencodePlatform) RemoveConfig(manifest DistManifest, roleIDs []string) error {
	configPath := filepath.Join(p.homeDir, "opencode.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read opencode.json: %w", err)
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("parse opencode.json: %w", err)
	}

	agents, ok := config["agent"].(map[string]any)
	if !ok {
		return nil // no agent section, nothing to remove
	}

	// Build the set of role IDs known to the manifest so we only remove
	// doc-agent-ai agents and leave user-defined agents untouched.
	manifestRoles := toSet(manifestRoleIDs(manifest))
	filter := toSet(roleIDs)

	for id := range agents {
		if !manifestRoles[id] {
			continue // not a doc-agent-ai agent, preserve it
		}
		if len(filter) > 0 && !filter[id] {
			continue
		}
		delete(agents, id)
	}

	out, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal opencode.json: %w", err)
	}
	out = append(out, '\n')

	if err := os.WriteFile(configPath, out, 0644); err != nil {
		return fmt.Errorf("write opencode.json: %w", err)
	}
	return nil
}

func (p *opencodePlatform) SkillRegistryTrigger() string { return "opencode" }

func (p *opencodePlatform) WriteSkillRegistry(basePath string) error {
	return writeSkillRegistryTo(p.skillRegistryPath(), basePath, p.SkillsDir(), "opencode")
}

// ---------------------------------------------------------------------------
// qwenPlatform
// ---------------------------------------------------------------------------

type qwenPlatform struct {
	basePlatform
}

func (p *qwenPlatform) AgentsDir() string { return filepath.Join(p.homeDir, "agents") }

func (p *qwenPlatform) Detect() bool {
	_, err := os.Stat(p.homeDir)
	return err == nil
}

func (p *qwenPlatform) GetAgentIDs(manifest DistManifest, installed map[string]bool) ([]string, error) {
	return getAgentIDsFromDir(p.AgentsDir(), manifest, installed, ""), nil
}

func (p *qwenPlatform) GetPromptIDs(manifest DistManifest, installed map[string]bool) ([]string, error) {
	return getPromptIDsFromDir(p.PromptsDir(), manifest, installed), nil
}

func (p *qwenPlatform) GetSkillIDs(manifest DistManifest) []string {
	return getSkillIDsFromDir(p.SkillsDir(), manifest)
}

func (p *qwenPlatform) GetCommandIDs(manifest DistManifest) ([]string, error) {
	return nil, nil
}

func (p *qwenPlatform) PatchConfig(manifest DistManifest, basePath string, roleIDs []string) error {
	return nil
}

func (p *qwenPlatform) RemoveConfig(manifest DistManifest, roleIDs []string) error {
	return nil
}

func (p *qwenPlatform) SkillRegistryTrigger() string { return "opencode" }

func (p *qwenPlatform) WriteSkillRegistry(basePath string) error {
	return nil
}

// ---------------------------------------------------------------------------
// copilotPlatform
// ---------------------------------------------------------------------------

type copilotPlatform struct {
	basePlatform
}

func (p *copilotPlatform) AgentsDir() string { return filepath.Join(p.homeDir, "agents") }

func (p *copilotPlatform) Detect() bool {
	// homeDir was already set to the override or the standard ~/.copilot in the
	// constructor, so a non-empty homeDir that exists is sufficient.
	_, err := os.Stat(p.homeDir)
	return err == nil
}

func (p *copilotPlatform) GetAgentIDs(manifest DistManifest, installed map[string]bool) ([]string, error) {
	return getAgentIDsFromDir(p.AgentsDir(), manifest, installed, ""), nil
}

func (p *copilotPlatform) GetPromptIDs(manifest DistManifest, installed map[string]bool) ([]string, error) {
	return getPromptIDsFromDir(p.PromptsDir(), manifest, installed), nil
}

func (p *copilotPlatform) GetSkillIDs(manifest DistManifest) []string {
	return getSkillIDsFromDir(p.SkillsDir(), manifest)
}

func (p *copilotPlatform) GetCommandIDs(manifest DistManifest) ([]string, error) {
	return nil, nil
}

func (p *copilotPlatform) PatchConfig(manifest DistManifest, basePath string, roleIDs []string) error {
	return nil
}

func (p *copilotPlatform) RemoveConfig(manifest DistManifest, roleIDs []string) error {
	return nil
}

func (p *copilotPlatform) SkillRegistryTrigger() string { return "opencode" }

func (p *copilotPlatform) WriteSkillRegistry(basePath string) error {
	return nil
}

// ---------------------------------------------------------------------------
// claudePlatform
// ---------------------------------------------------------------------------

type claudePlatform struct {
	basePlatform
}

func (p *claudePlatform) AgentsDir() string { return filepath.Join(p.homeDir, "agents") }

func (p *claudePlatform) Detect() bool {
	_, err := os.Stat(p.homeDir)
	return err == nil
}

func (p *claudePlatform) GetAgentIDs(manifest DistManifest, installed map[string]bool) ([]string, error) {
	return getAgentIDsFromDir(p.AgentsDir(), manifest, installed, ""), nil
}

func (p *claudePlatform) GetPromptIDs(manifest DistManifest, installed map[string]bool) ([]string, error) {
	return getPromptIDsFromDir(p.PromptsDir(), manifest, installed), nil
}

func (p *claudePlatform) GetSkillIDs(manifest DistManifest) []string {
	return getSkillIDsFromDir(p.SkillsDir(), manifest)
}

func (p *claudePlatform) GetCommandIDs(manifest DistManifest) ([]string, error) {
	return nil, nil
}

func (p *claudePlatform) PatchConfig(manifest DistManifest, basePath string, roleIDs []string) error {
	return nil
}

func (p *claudePlatform) RemoveConfig(manifest DistManifest, roleIDs []string) error {
	return nil
}

func (p *claudePlatform) SkillRegistryTrigger() string { return "claude" }

func (p *claudePlatform) WriteSkillRegistry(basePath string) error {
	return writeSkillRegistryTo(p.skillRegistryPath(), basePath, p.SkillsDir(), "claude")
}

// ---------------------------------------------------------------------------
// Shared helper functions
// ---------------------------------------------------------------------------

// resolveHome replaces a leading "~" with os.UserHomeDir() and returns the
// parent directory of the resolved skillRoot as the platform home dir.
func resolveHome(skillRoot string) (string, error) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	resolved := strings.Replace(skillRoot, "~", userHome, 1)
	return filepath.Dir(resolved), nil
}

// opencodeConfigDir returns the opencode config directory, honouring
// XDG_CONFIG_HOME when set and non-empty.  Falls back to ~/.config/opencode.
func opencodeConfigDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "opencode"), nil
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(userHome, ".config", "opencode"), nil
}

// newOpenCodePlatform creates an opencode platform from its config.
// Resolution rule: XDG_CONFIG_HOME takes priority when set; otherwise falls back
// to ~/.config/opencode. The cfg.SkillRoot field is intentionally ignored here —
// opencode follows the XDG Base Directory spec rather than the home-suffix
// convention used by qwen/copilot/claude.
func newOpenCodePlatform(cfg PlatformConfig) (*opencodePlatform, error) {
	home, err := opencodeConfigDir()
	if err != nil {
		return nil, err
	}
	return &opencodePlatform{basePlatform{id: "opencode", homeDir: home, cfg: cfg}}, nil
}

// newQwenPlatform creates a qwen platform from its config.
func newQwenPlatform(cfg PlatformConfig) (*qwenPlatform, error) {
	home, err := resolveHome(cfg.SkillRoot)
	if err != nil {
		return nil, err
	}
	return &qwenPlatform{basePlatform{id: "qwen", homeDir: home, cfg: cfg}}, nil
}

// findPortableCopilotHome attempts to locate the VS Code portable Copilot Chat
// globalStorage directory by asking 'code' where it installed its shell
// integration.  Returns ("", false) if code is not in PATH, the subprocess
// fails, or the expected directory does not exist.
func findPortableCopilotHome() (string, bool) {
	out, err := exec.Command("code", "--locate-shell-integration-path", "bash").Output()
	if err != nil {
		return "", false
	}
	// The output is a path like /.../vscode/bin/code; the install root is four
	// directories up from that path (bin/code → <install>/bin/code).
	shellPath := strings.TrimSpace(string(out))
	if shellPath == "" {
		return "", false
	}
	// Walk upward to find the install root: locate "data/user-data" relative to
	// the directory reported by --locate-shell-integration-path.  The integration
	// path lives at <install>/out/vs/workbench/…, so we probe parent dirs.
	candidate := shellPath
	for i := 0; i < 10; i++ {
		candidate = filepath.Dir(candidate)
		copilotDir := filepath.Join(candidate, "data", "user-data", "User",
			"globalStorage", "github.copilot-chat")
		if _, err := os.Stat(copilotDir); err == nil {
			return copilotDir, true
		}
	}
	return "", false
}

// newCopilotPlatform creates a copilot platform from its config.
// Resolution order: --copilot-path override → ~/.copilot standard → portable VS Code.
func newCopilotPlatform(cfg PlatformConfig) (*copilotPlatform, error) {
	// 1. Explicit override from --copilot-path flag.
	if copilotPathOverride != "" {
		if _, err := os.Stat(copilotPathOverride); os.IsNotExist(err) {
			warn("Path does not exist: " + copilotPathOverride)
		}
		return &copilotPlatform{basePlatform{id: "copilot", homeDir: copilotPathOverride, cfg: cfg}}, nil
	}

	// 2. Standard ~/.copilot directory — requires that `code` is also on PATH,
	//    otherwise the directory may be a stale leftover from an uninstalled VS Code.
	stdHome, err := resolveHome(cfg.SkillRoot) // resolves ~/.copilot/skills → ~/.copilot
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(stdHome); err == nil {
		if _, lookErr := exec.LookPath("code"); lookErr == nil {
			return &copilotPlatform{basePlatform{id: "copilot", homeDir: stdHome, cfg: cfg}}, nil
		}
	}

	// 3. Portable VS Code: look for globalStorage/github.copilot-chat via `code`.
	if portableHome, ok := findPortableCopilotHome(); ok {
		return &copilotPlatform{basePlatform{id: "copilot", homeDir: portableHome, cfg: cfg}}, nil
	}

	// No copilot home found; return with stdHome so Detect() returns false.
	return &copilotPlatform{basePlatform{id: "copilot", homeDir: stdHome, cfg: cfg}}, nil
}

// newClaudePlatform creates a claude platform from its config.
func newClaudePlatform(cfg PlatformConfig) (*claudePlatform, error) {
	home, err := resolveHome(cfg.SkillRoot)
	if err != nil {
		return nil, err
	}
	return &claudePlatform{basePlatform{id: "claude", homeDir: home, cfg: cfg}}, nil
}

// newPlatform creates a Platform for the given platform ID from its config.
func newPlatform(id string, cfg PlatformConfig) (Platform, error) {
	switch id {
	case "opencode":
		return newOpenCodePlatform(cfg)
	case "qwen":
		return newQwenPlatform(cfg)
	case "copilot":
		return newCopilotPlatform(cfg)
	case "claude":
		return newClaudePlatform(cfg)
	default:
		return nil, fmt.Errorf("unknown platform: %s", id)
	}
}

// --- file-system helpers shared by platform methods ---

// getSkillIDsFromDir returns skill IDs from manifest whose skill dirs exist on
// disk under skillsDir.
func getSkillIDsFromDir(skillsDir string, manifest DistManifest) []string {
	var result []string
	for _, skill := range manifest.Skills {
		if _, err := os.Stat(filepath.Join(skillsDir, skill)); err == nil {
			result = append(result, skill)
		}
	}
	return result
}

// getPromptIDsFromDir returns role IDs that have a .md prompt file in promptsDir.
// installed, if non-nil, acts as a candidate filter.
func getPromptIDsFromDir(promptsDir string, manifest DistManifest, installed map[string]bool) []string {
	roleIDs := candidateRoleIDs(manifest, installed)
	var result []string
	for _, id := range roleIDs {
		promptPath := filepath.Join(promptsDir, id+".md")
		if _, err := os.Stat(promptPath); err == nil {
			result = append(result, id)
			if installed != nil {
				installed[id] = true
			}
		}
	}
	return result
}

// getAgentIDsFromDir returns role IDs that have agent files in agentsDir.
// The file is expected to be named <roleID><extension> (extension is
// platform-specific, e.g. ".md" for qwen/claude, ".agent.md" for copilot).
// When extension is "", any file whose name starts with <roleID> matches.
func getAgentIDsFromDir(agentsDir string, manifest DistManifest, installed map[string]bool, extension string) []string {
	roleIDs := candidateRoleIDs(manifest, installed)
	var result []string
	for _, id := range roleIDs {
		var checkPath string
		if extension != "" {
			checkPath = filepath.Join(agentsDir, id+extension)
		} else {
			// No extension specified — check common patterns
			candidates := []string{
				filepath.Join(agentsDir, id+".md"),
				filepath.Join(agentsDir, id+".agent.md"),
			}
			found := false
			for _, c := range candidates {
				if _, err := os.Stat(c); err == nil {
					found = true
					break
				}
			}
			if !found {
				continue
			}
			checkPath = filepath.Join(agentsDir, id+".md") // fallback for naming
		}
		// For extension != "" case
		if extension != "" {
			if _, err := os.Stat(checkPath); err == nil {
				result = append(result, id)
				if installed != nil {
					installed[id] = true
				}
			}
		} else {
			result = append(result, id)
			if installed != nil {
				installed[id] = true
			}
		}
	}
	return result
}

// candidateRoleIDs returns the role IDs to check: all roles from the manifest
// filtered by the installed map if non-nil.
func candidateRoleIDs(manifest DistManifest, installed map[string]bool) []string {
	all := manifestRoleIDs(manifest)
	if installed == nil {
		return all
	}
	var filtered []string
	for _, id := range all {
		if _, ok := installed[id]; ok {
			filtered = append(filtered, id)
		}
	}
	return filtered
}

// manifestRoleIDs returns all role IDs from a DistManifest.
func manifestRoleIDs(manifest DistManifest) []string {
	ids := make([]string, len(manifest.Roles))
	for i, role := range manifest.Roles {
		ids[i] = role.ID
	}
	return ids
}

// toSet converts a slice of strings to a set (map[string]bool).
func toSet(items []string) map[string]bool {
	if len(items) == 0 {
		return nil
	}
	m := make(map[string]bool, len(items))
	for _, item := range items {
		m[item] = true
	}
	return m
}

// --- skill registry ---

// registryTemplate returns the full content of .atl/skill-registry.md.
// triggerStyle is "opencode" or "claude".
func registryTemplate(basePath, skillsDir, triggerStyle string) string {
	docArchTrigger := "`/arch`, `/idea`, `/rec`, `/prd`, `/refine`, `/tech`, `/pti`, `/mod` — project documentation workflow"
	if triggerStyle == "claude" {
		docArchTrigger = "Documenting a project, /arch, /idea, /rec, /prd, /refine, /tech, /pti, /mod"
	}

	return fmt.Sprintf(`# Skill Registry — doc-agent-ai

**Auto-generated by doc-agent-ai installer v%s**
**Base path:** %s

## User Skills

| Trigger | Skill | Path |
|---------|-------|------|
| %s | doc-arch | %s |
| Refining a vague product idea into a clear concept | doc-idea | %s |
| Gathering requirements, elicitation session, stakeholder interview | doc-rec | %s |
| Generating PRD for a project | doc-prd | %s |
| Auditing user stories against INVEST criteria | doc-refinement | %s |
| Creating technical specification, new feature needing documented architecture | doc-tech | %s |
| Converting PRD into local issues, breaking down PRD into work items | doc-pti | %s |

## Compact Rules

### doc-arch
- Base path for all projects: `+"`%s`"+`
- Commands: `+"`arch <s>`"+` (full flow), `+"`idea`"+`, `+"`rec`"+`, `+"`prd`"+`, `+"`refine`"+`, `+"`tech`"+`, `+"`pti`"+` (individual), `+"`mod <s> <m>`"+` (module)
- Workflow order: idea → rec → prd → refine → tech → pti (6 phases)
- Module paths: `+"`<sistema>/<modulo>`"+` and `+"`<sistema>/<modulo>/<submodulo>`"+` — max 2 levels deep
- Archetype detected in `+"`rec`"+`: bounded (single delivery) or evolving (long lifecycle with modules)
- Index file `+"`<sistema>.md`"+` uses Obsidian `+"`[[...]]`"+` links and checkboxes — update after every phase
- Tech spec for modules: ALWAYS ask if inherits parent architecture (delta) or diverges (full spec)
- Modules read parent context: rec reads parent requirements, prd reads parent prd
- Issues generated as local .md only — never push to GitHub unless user explicitly asks
- Never invent context — if prerequisite missing, stop and show exact command to run

### doc-idea
- Pure product discovery — no stack, no APIs, no databases
- Output: master index description plus optional `+"`_idea-brief.md`"+`

### doc-rec
- Start in executive/business language and increase technical depth progressively
- Do not open with BABOK jargon or implementation detail unless stakeholder already speaks that way

### doc-prd
- PRD is more technical than rec, but clear and pedagogical
- Ask before assuming; unknowns become `+"`TBD`"+`

### doc-refinement
- Quality gate for user stories — audit against INVEST criteria
- Never add, delete, or change story scope without explicit user confirmation

### doc-tech
- Highest technical precision, still legible
- Ask before assuming; unknowns become `+"`TBD`"+` / `+"`Open Decision`"+`

### doc-pti
- Local-first issues, end-to-end slices, visible blockers and TBDs
`,
		version,
		basePath,
		docArchTrigger,
		filepath.Join(skillsDir, "doc-arch", "SKILL.md"),
		filepath.Join(skillsDir, "doc-idea", "SKILL.md"),
		filepath.Join(skillsDir, "doc-rec", "SKILL.md"),
		filepath.Join(skillsDir, "doc-prd", "SKILL.md"),
		filepath.Join(skillsDir, "doc-refinement", "SKILL.md"),
		filepath.Join(skillsDir, "doc-tech", "SKILL.md"),
		filepath.Join(skillsDir, "doc-pti", "SKILL.md"),
		basePath,
	)
}

// writeSkillRegistryTo writes .atl/skill-registry.md to the given path.
func writeSkillRegistryTo(registryPath, basePath, skillsDir, triggerStyle string) error {
	dir := filepath.Dir(registryPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create registry dir: %w", err)
	}

	content := registryTemplate(basePath, skillsDir, triggerStyle)
	if err := os.WriteFile(registryPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write registry: %w", err)
	}
	return nil
}

// --- empty-directory cleanup ---

// pruneEmptyDirs removes empty directories starting from dir, walking
// upward until stopDir is reached. Directories outside stopDir are never
// removed (safety guard).
func pruneEmptyDirs(dir, stopDir string) error {
	for {
		if dir == stopDir || !strings.HasPrefix(dir, stopDir+string(filepath.Separator)) && dir != stopDir {
			return nil
		}
		info, err := os.Stat(dir)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("stat %s: %w", dir, err)
		}
		if !info.IsDir() {
			return nil
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return fmt.Errorf("readdir %s: %w", dir, err)
		}
		if len(entries) > 0 {
			return nil
		}
		if err := os.Remove(dir); err != nil {
			return fmt.Errorf("rmdir %s: %w", dir, err)
		}
		dir = filepath.Dir(dir)
	}
}
