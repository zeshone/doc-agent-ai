package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ---------------------------------------------------------------------------
// Uninstall output helpers (matching uninstall.js)
// ---------------------------------------------------------------------------

// skip prints a dim "not found, skipped" message for missing artifacts.
func skipMsg(msg string) {
	fmt.Printf("  %s–%s %s %s(not found, skipped)%s\n", ansiGray, ansiReset, msg, ansiGray, ansiReset)
}

// subInfo prints an indented info line (used in uninstall summary).
func subInfo(msg string) {
	fmt.Printf("    %s→%s %s\n", ansiBlue, ansiReset, msg)
}

// ---------------------------------------------------------------------------
// What-is-installed data
// ---------------------------------------------------------------------------

// installedDetails describes what doc-agent-ai artifacts are present
// on a single platform.
type installedDetails struct {
	platform Platform
	skills   []string
	prompts  []string
	agents   []string
	commands []string
	registry bool
}

// hasAny returns true if any artifact is present.
func (d *installedDetails) hasAny() bool {
	return len(d.skills) > 0 || len(d.prompts) > 0 || len(d.agents) > 0 || len(d.commands) > 0 || d.registry
}

// checkWhatIsInstalled scans every detected platform and returns only
// those that have at least one doc-agent-ai artifact.
func checkWhatIsInstalled(manifest DistManifest, platforms []Platform) []installedDetails {
	var result []installedDetails

	for _, p := range platforms {
		details := installedDetails{platform: p}

		details.skills = p.GetSkillIDs(manifest)

		promptIDs, err := p.GetPromptIDs(manifest, nil)
		if err == nil {
			details.prompts = promptIDs
		}

		agentIDs, err := p.GetAgentIDs(manifest, nil)
		if err == nil {
			details.agents = agentIDs
		} else {
			warn("Cannot read agent config — detection skipped for " + platformDisplayName(p.ID()) + ".")
		}

		if p.ID() == "opencode" {
			cmdIDs, err := p.GetCommandIDs(manifest)
			if err == nil {
				details.commands = cmdIDs
			}
		}

		// Check registry on opencode and claude
		if p.ID() == "opencode" || p.ID() == "claude" {
			registryPath := filepath.Join(p.HomeDir(), ".atl", "skill-registry.md")
			if _, err := os.Stat(registryPath); err == nil {
				details.registry = true
			}
		}

		if details.hasAny() {
			result = append(result, details)
		}
	}

	return result
}

// ---------------------------------------------------------------------------
// Removal helpers
// ---------------------------------------------------------------------------

// removeDirIfExists removes dirPath if it exists. Prints ok or skip.
func removeDirIfExists(dirPath, label string) {
	if _, err := os.Stat(dirPath); err == nil {
		if err := os.RemoveAll(dirPath); err != nil {
			errOut("failed to remove " + label + ": " + err.Error())
			return
		}
		ok("removed: " + label)
	} else {
		skipMsg(label)
	}
}

// removeFileIfExists removes filePath if it exists. Prints ok or skip.
func removeFileIfExists(filePath, label string) {
	if _, err := os.Stat(filePath); err == nil {
		if err := os.Remove(filePath); err != nil {
			errOut("failed to remove " + label + ": " + err.Error())
			return
		}
		ok("removed: " + label)
	} else {
		skipMsg(label)
	}
}

// ---------------------------------------------------------------------------
// Artifact removal per platform
// ---------------------------------------------------------------------------

// removeSkillsForPlatform removes skill directories and prunes
// empty parent directories up to (but not including) the platform home.
func removeSkillsForPlatform(plat Platform, skillIDs []string) {
	skillsDir := plat.SkillsDir()
	for _, skillID := range skillIDs {
		skillDir := filepath.Join(skillsDir, skillID)
		removeDirIfExists(skillDir, "skill: "+skillID)
	}
	pruneEmptyDirs(skillsDir, plat.HomeDir())
}

