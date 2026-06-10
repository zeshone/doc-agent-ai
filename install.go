package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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

// copyFile copies a single file from src to dst. Creates parent directories.
func copyFile(src, dst string) error {
	if err := ensureDir(filepath.Dir(dst)); err != nil {
		return err
	}
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// copyDir recursively copies a directory from src to dst. Mirrors install.js
// copyDirSync: iterates entries, recurses on subdirs, copies files.
func copyDir(src, dst string) error {
	if err := ensureDir(dst); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// replaceInFile replaces all occurrences of search with replace in file.
// If search is not found, the file is left unchanged.
func replaceInFile(path, search, replace string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)
	if !strings.Contains(content, search) {
		return nil
	}
	content = strings.ReplaceAll(content, search, replace)
	return os.WriteFile(path, []byte(content), 0644)
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
// Manifest reading
// ---------------------------------------------------------------------------

// readManifestFrom reads and parses dist/manifest.json from the given dist dir.
func readManifestFrom(distDir string) (DistManifest, error) {
	var manifest DistManifest
	data, err := os.ReadFile(filepath.Join(distDir, "manifest.json"))
	if err != nil {
		return manifest, fmt.Errorf("read %s/manifest.json: %w", distDir, err)
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return manifest, fmt.Errorf("parse %s/manifest.json: %w", distDir, err)
	}
	return manifest, nil
}

// ---------------------------------------------------------------------------
// Path normalization
// ---------------------------------------------------------------------------

// normalizeBasePath ensures the base path uses forward slashes and ends with "/".
// An empty or whitespace-only input defaults to the current working directory.
func normalizeBasePath(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		trimmed, _ = os.Getwd()
	}
	normalized := filepath.ToSlash(filepath.Clean(trimmed))
	if !strings.HasSuffix(normalized, "/") {
		normalized += "/"
	}
	return normalized
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
// Dist validation
// ---------------------------------------------------------------------------

// validateDist checks that all files referenced in the manifest exist in distDir.
// Returns a list of missing paths (relative to distDir).
func validateDist(manifest DistManifest, distDir string) []string {
	var missing []string

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
			path := filepath.Join(distDir, filepath.ToSlash(file))
			if _, err := os.Stat(path); os.IsNotExist(err) {
				missing = append(missing, file)
			}
		}
	}

	for _, cmd := range manifest.Commands {
		path := filepath.Join(distDir, filepath.ToSlash(cmd.File))
		if _, err := os.Stat(path); os.IsNotExist(err) {
			missing = append(missing, cmd.File)
		}
	}

	for _, skill := range manifest.Skills {
		skillDir := filepath.Join(distDir, "skills", skill)
		if _, err := os.Stat(skillDir); os.IsNotExist(err) {
			missing = append(missing, filepath.Join("skills", skill))
		}
	}

	return missing
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

// installToPlatform installs all artifacts from distDir to a single platform.
// This is the non-interactive core — tests can call it directly.
//
// The optional globalMode variadic argument (0 or 1 values accepted) provides the
// resolved global documentation mode string (e.g. "vault" or "in-project") that
// is substituted for the __DOC_AGENT_GLOBAL_MODE__ placeholder in installed files.
// When omitted, the mode defaults to "vault" to preserve pre-v4 install behaviour.
func installToPlatform(manifest DistManifest, plat Platform, basePath, distDir string, globalMode ...string) error {
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
		{"__DOC_AGENT_GLOBAL_BASE__", strings.TrimSuffix(basePath, "/")}, // vault base without trailing slash (preamble prose)
	}

	// --- Copy skills ---
	skillsDir := plat.SkillsDir()
	for _, skill := range manifest.Skills {
		srcDir := filepath.Join(distDir, "skills", skill)
		dstDir := filepath.Join(skillsDir, skill)
		if err := copyDir(srcDir, dstDir); err != nil {
			return fmt.Errorf("copy skills/%s: %w", skill, err)
		}
		ok("skill: " + skill)
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
		src := filepath.Join(distDir, filepath.ToSlash(promptFile))
		dst := filepath.Join(promptsDir, filepath.Base(promptFile))
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("copy prompt %s: %w", role.ID, err)
		}
		if err := replaceAllInFile(dst, installPlaceholders); err != nil {
			return fmt.Errorf("replace placeholder in %s: %w", dst, err)
		}
		ok("prompt: " + filepath.Base(dst))
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
			src := filepath.Join(distDir, filepath.ToSlash(agentFile))
			dst := filepath.Join(agentsDir, filepath.Base(agentFile))
			if err := copyFile(src, dst); err != nil {
				return fmt.Errorf("copy agent %s: %w", role.ID, err)
			}
			if err := replaceAllInFile(dst, installPlaceholders); err != nil {
				return fmt.Errorf("replace placeholder in %s: %w", dst, err)
			}
			ok("agent: " + filepath.Base(dst))
		}
	}

	// --- Copy commands (opencode only) ---
	if plat.ID() == "opencode" {
		cmdsDir := filepath.Join(plat.HomeDir(), "commands")
		if err := ensureDir(cmdsDir); err != nil {
			return fmt.Errorf("create commands dir: %w", err)
		}
		for _, cmd := range manifest.Commands {
			src := filepath.Join(distDir, filepath.ToSlash(cmd.File))
			dst := filepath.Join(cmdsDir, filepath.Base(cmd.File))
			if err := copyFile(src, dst); err != nil {
				return fmt.Errorf("copy command %s: %w", cmd.ID, err)
			}
			if err := replaceAllInFile(dst, installPlaceholders); err != nil {
				return fmt.Errorf("replace placeholder in %s: %w", dst, err)
			}
			ok("command: " + filepath.Base(dst))
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
			ok("agent registered: " + role.ID)
		}
	}

	// --- Write skill registry (opencode + claude only) ---
	if err := plat.WriteSkillRegistry(basePath); err != nil {
		return fmt.Errorf("write skill registry: %w", err)
	}
	if plat.SkillRegistryTrigger() != "" {
		ok("skill-registry.md written")
	}

	return nil
}

