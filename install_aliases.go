package docagent

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

func registryTemplate(basePath, skillsDir, triggerStyle string) string {
	return installpkg.RegistryTemplate(basePath, skillsDir, triggerStyle)
}