// removePromptFilesForPlatform removes prompt .md files for the given roles
// and prunes empty prompt directories.
func removePromptFilesForPlatform(plat Platform, promptIDs []string, manifest DistManifest) {
	promptsDir := plat.PromptsDir()
	roleSet := make(map[string]bool, len(promptIDs))
	for _, id := range promptIDs {
		roleSet[id] = true
	}

	for _, role := range manifest.Roles {
		if !roleSet[role.ID] {
			continue
		}
		promptFile := promptFileFor(plat.ID(), role)
		if promptFile == "" {
			continue
		}
		base := filepath.Base(promptFile)
		filePath := filepath.Join(promptsDir, base)
		removeFileIfExists(filePath, "prompt: "+base)
	}

	pruneEmptyDirs(promptsDir, plat.HomeDir())
}

// removeCommandFiles removes command .md files (opencode only) and prunes empty dirs.
func removeCommandFiles(plat Platform, commandIDs []string, manifest DistManifest) {
	cmdsDir := filepath.Join(plat.HomeDir(), "commands")
	cmdSet := make(map[string]bool, len(commandIDs))
	for _, id := range commandIDs {
		cmdSet[id] = true
	}

	for _, cmd := range manifest.Commands {
		if !cmdSet[cmd.ID] {
			continue
		}
		base := filepath.Base(cmd.File)
		filePath := filepath.Join(cmdsDir, base)
		removeFileIfExists(filePath, "command: /"+cmd.ID)
	}

	pruneEmptyDirs(cmdsDir, plat.HomeDir())
}

// removeAgentFilesForPlatform removes agent files for non-opencode platforms.
func removeAgentFilesForPlatform(plat Platform, agentIDs []string, manifest DistManifest) {
	agentsDir := plat.AgentsDir()
	roleSet := make(map[string]bool, len(agentIDs))
	for _, id := range agentIDs {
		roleSet[id] = true
	}

	for _, role := range manifest.Roles {
		if !roleSet[role.ID] {
			continue
		}
		agentFile := agentFileFor(plat.ID(), role)
		if agentFile == "" {
			continue
		}
		base := filepath.Base(agentFile)
		filePath := filepath.Join(agentsDir, base)
		removeFileIfExists(filePath, "agent: "+base)
	}

	pruneEmptyDirs(agentsDir, plat.HomeDir())
}

// removeSkillRegistryForPlatform removes .atl/skill-registry.md and prunes empty dirs.
func removeSkillRegistryForPlatform(plat Platform) {
	registryPath := filepath.Join(plat.HomeDir(), ".atl", "skill-registry.md")
	removeFileIfExists(registryPath, ".atl/skill-registry.md")
	registryDir := filepath.Dir(registryPath)
	pruneEmptyDirs(registryDir, plat.HomeDir())
}

// uninstallPlatform removes all detected artifacts from a single platform.
func uninstallPlatform(details installedDetails, manifest DistManifest) {
	plat := details.platform
	head("Removing from " + plat.ID() + "...")

	if len(details.skills) > 0 {
		removeSkillsForPlatform(plat, details.skills)
	}
	if len(details.prompts) > 0 {
		removePromptFilesForPlatform(plat, details.prompts, manifest)
	}

	if plat.ID() == "opencode" {
		if len(details.commands) > 0 {
			removeCommandFiles(plat, details.commands, manifest)
		}
		sweepLegacyCommands(plat.HomeDir(), manifest.LegacyCommandIds)
		if len(details.agents) > 0 {
			if err := plat.RemoveConfig(manifest, nil); err != nil {
				errOut("Failed to clean opencode.json: " + err.Error())
			} else {
				for _, id := range details.agents {
					ok("agent removed: " + id)
				}
			}
		}
		if details.registry {
			removeSkillRegistryForPlatform(plat)
		}
		return
	}

	if len(details.agents) > 0 {
		removeAgentFilesForPlatform(plat, details.agents, manifest)
	}
	if details.registry {
		removeSkillRegistryForPlatform(plat)
	}
}

