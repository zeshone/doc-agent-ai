package install

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	configpkg "github.com/zeshone/doc-agent-ai/internal/config"
)

// ---------------------------------------------------------------------------
// ANSI color constants (matching install.js)
// ---------------------------------------------------------------------------

const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiRed    = "\x1b[31m"
	ansiCyan   = "\x1b[36m"
	ansiBlue   = "\x1b[34m"
	ansiGray   = "\x1b[90m"
)

// ---------------------------------------------------------------------------
// Output helpers (match install.js style exactly)
// ---------------------------------------------------------------------------

func ok(msg string)     { fmt.Printf("  %s✔%s %s\n", ansiGreen, ansiReset, msg) }
func warn(msg string)   { fmt.Printf("  %s⚠%s  %s\n", ansiYellow, ansiReset, msg) }
func errOut(msg string) { fmt.Printf("  %s✖%s %s\n", ansiRed, ansiReset, msg) }
func info(msg string)   { fmt.Printf("  %s→%s %s\n", ansiBlue, ansiReset, msg) }
func dim(msg string)    { fmt.Printf("%s  %s%s\n", ansiGray, msg, ansiReset) }
func head(msg string)   { fmt.Printf("\n%s  %s%s\n", ansiBold, msg, ansiReset) }

// ---------------------------------------------------------------------------
// Interactive input
// ---------------------------------------------------------------------------

// ask reads a line from scanner. Returns empty string on error/EOF.
func ask(scanner *bufio.Scanner, question string) string {
	fmt.Print(question)
	if scanner.Scan() {
		return scanner.Text()
	}
	return ""
}

// ---------------------------------------------------------------------------
// File system helpers
// ---------------------------------------------------------------------------

// ensureDir creates a directory and all parents if they don't exist.
func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}


// placeholderPair is a (placeholder, value) pair for ordered multi-token
// substitution. Order matters: each substitution runs in sequence on the
// result of the previous one.
type placeholderPair struct {
	placeholder string
	value       string
}

