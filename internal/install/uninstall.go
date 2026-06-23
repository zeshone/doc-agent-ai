package install

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	buildpkg "github.com/zeshone/doc-agent-ai/internal/build"
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

// InstalledDetails describes what doc-agent-ai artifacts are present
// on a single platform.
type InstalledDetails struct {
	Platform Platform
	Skills   []string
	Prompts  []string
	Agents   []string
	Commands []string
	Registry bool
}

// hasAny returns true if any artifact is present.
func (d *InstalledDetails) hasAny() bool {
	return len(d.Skills) > 0 || len(d.Prompts) > 0 || len(d.Agents) > 0 || len(d.Commands) > 0 || d.Registry
}

// checkWhatIsInstalled scans every detected platform and returns only
// those that have at least one doc-agent-ai artifact.
func checkWhatIsInstalled(manifest DistManifest, platforms []Platform) []InstalledDetails {
	var result []InstalledDetails

	for _, p := range platforms {
		details := InstalledDetails{Platform: p}

		details.Skills = p.GetSkillIDs(manifest)

		promptIDs, err := p.GetPromptIDs(manifest, nil)
		if err == nil {
			details.Prompts = promptIDs
		}

		agentIDs, err := p.GetAgentIDs(manifest, nil)
		if err == nil {
			details.Agents = agentIDs
		} else {
			warn("Cannot read agent config — detection skipped for " + platformDisplayName(p.ID()) + ".")
		}

		if p.ID() == "opencode" {
			cmdIDs, err := p.GetCommandIDs(manifest)
			if err == nil {
				details.Commands = cmdIDs
			}
		}

		// Check registry on opencode and claude
		if p.ID() == "opencode" || p.ID() == "claude" {
			registryPath := filepath.Join(p.HomeDir(), ".atl", "skill-registry.md")
			if _, err := os.Stat(registryPath); err == nil {
				details.Registry = true
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
func uninstallPlatform(details InstalledDetails, manifest DistManifest) {
	plat := details.Platform
	head("Removing from " + plat.ID() + "...")

	if len(details.Skills) > 0 {
		removeSkillsForPlatform(plat, details.Skills)
	}
	if len(details.Prompts) > 0 {
		removePromptFilesForPlatform(plat, details.Prompts, manifest)
	}

	if plat.ID() == "opencode" {
		if len(details.Commands) > 0 {
			removeCommandFiles(plat, details.Commands, manifest)
		}
		sweepLegacyCommands(plat.HomeDir(), manifest.LegacyCommandIds)
		if len(details.Agents) > 0 {
			if err := plat.RemoveConfig(manifest, nil); err != nil {
				errOut("Failed to clean opencode.json: " + err.Error())
			} else {
				for _, id := range details.Agents {
					ok("agent removed: " + id)
				}
			}
		}
		if details.Registry {
			removeSkillRegistryForPlatform(plat)
		}
		return
	}

	if len(details.Agents) > 0 {
		removeAgentFilesForPlatform(plat, details.Agents, manifest)
	}
	if details.Registry {
		removeSkillRegistryForPlatform(plat)
	}
}

// ---------------------------------------------------------------------------
// Interactive uninstall flow
// ---------------------------------------------------------------------------

// uninstallInteractive runs the full interactive uninstall command.
// It mirrors uninstall.js exactly: banner → detect → check installed →
// show summary → confirm → remove → done.
func uninstallInteractive(manifest DistManifest) error {
	// Banner
	fmt.Println()
	fmt.Printf("%s%s  doc-agent-ai%s %sv%s — uninstaller%s\n", ansiBold, ansiCyan, ansiReset, ansiGray, buildpkg.Version, ansiReset)
	fmt.Println()

	// Step 1: Detect platforms
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

	// Step 2: Check what's installed
	installed := checkWhatIsInstalled(manifest, allPlatforms)

	if len(installed) == 0 {
		fmt.Println()
		warn("doc-agent-ai does not appear to be installed on detected platforms.")
		info("Nothing to uninstall.")
		return nil
	}

	// Step 3: Show what will be removed
	head("The following will be removed:")
	fmt.Printf("%s  ─────────────────────────────────%s\n", ansiGray, ansiReset)

	for _, details := range installed {
		platID := details.Platform.ID()
		fmt.Printf("  %s:\n", platID)
		if len(details.Skills) > 0 {
			subInfo("Skills: " + strings.Join(details.Skills, ", "))
		}
		if len(details.Prompts) > 0 {
			relPrompts := fmt.Sprintf("prompts/%s/", "doc")
			subInfo("Prompts: " + relPrompts)
		}
		if len(details.Commands) > 0 {
			cmdNames := make([]string, len(details.Commands))
			for i, id := range details.Commands {
				cmdNames[i] = "/" + id
			}
			subInfo("Commands: " + strings.Join(cmdNames, ", "))
		}
		if len(details.Agents) > 0 {
			subInfo("Agents: " + strings.Join(details.Agents, ", "))
		}
		if details.Registry {
			subInfo("Registry: .atl/skill-registry.md")
		}
	}

	fmt.Println()
	warn("Your documentation files are NOT affected.")
	fmt.Println()

	// Step 4: Confirm
	scanner := bufio.NewScanner(os.Stdin)
	confirm := ask(scanner, fmt.Sprintf("  %s%sUninstall from all detected platforms?%s (y/N) ", ansiBold, ansiRed, ansiReset))
	if strings.TrimSpace(strings.ToLower(confirm)) != "y" {
		info("Uninstall cancelled.")
		return nil
	}

	// Step 5: Remove
	for _, details := range installed {
		uninstallPlatform(details, manifest)
	}

	fmt.Println()
	fmt.Printf("%s%s  ✔ Uninstall complete.%s\n", ansiBold, ansiGreen, ansiReset)
	dim("Restart your AI tool if it is currently running.")
	fmt.Println()

	return nil
}
