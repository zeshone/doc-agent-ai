package install

// testBundle returns a deterministic in-memory Bundle covering the manifest
// shape needed by all install package tests. It is hand-built — no subprocess
// or disk I/O. The fixture is intentionally minimal: it contains exactly the
// entries needed to satisfy the assertions in the test suite.
//
// Roles: doc-arch (opencode primary, all platforms), doc-prd (subagent, all platforms)
// Skills: doc-arch, doc-rec-lite, doc-reader (conditional)
// Commands: doc-arch, doc-rec
// LegacyCommandIds: the 11 ids retired in v4.0.0
// ConditionalSkills: doc-reader
//
// Placeholder tokens included in prompt/command content so substitution tests can
// assert both __DOC_AGENT_BASE_PATH__/ and __DOC_AGENT_GLOBAL_MODE__ are replaced.
func testBundle() Bundle {
	placeholder := "__DOC_AGENT_BASE_PATH__/"

	roles := []DistRole{
		{
			ID:          "doc-arch",
			Description: "Documentation orchestrator",
			Hidden:      false,
			Mode:        "primary",
			PromptFiles: PromptFileMap{
				OpenCode: "prompts/doc-arch.md",
				Copilot:  "prompts-copilot/doc-arch.md",
				Claude:   "prompts-claude/doc-arch.md",
				Qwen:     "prompts-qwen/doc-arch.md",
				Pi:       "prompts-pi/doc-arch.md",
			},
			AgentFiles: AgentFileMap{
				Copilot: "agents-copilot/doc-arch.agent.md",
				Claude:  "agents-claude/doc-arch.md",
				Qwen:    "agents-qwen/doc-arch.md",
			},
		},
		{
			ID:          "doc-prd",
			Description: "PRD generation executor",
			Hidden:      true,
			Mode:        "subagent",
			PromptFiles: PromptFileMap{
				OpenCode: "prompts/doc-prd.md",
				Copilot:  "prompts-copilot/doc-prd.md",
				Claude:   "prompts-claude/doc-prd.md",
				Qwen:     "prompts-qwen/doc-prd.md",
				Pi:       "prompts-pi/doc-prd.md",
			},
			AgentFiles: AgentFileMap{
				Copilot: "agents-copilot/doc-prd.agent.md",
				Claude:  "agents-claude/doc-prd.md",
				Qwen:    "agents-qwen/doc-prd.md",
			},
		},
	}

	commands := []DistCommand{
		{ID: "doc-arch", Description: "Full flow", Agent: "doc-arch", File: "commands/doc-arch.md"},
		{ID: "doc-rec", Description: "Requirements", Agent: "doc-arch", File: "commands/doc-rec.md"},
	}

	skills := []string{"doc-arch", "doc-rec-lite", "doc-reader"}
	conditionalSkills := []string{"doc-reader"}
	legacyIDs := []string{"arch", "idea", "rec", "prd", "refine", "tech", "pti", "mod", "feat", "ddd", "to-sdd"}

	manifest := DistManifest{
		PlaceholderBasePath: placeholder,
		Skills:              skills,
		ConditionalSkills:   conditionalSkills,
		Roles:               roles,
		Commands:            commands,
		LegacyCommandIds:    legacyIDs,
		Platforms: PlatformManifest{
			OpenCode: PlatformConfig{SkillRoot: "~/.config/opencode/skills", PromptDir: "prompts", CommandDir: "commands"},
			Copilot:  PlatformConfig{SkillRoot: "~/.copilot/skills", PromptDir: "prompts-copilot", AgentDir: "agents-copilot"},
			Claude:   PlatformConfig{SkillRoot: "~/.claude/skills", PromptDir: "prompts-claude", AgentDir: "agents-claude"},
			Qwen:     PlatformConfig{SkillRoot: "~/.qwen/skills", PromptDir: "prompts-qwen", AgentDir: "agents-qwen"},
			Pi:       PlatformConfig{SkillRoot: "~/.pi/agent/skills", PromptDir: "prompts-pi"},
		},
	}

	// Prompt content includes the BASE_PATH placeholder and the GLOBAL_MODE
	// placeholder so that substitution tests can assert both are replaced.
	promptContent := func(roleID string) []byte {
		return []byte("# " + roleID + "\npath: " + placeholder + "\nmode: __DOC_AGENT_GLOBAL_MODE__\nbase: __DOC_AGENT_GLOBAL_BASE__\n")
	}
	agentContent := func(roleID string) []byte {
		return []byte("# agent " + roleID + "\nbase: __DOC_AGENT_GLOBAL_BASE__\n")
	}
	commandContent := func(id string) []byte {
		return []byte("# /" + id + "\nbase: __DOC_AGENT_GLOBAL_BASE__\n")
	}
	skillContent := func(id string) []byte {
		return []byte("# " + id + "\nSKILL body for " + id + "\n")
	}

	files := map[string][]byte{}

	// Prompts for both roles, all platforms.
	for _, role := range roles {
		files["prompts/"+role.ID+".md"] = promptContent(role.ID)
		files["prompts-copilot/"+role.ID+".md"] = promptContent(role.ID)
		files["prompts-claude/"+role.ID+".md"] = promptContent(role.ID)
		files["prompts-qwen/"+role.ID+".md"] = promptContent(role.ID)
		files["prompts-pi/"+role.ID+".md"] = promptContent(role.ID)

		if role.AgentFiles.Copilot != "" {
			files[role.AgentFiles.Copilot] = agentContent(role.ID)
		}
		if role.AgentFiles.Claude != "" {
			files[role.AgentFiles.Claude] = agentContent(role.ID)
		}
		if role.AgentFiles.Qwen != "" {
			files[role.AgentFiles.Qwen] = agentContent(role.ID)
		}
	}

	// Commands.
	for _, cmd := range commands {
		files[cmd.File] = commandContent(cmd.ID)
	}

	// Skills (each skill needs at least one file so writeBundleDir finds the prefix).
	for _, skill := range skills {
		files["skills/"+skill+"/SKILL.md"] = skillContent(skill)
	}

	return Bundle{Manifest: manifest, Files: files}
}