// installPlatforms installs to multiple platforms non-interactively.
// This is the entry point for tests.
func installPlatforms(manifest DistManifest, platforms []Platform, basePath, distDir string) error {
	for _, plat := range platforms {
		head("Installing for " + platformDisplayName(plat.ID()) + "...")
		if err := installToPlatform(manifest, plat, basePath, distDir); err != nil {
			return fmt.Errorf("install to %s: %w", plat.ID(), err)
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

// ---------------------------------------------------------------------------
// Interactive install flow
// ---------------------------------------------------------------------------

// installInteractive runs the full interactive install command.
func installInteractive() error {
	// Banner
	fmt.Println()
	fmt.Printf("%s%s  doc-agent-ai%s %sv%s%s\n", ansiBold, ansiCyan, ansiReset, ansiGray, version, ansiReset)
	fmt.Printf("%s  Documentation workflow agent installer%s\n", ansiGray, ansiReset)
	fmt.Println()

	// Step 0: Ensure dist/ exists (auto-generate if missing)
	distDir := "dist"
	if _, err := os.Stat(filepath.Join(distDir, "manifest.json")); os.IsNotExist(err) {
		info("Auto-generating dist/ from embedded source...")
		if err := generate(distDir); err != nil {
			return fmt.Errorf("auto-generate dist: %w", err)
		}
		ok("dist/ generated")
	}

	// Step 1: Read and validate manifest
	manifest, err := readManifestFrom(distDir)
	if err != nil {
		errOut("dist/manifest.json is missing or invalid JSON.")
		errOut("Run `doc-agent-ai generate` before installing.")
		return err
	}

	if missing := validateDist(manifest, distDir); len(missing) > 0 {
		errOut("dist/ is incomplete. Missing generated artifacts:")
		for _, item := range missing {
			errOut("  " + item)
		}
		errOut("Run `doc-agent-ai generate` and try again.")
		return fmt.Errorf("incomplete dist: %d missing artifacts", len(missing))
	}

	// Step 2: Detect platforms
	head("Detecting platforms...")
	allPlatforms := detectAllPlatforms(manifest)
	detected := detectedSet(allPlatforms)

	for _, id := range []string{"opencode", "qwen", "copilot", "claude", "pi"} {
		if detected[id] {
			home := platformHome(id, manifest)
			ok(platformDisplayName(id) + " detected  " + ansiGray + "(" + home + ")" + ansiReset)
		} else {
			warn(platformDisplayName(id) + " not found  " + ansiGray + "(" + platformMissingReason(id) + ")" + ansiReset)
		}
	}

	if len(allPlatforms) == 0 {
		fmt.Println()
		errOut("No supported platform detected.")
		errOut("Install opencode, Qwen Code, GitHub Copilot, or Claude Code before running this installer.")
		return fmt.Errorf("no platforms detected")
	}

	scanner := bufio.NewScanner(os.Stdin)

	// Step 3: Platform selection (if multiple detected)
	selected := allPlatforms
	if len(allPlatforms) > 1 {
		head("Platform selection")
		dim("Multiple platforms detected. Choose which to install:")
		idx := 1
		labelMap := make(map[int]int) // selection number → index in allPlatforms
		for i, p := range allPlatforms {
			fmt.Printf("%s  [%d] %s only%s\n", ansiGray, idx, platformDisplayName(p.ID()), ansiReset)
			labelMap[idx] = i
			idx++
		}
		fmt.Printf("%s  [%d] All (default)%s\n", ansiGray, idx, ansiReset)
		fmt.Println()

		choice := ask(scanner, fmt.Sprintf("  Selection %s(Enter = all)%s: ", ansiGray, ansiReset))
		var sel int
		fmt.Sscanf(strings.TrimSpace(choice), "%d", &sel)

		if i, ok := labelMap[sel]; ok {
			selected = []Platform{allPlatforms[i]}
		}
		// else "all" or invalid → keep selected = allPlatforms
	}

	// Step 4: Check already installed (overwrite prompt per platform)
	var finalPlatforms []Platform
	for _, plat := range selected {
		existing := checkAlreadyInstalled(manifest, plat)
		if len(existing) > 0 {
			fmt.Println()
			warn("The following agents are already installed in " + platformDisplayName(plat.ID()) + ":")
			for _, id := range existing {
				dim("  - " + id)
			}
			answer := ask(scanner, fmt.Sprintf("\n  %sOverwrite %s installation?%s (y/N) ", ansiYellow, platformDisplayName(plat.ID()), ansiReset))
			if strings.TrimSpace(strings.ToLower(answer)) != "y" {
				info("Skipping " + platformDisplayName(plat.ID()) + ".")
				continue
			}
		}
		finalPlatforms = append(finalPlatforms, plat)
	}

	if len(finalPlatforms) == 0 {
		info("Nothing to install. Exiting.")
		return nil
	}

	// Step 5: Configuration — documentation base path
	head("Configuration")
	dim("Where should the agent save your project documentation?")
	dim("This is the root folder where all systems, PRDs and specs will be created.")
	cwd, _ := os.Getwd()
	warn("  ⚠  If you skip this, files will be saved in the current directory: " + cwd)
	fmt.Println()

	rawBase := ask(scanner, fmt.Sprintf("  Documentation path %s(press Enter to use current dir)%s: ", ansiGray, ansiReset))
	basePath := normalizeBasePath(rawBase)

	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		warn("Path does not exist yet: " + basePath)
		warn("The agent will still work — create the folder before first use.")
	}

	// Step 6: Summary + final confirmation
	fmt.Println()
	head("Ready to install")
	for _, plat := range finalPlatforms {
		info(platformDisplayName(plat.ID()) + " config:        " + plat.HomeDir())
	}
	info("projects base:          " + basePath)
	info("artifact source:        " + distDir + "/")
	fmt.Println()

	confirm := ask(scanner, fmt.Sprintf("  %sProceed?%s (Y/n) ", ansiBold, ansiReset))
	if strings.TrimSpace(strings.ToLower(confirm)) == "n" {
		info("Installation cancelled.")
		return nil
	}

	// Step 7: Install to each selected platform
	for _, plat := range finalPlatforms {
		head("Installing for " + platformDisplayName(plat.ID()) + "...")
		if err := installToPlatform(manifest, plat, basePath, distDir); err != nil {
			errOut("Failed to install to " + plat.ID() + ": " + err.Error())
			continue
		}
	}

	fmt.Println()
	fmt.Printf("%s%s  ✔ Installation complete!%s\n", ansiBold, ansiGreen, ansiReset)
	fmt.Println()

	return nil
}
