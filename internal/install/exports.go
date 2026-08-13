package install

func NewStdoutReporter() Reporter                              { return newStdoutReporter() }
func NewOpenCodePlatform(cfg PlatformConfig) (Platform, error) { return newOpenCodePlatform(cfg) }
func RegistryTemplate(basePath, skillsDir, triggerStyle string) string {
	return registryTemplate(basePath, skillsDir, triggerStyle)
}
func DetectAllPlatforms(manifest DistManifest) []Platform { return detectAllPlatforms(manifest) }
func PlatformDisplayName(id string) string                { return platformDisplayName(id) }
func CheckAlreadyInstalled(manifest DistManifest, plat Platform) []string {
	return checkAlreadyInstalled(manifest, plat)
}
func CheckWhatIsInstalled(manifest DistManifest, platforms []Platform) []InstalledDetails {
	return checkWhatIsInstalled(manifest, platforms)
}
func UninstallInteractive(manifest DistManifest) error { return uninstallInteractive(manifest) }
func UninstallPlatform(details InstalledDetails, manifest DistManifest) {
	uninstallPlatform(details, manifest)
}
func SummarizeMissingArtifacts(missing []string) string { return summarizeMissingArtifacts(missing) }
