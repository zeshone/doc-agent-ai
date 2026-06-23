package docagent

import (
	configpkg "github.com/zeshone/doc-agent-ai/internal/config"
	installpkg "github.com/zeshone/doc-agent-ai/internal/install"
)

type AppConfig = configpkg.AppConfig
type InstallPlan = configpkg.InstallPlan
type FlagSet = configpkg.FlagSet

const (
	ModeVault     = configpkg.ModeVault
	ModeInProject = configpkg.ModeInProject
)

type Bundle = installpkg.Bundle
type DistManifest = installpkg.DistManifest
type DistRole = installpkg.DistRole
type PlatformManifest = installpkg.PlatformManifest
type PlatformConfig = installpkg.PlatformConfig
type Platform = installpkg.Platform

func configPath() (string, error)                         { return configpkg.ConfigPath() }
func loadConfig() (AppConfig, bool, error)                { return configpkg.Load() }
func parseInstallFlags(args []string) (FlagSet, []string) { return configpkg.ParseInstallFlags(args) }
func hasInstallFlags(f FlagSet) bool                      { return configpkg.HasInstallFlags(f) }
func registryTemplate(basePath, skillsDir, triggerStyle string) string {
	return installpkg.RegistryTemplate(basePath, skillsDir, triggerStyle)
}
