package main

import installpkg "github.com/zeshone/doc-agent-ai/internal/install"

type Bundle = installpkg.Bundle
type DistManifest = installpkg.DistManifest
type DistRole = installpkg.DistRole
type PromptFileMap = installpkg.PromptFileMap
type AgentFileMap = installpkg.AgentFileMap
type DistCommand = installpkg.DistCommand
type PlatformManifest = installpkg.PlatformManifest
type PlatformConfig = installpkg.PlatformConfig
type Platform = installpkg.Platform
type Reporter = installpkg.Reporter
type InstalledDetails = installpkg.InstalledDetails

var defaultReporter Reporter = installpkg.NewStdoutReporter()

func readManifestFrom(distDir string) (DistManifest, error) { return installpkg.ReadManifestFrom(distDir) }
func newOpenCodePlatform(cfg PlatformConfig) (Platform, error) { return installpkg.NewOpenCodePlatform(cfg) }
func newCopilotPlatform(cfg PlatformConfig) (Platform, error) { return installpkg.NewCopilotPlatform(cfg) }
func newPiPlatform(cfg PlatformConfig) (Platform, error)      { return installpkg.NewPiPlatform(cfg) }
func resolveHome(path string) (string, error)                 { return installpkg.ResolveHome(path) }
func registryTemplate(basePath, skillsDir, triggerStyle string) string {
	return installpkg.RegistryTemplate(basePath, skillsDir, triggerStyle)
}
func promptFileFor(platformID string, role DistRole) string { return installpkg.PromptFileFor(platformID, role) }
func agentFileFor(platformID string, role DistRole) string  { return installpkg.AgentFileFor(platformID, role) }
func detectAllPlatforms(manifest DistManifest) []Platform   { return installpkg.DetectAllPlatforms(manifest) }
func detectedSet(platforms []Platform) map[string]bool      { return installpkg.DetectedSet(platforms) }
func platformDisplayName(id string) string                  { return installpkg.PlatformDisplayName(id) }
func platformHome(id string, manifest DistManifest) string  { return installpkg.PlatformHome(id, manifest) }
func platformMissingReason(id string) string                { return installpkg.PlatformMissingReason(id) }
func checkAlreadyInstalled(manifest DistManifest, plat Platform) []string {
	return installpkg.CheckAlreadyInstalled(manifest, plat)
}
func installToPlatformWithReporter(manifest DistManifest, plat Platform, basePath, distDir string, r Reporter, globalMode ...string) error {
	bundle, err := installpkg.BundleFromDistDir(manifest, distDir)
	if err != nil {
		return err
	}
	return installpkg.InstallToPlatformWithReporter(manifest, bundle, plat, basePath, r, globalMode...)
}
func installToPlatform(manifest DistManifest, plat Platform, basePath, distDir string, globalMode ...string) error {
	bundle, err := installpkg.BundleFromDistDir(manifest, distDir)
	if err != nil {
		return err
	}
	return installpkg.InstallToPlatform(manifest, bundle, plat, basePath, globalMode...)
}
func installPlatforms(manifest DistManifest, platforms []Platform, basePath, distDir string) error {
	bundle, err := installpkg.BundleFromDistDir(manifest, distDir)
	if err != nil {
		return err
	}
	for _, plat := range platforms {
		if err := installpkg.InstallToPlatform(manifest, bundle, plat, basePath, string(ModeVault)); err != nil {
			return err
		}
	}
	return nil
}
func checkWhatIsInstalled(manifest DistManifest, platforms []Platform) []InstalledDetails {
	return installpkg.CheckWhatIsInstalled(manifest, platforms)
}
func uninstallInteractive(manifest DistManifest) error { return installpkg.UninstallInteractive(manifest) }
func ValidateBundle(bundle Bundle) []string            { return installpkg.ValidateBundleExport(bundle) }
func ExecuteInstall(bundle Bundle, plan InstallPlan, allPlatforms []Platform, r Reporter) error {
	return installpkg.ExecuteInstallExport(bundle, plan, allPlatforms, r)
}
func InstallToPlatformWithReporter(manifest DistManifest, bundle Bundle, plat Platform, basePath string, r Reporter, globalMode ...string) error {
	return installpkg.InstallToPlatformWithReporter(manifest, bundle, plat, basePath, r, globalMode...)
}
func InstallToPlatform(manifest DistManifest, bundle Bundle, plat Platform, basePath string, globalMode ...string) error {
	return installpkg.InstallToPlatform(manifest, bundle, plat, basePath, globalMode...)
}
func executeInstall(manifest DistManifest, plan InstallPlan, distDir string, allPlatforms []Platform, r Reporter) error {
	bundle, err := installpkg.BundleFromDistDir(manifest, distDir)
	if err != nil {
		return err
	}
	return installpkg.ExecuteInstallExport(bundle, plan, allPlatforms, r)
}
func runModeSwitchHookWithPlatforms(plan InstallPlan, platforms []Platform, r Reporter) {
	installpkg.RunModeSwitchHookWithPlatforms(plan, platforms, r)
}
func sweepDocReaderIfLeavingInProject(platforms []Platform, r Reporter) {
	installpkg.SweepDocReaderIfLeavingInProject(platforms, r)
}
func removePromptFilesForPlatform(plat Platform, promptIDs []string, manifest DistManifest) {
	installpkg.RemovePromptFilesForPlatform(plat, promptIDs, manifest)
}
func uninstallPlatform(details InstalledDetails, manifest DistManifest) {
	installpkg.UninstallPlatform(details, manifest)
}
func pruneEmptyDirs(startDir, stopDir string) error { return installpkg.PruneEmptyDirs(startDir, stopDir) }
func setCopilotPathOverride(path string) { installpkg.SetCopilotPathOverride(path) }
func copilotPathOverride() string        { return installpkg.CopilotPathOverride() }
func setPiPathOverride(path string)      { installpkg.SetPiPathOverride(path) }
func piPathOverride() string             { return installpkg.PiPathOverride() }
