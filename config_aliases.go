package docagent

import configpkg "github.com/zeshone/doc-agent-ai/internal/config"

type AppConfig = configpkg.AppConfig
type InstallPlan = configpkg.InstallPlan
type FlagSet = configpkg.FlagSet
type DocsMode = configpkg.DocsMode
type ProjectMarker = configpkg.ProjectMarker

const (
	ModeVault     = configpkg.ModeVault
	ModeInProject = configpkg.ModeInProject
)

func configPath() (string, error) { return configpkg.ConfigPath() }
func loadConfig() (AppConfig, bool, error) { return configpkg.Load() }
func saveConfig(cfg AppConfig) error { return configpkg.Save(cfg) }
func parsePlanFromFlags(flags FlagSet, cfg AppConfig) (InstallPlan, error) {
	return configpkg.ParsePlanFromFlags(flags, cfg)
}
func parseInstallFlags(args []string) (FlagSet, []string) { return configpkg.ParseInstallFlags(args) }
func hasInstallFlags(f FlagSet) bool                       { return configpkg.HasInstallFlags(f) }
func readMarker(dir string) (ProjectMarker, bool, error)  { return configpkg.ReadMarker(dir) }
func resolveMode(markerMode DocsMode, markerFound bool, globalMode DocsMode) DocsMode {
	return configpkg.ResolveMode(markerMode, markerFound, globalMode)
}
