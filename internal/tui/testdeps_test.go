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

func platformDisplayName(id string) string { return installpkg.PlatformDisplayName(id) }
