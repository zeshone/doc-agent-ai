package tui

import (
	configpkg "github.com/zeshone/doc-agent-ai/internal/config"
	installpkg "github.com/zeshone/doc-agent-ai/internal/install"
)

type AppConfig = configpkg.AppConfig
type DocsMode = configpkg.DocsMode

const (
	ModeVault     = configpkg.ModeVault
	ModeInProject = configpkg.ModeInProject
)

type Bundle = installpkg.Bundle
type DistManifest = installpkg.DistManifest
type DistRole = installpkg.DistRole
type PromptFileMap = installpkg.PromptFileMap
type AgentFileMap = installpkg.AgentFileMap
type DistCommand = installpkg.DistCommand
type PlatformManifest = installpkg.PlatformManifest
type PlatformConfig = installpkg.PlatformConfig
type Platform = installpkg.Platform
type InstalledDetails = installpkg.InstalledDetails
type InstallPlan = configpkg.InstallPlan
type Reporter = installpkg.Reporter

func loadConfig() (AppConfig, bool, error)                          { return configpkg.Load() }
func platformDisplayName(id string) string                          { return installpkg.PlatformDisplayName(id) }
func checkAlreadyInstalled(manifest DistManifest, plat Platform) []string { return installpkg.CheckAlreadyInstalled(manifest, plat) }
func checkWhatIsInstalled(manifest DistManifest, platforms []Platform) []InstalledDetails {
	return installpkg.CheckWhatIsInstalled(manifest, platforms)
}
func detectAllPlatforms(manifest DistManifest) []Platform { return installpkg.DetectAllPlatforms(manifest) }
func uninstallPlatform(details InstalledDetails, manifest DistManifest) {
	installpkg.UninstallPlatform(details, manifest)
}
func ExecuteInstall(bundle Bundle, plan InstallPlan, allPlatforms []Platform, r Reporter) error {
	return installpkg.ExecuteInstall(bundle, plan, allPlatforms, r)
}
func ValidateBundle(bundle Bundle) []string { return installpkg.ValidateBundle(bundle) }
func summarizeMissingArtifacts(missing []string) string { return installpkg.SummarizeMissingArtifacts(missing) }
