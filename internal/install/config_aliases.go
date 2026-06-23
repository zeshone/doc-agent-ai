package install

import configpkg "github.com/zeshone/doc-agent-ai/internal/config"

type AppConfig = configpkg.AppConfig
type InstallPlan = configpkg.InstallPlan
type FlagSet = configpkg.FlagSet
type DocsMode = configpkg.DocsMode

const (
	ModeVault     = configpkg.ModeVault
	ModeInProject = configpkg.ModeInProject
)

func loadConfig() (AppConfig, bool, error) { return configpkg.Load() }
func saveConfig(cfg AppConfig) error       { return configpkg.Save(cfg) }