// ---------------------------------------------------------------------------
// Interactive uninstall flow
// ---------------------------------------------------------------------------

// uninstallInteractive runs the full interactive uninstall command.
// It mirrors uninstall.js exactly: banner → detect → check installed →
// show summary → confirm → remove → done.
func uninstallInteractive() error {
	// Banner
	fmt.Println()
	fmt.Printf("%s%s  doc-agent-ai%s %sv%s — uninstaller%s\n", ansiBold, ansiCyan, ansiReset, ansiGray, version, ansiReset)
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

	// Step 1: Read manifest
	manifest, err := readManifestFrom(distDir)
	if err != nil {
		errOut("dist/manifest.json is missing or invalid JSON.")
		errOut("Run `doc-agent-ai generate` before uninstalling.")
		return err
	}

	// Step 2: Detect platforms
	head("Detecting platforms...")
	allPlatforms := detectAllPlatforms(manifest)
	detected := detectedSet(allPlatforms)

	for _, id := range []string{"opencode", "qwen", "copilot", "claude"} {
		if detected[id] {
			home := platformHome(id, manifest)
			ok(platformDisplayName(id) + " detected  " + ansiGray + "(" + home + ")" + ansiReset)
		} else {
			warn(platformDisplayName(id) + " not found  " + ansiGray + "(" + platformMissingReason(id) + ")" + ansiReset)
		}
	}

	if len(allPlatforms) == 0 {
		fmt.Println()
		warn("No supported platform detected.")
		info("Nothing to uninstall.")
		return nil
	}

	// Step 3: Check what's installed
	installed := checkWhatIsInstalled(manifest, allPlatforms)

	if len(installed) == 0 {
		fmt.Println()
		warn("doc-agent-ai does not appear to be installed on detected platforms.")
		info("Nothing to uninstall.")
		return nil
	}

	// Step 4: Show what will be removed
	head("The following will be removed:")
	fmt.Printf("%s  ─────────────────────────────────%s\n", ansiGray, ansiReset)

	for _, details := range installed {
		platID := details.platform.ID()
		fmt.Printf("  %s:\n", platID)
		if len(details.skills) > 0 {
			subInfo("Skills: " + strings.Join(details.skills, ", "))
		}
		if len(details.prompts) > 0 {
			relPrompts := fmt.Sprintf("prompts/%s/", "doc")
			subInfo("Prompts: " + relPrompts)
		}
		if len(details.commands) > 0 {
			cmdNames := make([]string, len(details.commands))
			for i, id := range details.commands {
				cmdNames[i] = "/" + id
			}
			subInfo("Commands: " + strings.Join(cmdNames, ", "))
		}
		if len(details.agents) > 0 {
			subInfo("Agents: " + strings.Join(details.agents, ", "))
		}
		if details.registry {
			subInfo("Registry: .atl/skill-registry.md")
		}
	}

	fmt.Println()
	warn("Your documentation files are NOT affected.")
	fmt.Println()

	// Step 5: Confirm
	scanner := bufio.NewScanner(os.Stdin)
	confirm := ask(scanner, fmt.Sprintf("  %s%sUninstall from all detected platforms?%s (y/N) ", ansiBold, ansiRed, ansiReset))
	if strings.TrimSpace(strings.ToLower(confirm)) != "y" {
		info("Uninstall cancelled.")
		return nil
	}

	// Step 6: Remove
	for _, details := range installed {
		uninstallPlatform(details, manifest)
	}

	fmt.Println()
	fmt.Printf("%s%s  ✔ Uninstall complete.%s\n", ansiBold, ansiGreen, ansiReset)
	dim("Restart your AI tool if it is currently running.")
	fmt.Println()

	return nil
}
