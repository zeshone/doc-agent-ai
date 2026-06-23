package install

func NewStdoutReporter() Reporter                           { return newStdoutReporter() }
func RegistryTemplate(basePath, skillsDir, triggerStyle string) string {
	return registryTemplate(basePath, skillsDir, triggerStyle)
}
func DetectAllPlatforms(manifest DistManifest) []Platform    { return detectAllPlatforms(manifest) }
func PlatformDisplayName(id string) string                   { return platformDisplayName(id) }
func CheckAlreadyInstalled(manifest DistManifest, plat Platform) []string {
	return checkAlreadyInstalled(manifest, plat)
}
func CheckWhatIsInstalled(manifest DistManifest, platforms []Platform) []InstalledDetails {
	return checkWhatIsInstalled(manifest, platforms)
}
func UninstallInteractive(manifest DistManifest) error { return uninstallInteractive(manifest) }
func ValidateBundleExport(bundle Bundle) []string { return ValidateBundle(bundle) }
func ExecuteInstallExport(bundle Bundle, plan InstallPlan, allPlatforms []Platform, r Reporter) error {
	return ExecuteInstall(bundle, plan, allPlatforms, r)
}
func RunModeSwitchHookWithPlatforms(plan InstallPlan, platforms []Platform, r Reporter) {
	runModeSwitchHookWithPlatforms(plan, platforms, r)
}
func SweepDocReaderIfLeavingInProject(platforms []Platform, r Reporter) {
	sweepDocReaderIfLeavingInProject(platforms, r)
}
func RemovePromptFilesForPlatform(plat Platform, promptIDs []string, manifest DistManifest) {
	removePromptFilesForPlatform(plat, promptIDs, manifest)
}
func UninstallPlatform(details InstalledDetails, manifest DistManifest) { uninstallPlatform(details, manifest) }
func NewPlatformForTest(id string, homeDir string) Platform {
	cfg := PlatformConfig{SkillRoot: homeDir + "/skills", PromptDir: "prompts"}
	switch id {
	case "opencode":
		cfg.CommandDir = "commands"
		return &opencodePlatform{basePlatform{id: id, homeDir: homeDir, cfg: cfg}}
	case "qwen":
		cfg.AgentDir = "agents"
		return &qwenPlatform{basePlatform{id: id, homeDir: homeDir, cfg: cfg}}
	case "copilot":
		cfg.AgentDir = "agents"
		return &copilotPlatform{basePlatform{id: id, homeDir: homeDir, cfg: cfg}}
	case "claude":
		cfg.AgentDir = "agents"
		return &claudePlatform{basePlatform{id: id, homeDir: homeDir, cfg: cfg}}
	case "pi":
		return &piPlatform{basePlatform{id: id, homeDir: homeDir, cfg: cfg}}
	default:
		return nil
	}
}
func SummarizeMissingArtifacts(missing []string) string { return summarizeMissingArtifacts(missing) }
