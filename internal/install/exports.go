package install

func ReadManifestFrom(distDir string) (DistManifest, error) { return readManifestFrom(distDir) }
func NewStdoutReporter() Reporter                           { return newStdoutReporter() }
func NewOpenCodePlatform(cfg PlatformConfig) (Platform, error) { return newOpenCodePlatform(cfg) }
func NewCopilotPlatform(cfg PlatformConfig) (Platform, error) { return newCopilotPlatform(cfg) }
func NewPiPlatform(cfg PlatformConfig) (Platform, error)      { return newPiPlatform(cfg) }
func ResolveHome(path string) (string, error)                 { return resolveHome(path) }
func RegistryTemplate(basePath, skillsDir, triggerStyle string) string {
	return registryTemplate(basePath, skillsDir, triggerStyle)
}
func PromptFileFor(platformID string, role DistRole) string { return promptFileFor(platformID, role) }
func AgentFileFor(platformID string, role DistRole) string  { return agentFileFor(platformID, role) }
func DetectAllPlatforms(manifest DistManifest) []Platform    { return detectAllPlatforms(manifest) }
func DetectedSet(platforms []Platform) map[string]bool       { return detectedSet(platforms) }
func PlatformDisplayName(id string) string                   { return platformDisplayName(id) }
func PlatformHome(id string, manifest DistManifest) string   { return platformHome(id, manifest) }
func PlatformMissingReason(id string) string                 { return platformMissingReason(id) }
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
func PruneEmptyDirs(startDir, stopDir string) error { return pruneEmptyDirs(startDir, stopDir) }
func BundleFromDistDir(manifest DistManifest, distDir string) (Bundle, error) { return bundleFromDistDir(manifest, distDir) }
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