// replaceAllInFile applies an ordered list of (placeholder → value) substitutions
// to a file. Each substitution is applied to the result of the previous one.
// If a placeholder is not found in the current content, that step is a no-op.
func replaceAllInFile(path string, pairs []placeholderPair) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)
	changed := false
	for _, p := range pairs {
		if strings.Contains(content, p.placeholder) {
			content = strings.ReplaceAll(content, p.placeholder, p.value)
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// ---------------------------------------------------------------------------
// Platform detection
// ---------------------------------------------------------------------------

// detectAllPlatforms creates platform implementations from the manifest config
// and returns those that detect successfully.
func detectAllPlatforms(manifest DistManifest) []Platform {
	var platforms []Platform

	if p, err := newOpenCodePlatform(manifest.Platforms.OpenCode); err == nil && p.Detect() {
		platforms = append(platforms, p)
	}
	if p, err := newQwenPlatform(manifest.Platforms.Qwen); err == nil && p.Detect() {
		platforms = append(platforms, p)
	}
	if p, err := newCopilotPlatform(manifest.Platforms.Copilot); err == nil && p.Detect() {
		platforms = append(platforms, p)
	}
	if p, err := newClaudePlatform(manifest.Platforms.Claude); err == nil && p.Detect() {
		platforms = append(platforms, p)
	}
	if p, err := newPiPlatform(manifest.Platforms.Pi); err == nil && p.Detect() {
		platforms = append(platforms, p)
	}

	return platforms
}

// detectedSet returns a map of platform ID → detected.
func detectedSet(platforms []Platform) map[string]bool {
	m := make(map[string]bool)
	for _, p := range platforms {
		m[p.ID()] = true
	}
	return m
}

// platformDisplayName returns the human-readable name for a platform ID.
func platformDisplayName(id string) string {
	switch id {
	case "opencode":
		return "opencode"
	case "qwen":
		return "Qwen Code"
	case "copilot":
		return "GitHub Copilot"
	case "claude":
		return "Claude Code"
	case "pi":
		return "Pi"
	default:
		return id
	}
}

// platformHome returns the home directory for a platform from the manifest config.
func platformHome(id string, manifest DistManifest) string {
	var cfg PlatformConfig
	switch id {
	case "opencode":
		cfg = manifest.Platforms.OpenCode
	case "qwen":
		cfg = manifest.Platforms.Qwen
	case "copilot":
		cfg = manifest.Platforms.Copilot
	case "claude":
		cfg = manifest.Platforms.Claude
	case "pi":
		cfg = manifest.Platforms.Pi
	default:
		return ""
	}
	home, err := resolveHome(cfg.SkillRoot)
	if err != nil {
		return ""
	}
	return home
}

// platformMissingReason returns a human-readable reason why a platform wasn't detected.
func platformMissingReason(id string) string {
	switch id {
	case "opencode":
		return "opencode.json missing"
	case "qwen":
		return "~/.qwen missing"
	case "copilot":
		return "~/.copilot missing or 'code' not in PATH"
	case "claude":
		return "~/.claude missing"
	case "pi":
		return "~/.pi/agent missing and 'pi' not in PATH"
	default:
		return ""
	}
}

// checkAlreadyInstalled returns role IDs that are already installed on a platform.
func checkAlreadyInstalled(manifest DistManifest, plat Platform) []string {
	if plat.ID() == "opencode" {
		ids, err := plat.GetAgentIDs(manifest, nil)
		if err != nil {
			warn("opencode.json is not valid JSON — cannot detect existing agents.")
			return nil
		}
		return ids
	}

	agentsDir := plat.AgentsDir()
	if agentsDir == "" {
		return nil
	}

	var existing []string
	for _, role := range manifest.Roles {
		var agentPath string
		switch plat.ID() {
		case "qwen":
			agentPath = filepath.Join(agentsDir, role.ID+".md")
		case "copilot":
			agentPath = filepath.Join(agentsDir, role.ID+".agent.md")
		case "claude":
			agentPath = filepath.Join(agentsDir, role.ID+".md")
		}
		if agentPath != "" {
			if _, err := os.Stat(agentPath); err == nil {
				existing = append(existing, role.ID)
			}
		}
	}
	return existing
}

// ---------------------------------------------------------------------------
// Bundle validation
// ---------------------------------------------------------------------------

// ValidateBundle checks that all files referenced in the manifest exist in the in-memory bundle.
// Returns missing relative paths.
func ValidateBundle(bundle Bundle) []string {
	var missing []string
	manifest := bundle.Manifest

	for _, role := range manifest.Roles {
		for _, file := range []string{
			role.PromptFiles.OpenCode,
			role.PromptFiles.Qwen,
			role.PromptFiles.Copilot,
			role.PromptFiles.Claude,
			role.PromptFiles.Pi,
			role.AgentFiles.Qwen,
			role.AgentFiles.Copilot,
			role.AgentFiles.Claude,
		} {
			if file == "" {
				continue
			}
			if _, ok := bundle.Files[filepath.ToSlash(file)]; !ok {
				missing = append(missing, file)
			}
		}
	}

	for _, cmd := range manifest.Commands {
		if _, ok := bundle.Files[filepath.ToSlash(cmd.File)]; !ok {
			missing = append(missing, cmd.File)
		}
	}

	for _, skill := range manifest.Skills {
		prefix := filepath.ToSlash(filepath.Join("skills", skill)) + "/"
		found := false
		for rel := range bundle.Files {
			if strings.HasPrefix(rel, prefix) {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, filepath.Join("skills", skill))
		}
	}

	return missing
}

func summarizeMissingArtifacts(missing []string) string {
	if len(missing) == 0 {
		return "none"
	}
	if len(missing) <= 3 {
		return strings.Join(missing, ", ")
	}
	return fmt.Sprintf("%s, %s, %s (+%d more)", missing[0], missing[1], missing[2], len(missing)-3)
}

// ---------------------------------------------------------------------------
// Legacy command sweep
// ---------------------------------------------------------------------------

// sweepLegacyCommands removes any legacy opencode command files (bare-name IDs
// from prior installs) from the platform's commands directory. It is idempotent:
// absent files are silently skipped via removeFileIfExists.
func sweepLegacyCommands(homeDir string, legacyIDs []string) {
	cmdsDir := filepath.Join(homeDir, "commands")
	for _, id := range legacyIDs {
		removeFileIfExists(filepath.Join(cmdsDir, id+".md"), "legacy command: /"+id)
	}
}

// ---------------------------------------------------------------------------
// Non-interactive install (used by tests and interactive flow)
// ---------------------------------------------------------------------------

// InstallToPlatformWithReporter installs all artifacts from an in-memory bundle
// to a single platform and routes all user-visible output through the supplied Reporter.
// This is the canonical implementation; installToPlatform delegates to it.
//
// The optional globalMode variadic argument (0 or 1 values accepted) provides the
// resolved global documentation mode string (e.g. "vault" or "in-project") that
// is substituted for the __DOC_AGENT_GLOBAL_MODE__ placeholder in installed files.
// When omitted, the mode defaults to "vault" to preserve pre-v4 install behaviour.
func InstallToPlatformWithReporter(manifest DistManifest, bundle Bundle, plat Platform, basePath string, r Reporter, globalMode ...string) error {
	// Resolve global mode: first explicit argument wins, otherwise default to vault.
	resolvedGlobalMode := "vault"
	if len(globalMode) > 0 && globalMode[0] != "" {
		resolvedGlobalMode = globalMode[0]
	}

	// Ordered placeholder substitution applied to every installed file.
	// Order is significant: BASE_PATH is replaced first so that any remaining
	// __DOC_AGENT_BASE_PATH__-prefixed tokens (e.g. in path-resolution preamble)
	// are already resolved before the mode token runs.
	installPlaceholders := []placeholderPair{
		{manifest.PlaceholderBasePath, basePath},                         // __DOC_AGENT_BASE_PATH__/
		{"__DOC_AGENT_GLOBAL_MODE__", resolvedGlobalMode},                // vault | in-project
		{"__DOC_AGENT_GLOBAL_BASE__", strings.TrimSuffix(basePath, "/")}, // vault base without trailing slash
	}

	// --- Copy skills ---
	// Build a set of conditional (in-project-only) skill IDs so the loop below
	// can skip them when the resolved mode is not in-project.
	conditionalSkillSet := make(map[string]bool, len(manifest.ConditionalSkills))
	for _, id := range manifest.ConditionalSkills {
		conditionalSkillSet[id] = true
	}

	skillsDir := plat.SkillsDir()
	for _, skill := range manifest.Skills {
		// Skip conditional (in-project-only) skills when the resolved mode is vault.
		if conditionalSkillSet[skill] && resolvedGlobalMode != string(configpkg.ModeInProject) {
			continue
		}
		dstDir := filepath.Join(skillsDir, skill)
		if err := writeBundleDir(bundle, filepath.ToSlash(filepath.Join("skills", skill)), dstDir); err != nil {
			return fmt.Errorf("copy skills/%s: %w", skill, err)
		}
		r.Ok("skill: " + skill)
	}

	// --- Copy prompts ---
	promptsDir := plat.PromptsDir()
	if err := ensureDir(promptsDir); err != nil {
		return fmt.Errorf("create prompts dir: %w", err)
	}
	for _, role := range manifest.Roles {
		promptFile := promptFileFor(plat.ID(), role)
		if promptFile == "" {
			continue
		}
		dst := filepath.Join(promptsDir, filepath.Base(promptFile))
		if err := writeBundleFile(bundle, promptFile, dst); err != nil {
			return fmt.Errorf("copy prompt %s: %w", role.ID, err)
		}
		if err := replaceAllInFile(dst, installPlaceholders); err != nil {
			return fmt.Errorf("replace placeholder in %s: %w", dst, err)
		}
		r.Ok("prompt: " + filepath.Base(dst))
	}

	// --- Copy agents (non-opencode platforms) ---
	agentsDir := plat.AgentsDir()
	if agentsDir != "" {
		if err := ensureDir(agentsDir); err != nil {
			return fmt.Errorf("create agents dir: %w", err)
		}
		for _, role := range manifest.Roles {
			agentFile := agentFileFor(plat.ID(), role)
			if agentFile == "" {
				continue
			}
			dst := filepath.Join(agentsDir, filepath.Base(agentFile))
			if err := writeBundleFile(bundle, agentFile, dst); err != nil {
				return fmt.Errorf("copy agent %s: %w", role.ID, err)
			}
			if err := replaceAllInFile(dst, installPlaceholders); err != nil {
				return fmt.Errorf("replace placeholder in %s: %w", dst, err)
			}
			r.Ok("agent: " + filepath.Base(dst))
		}
	}

	// --- Copy commands (opencode only) ---
	if plat.ID() == "opencode" {
		cmdsDir := filepath.Join(plat.HomeDir(), "commands")
		if err := ensureDir(cmdsDir); err != nil {
			return fmt.Errorf("create commands dir: %w", err)
		}
		for _, cmd := range manifest.Commands {
			dst := filepath.Join(cmdsDir, filepath.Base(cmd.File))
			if err := writeBundleFile(bundle, cmd.File, dst); err != nil {
				return fmt.Errorf("copy command %s: %w", cmd.ID, err)
			}
			if err := replaceAllInFile(dst, installPlaceholders); err != nil {
				return fmt.Errorf("replace placeholder in %s: %w", dst, err)
			}
			r.Ok("command: " + filepath.Base(dst))
		}
		sweepLegacyCommands(plat.HomeDir(), manifest.LegacyCommandIds)
	}

	// --- Patch platform config (opencode only) ---
	if err := plat.PatchConfig(manifest, basePath, nil); err != nil {
		return fmt.Errorf("patch config: %w", err)
	}
	// Emit ok for each agent registered (matching install.js patchOpencodeJson)
	if plat.ID() == "opencode" {
		for _, role := range manifest.Roles {
			r.Ok("agent registered: " + role.ID)
		}
	}

	// --- Write skill registry (opencode + claude only) ---
	if err := plat.WriteSkillRegistry(basePath); err != nil {
		return fmt.Errorf("write skill registry: %w", err)
	}
	if plat.SkillRegistryTrigger() != "" {
		r.Ok("skill-registry.md written")
	}

	return nil
}

// InstallToPlatform installs all artifacts from an in-memory bundle to a single
// platform using the default stdout Reporter.
//
// The optional globalMode variadic argument behaves identically to
// InstallToPlatformWithReporter.
func InstallToPlatform(manifest DistManifest, bundle Bundle, plat Platform, basePath string, globalMode ...string) error {
	return InstallToPlatformWithReporter(manifest, bundle, plat, basePath, defaultReporter, globalMode...)
}

func writeBundleFile(bundle Bundle, rel, dst string) error {
	data, ok := bundle.Files[filepath.ToSlash(rel)]
	if !ok {
		return fmt.Errorf("bundle file not found: %s", rel)
	}
	if err := ensureDir(filepath.Dir(dst)); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func writeBundleDir(bundle Bundle, relPrefix, dst string) error {
	prefix := strings.TrimSuffix(filepath.ToSlash(relPrefix), "/") + "/"
	var rels []string
	for rel := range bundle.Files {
		if strings.HasPrefix(rel, prefix) {
			rels = append(rels, rel)
		}
	}
	if len(rels) == 0 {
		return fmt.Errorf("bundle directory not found: %s", relPrefix)
	}
	sort.Strings(rels)
	for _, rel := range rels {
		suffix := strings.TrimPrefix(rel, prefix)
		if err := writeBundleFile(bundle, rel, filepath.Join(dst, filepath.FromSlash(suffix))); err != nil {
			return err
		}
	}
	return nil
}

// promptFileFor returns the prompt file path from a DistRole for a given platform ID.
func promptFileFor(platformID string, role DistRole) string {
	switch platformID {
	case "opencode":
		return role.PromptFiles.OpenCode
	case "qwen":
		return role.PromptFiles.Qwen
	case "copilot":
		return role.PromptFiles.Copilot
	case "claude":
		return role.PromptFiles.Claude
	case "pi":
		return role.PromptFiles.Pi
	default:
		return ""
	}
}

// agentFileFor returns the agent file path from a DistRole for a given platform ID.
func agentFileFor(platformID string, role DistRole) string {
	switch platformID {
	case "qwen":
		return role.AgentFiles.Qwen
	case "copilot":
		return role.AgentFiles.Copilot
	case "claude":
		return role.AgentFiles.Claude
	default:
		return ""
	}
}
