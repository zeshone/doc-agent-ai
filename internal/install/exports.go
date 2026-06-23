package install

import configpkg "github.com/zeshone/doc-agent-ai/internal/config"

func NewStdoutReporter() Reporter                           { return newStdoutReporter() }
func NewOpenCodePlatform(cfg PlatformConfig) (Platform, error) { return newOpenCodePlatform(cfg) }
func NewCopilotPlatform(cfg PlatformConfig) (Platform, error) { return newCopilotPlatform(cfg) }
func NewPiPlatform(cfg PlatformConfig) (Platform, error)      { return newPiPlatform(cfg) }
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
func RunModeSwitchHookWithPlatforms(plan configpkg.InstallPlan, platforms []Platform, r Reporter) {
	runModeSwitchHookWithPlatforms(plan, platforms, r)
}
func SweepDocReaderIfLeavingInProject(platforms []Platform, r Reporter) {
	sweepDocReaderIfLeavingInProject(platforms, r)
}
func RemovePromptFilesForPlatform(plat Platform, promptIDs []string, manifest DistManifest) {
	removePromptFilesForPlatform(plat, promptIDs, manifest)
}
func UninstallPlatform(details InstalledDetails, manifest DistManifest) { uninstallPlatform(details, manifest) }
func SummarizeMissingArtifacts(missing []string) string { return summarizeMissingArtifacts(missing) }
