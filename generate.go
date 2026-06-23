package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

//go:embed src skills
var embedded embed.FS

// BuildBundle renders all install content into an in-memory bundle.
func BuildBundle() (Bundle, error) {
	// Fail fast on invalid skill frontmatter before touching the output dir.
	if err := lintEmbeddedSkills(); err != nil {
		return Bundle{}, fmt.Errorf("skill frontmatter lint failed: %w", err)
	}

	files := make(map[string][]byte)

	// Read manifests from embedded FS
	contentRaw, err := embedded.ReadFile("src/manifests/content.json")
	if err != nil {
		return Bundle{}, fmt.Errorf("read content.json: %w", err)
	}
	var contentManifest ContentManifest
	if err := json.Unmarshal(contentRaw, &contentManifest); err != nil {
		return Bundle{}, fmt.Errorf("parse content.json: %w", err)
	}

	platformsRaw, err := embedded.ReadFile("src/manifests/platforms.json")
	if err != nil {
		return Bundle{}, fmt.Errorf("read platforms.json: %w", err)
	}
	var platformManifest PlatformManifest
	if err := json.Unmarshal(platformsRaw, &platformManifest); err != nil {
		return Bundle{}, fmt.Errorf("parse platforms.json: %w", err)
	}

	// Read templates
	promptTmpl, err := embedded.ReadFile("src/templates/prompt.md.tmpl")
	if err != nil {
		return Bundle{}, fmt.Errorf("read prompt.md.tmpl: %w", err)
	}
	commandTmpl, err := embedded.ReadFile("src/templates/command.md.tmpl")
	if err != nil {
		return Bundle{}, fmt.Errorf("read command.md.tmpl: %w", err)
	}

	// Read the path-resolution preamble template. Its content uses __DOC_AGENT_*__
	// tokens (install-time substitution) rather than {{KEY}} template vars, so we
	// embed it verbatim as a string value in buildBodyVars. Content files that
	// reference {{PATH_RESOLUTION}} get the expanded preamble prose; content files
	// that do NOT reference it are unaffected (key present but unused is fine under
	// missingkey=error — only a MISSING key causes an error).
	pathResolutionTmplRaw, err := embedded.ReadFile("src/templates/path-resolution.md.tmpl")
	if err != nil {
		return Bundle{}, fmt.Errorf("read path-resolution.md.tmpl: %w", err)
	}
	// Trim the trailing newline added by the file so the injected block is clean.
	pathResolutionPreamble := strings.TrimRight(string(pathResolutionTmplRaw), "\n")

	// Ordered platform map (matches platforms.json key order for portability)
	type namedPlatform struct {
		id  string
		cfg PlatformConfig
	}
	platforms := []namedPlatform{
		{"opencode", platformManifest.OpenCode},
		{"copilot", platformManifest.Copilot},
		{"claude", platformManifest.Claude},
		{"qwen", platformManifest.Qwen},
		{"pi", platformManifest.Pi},
	}

	// Per-role variables that change across platforms.
	// PATH_RESOLUTION is always included in the map so that content files
	// which reference {{PATH_RESOLUTION}} expand to the canonical resolution
	// preamble. Content files that do NOT reference the token are unaffected
	// (missingkey=error only fires on keys that ARE referenced but absent;
	// a key that is present but not referenced in the template is silently ignored).
	buildBodyVars := func(platformID string, platform PlatformConfig, role RoleConfig) map[string]string {
		return map[string]string{
			"BASE_PATH":          contentManifest.PlaceholderBasePath,
			"SKILL_PATH":         platform.SkillRoot + "/" + role.Skill + "/SKILL.md",
			"RULES_SKILL_PATH":   platform.SkillRoot + "/" + role.RulesSkill + "/SKILL.md",
			"TECH_TEMPLATE_PATH": platform.SkillRoot + "/doc-tech/references/template.md",
			"PATH_RESOLUTION":    pathResolutionPreamble,
		}
	}

	// --- Roles × Platforms ---
	for _, role := range contentManifest.Roles {
		roleBodyRaw, err := embedded.ReadFile(path.Join("src/content", role.Content))
		if err != nil {
			return Bundle{}, fmt.Errorf("read role content %s: %w", role.Content, err)
		}

		for _, plat := range platforms {
			// Render prompt body with platform-specific variables
			promptBody, err := renderTemplate(string(roleBodyRaw), buildBodyVars(plat.id, plat.cfg, role))
			if err != nil {
				return Bundle{}, fmt.Errorf("render body for %s/%s: %w", plat.id, role.ID, err)
			}

			// Write prompt file
			promptOut, err := renderTemplate(string(promptTmpl), map[string]string{"BODY": promptBody})
			if err != nil {
				return Bundle{}, fmt.Errorf("render prompt for %s/%s: %w", plat.id, role.ID, err)
			}
			promptPath := filepath.ToSlash(filepath.Join(plat.cfg.PromptDir, role.ID+".md"))
			files[promptPath] = ensureTrailingNewline([]byte(promptOut))

			// Agent files (platforms with agent support)
			if plat.cfg.AgentDir == "" {
				continue
			}

			agentTmpl, err := embedded.ReadFile(path.Join("src/templates", plat.cfg.AgentTemplate))
			if err != nil {
				return Bundle{}, fmt.Errorf("read agent template %s: %w", plat.cfg.AgentTemplate, err)
			}

			// Select tools: orchestrator tools for doc-arch, agent tools otherwise
			tools := plat.cfg.AgentTools
			if role.ID == "doc-arch" && len(plat.cfg.OrchestratorTools) > 0 {
				tools = plat.cfg.OrchestratorTools
			}

			// Copilot agents block (children for doc-arch orchestrator)
			var agentsBlock string
			if plat.id == "copilot" && len(role.CopilotChildren) > 0 {
				agentsBlock = "agents:\n" + yamlList(role.CopilotChildren) + "\n"
			}

			// Qwen user-invocable line (always present for qwen agents)
			var userInvocableLine string
			if plat.id == "qwen" {
				userInvocableLine = "user-invocable: " + boolText(role.UserInvocable) + "\n"
			}

			// Approval mode: orchestrator gets orchestratorApprovalMode, others get approvalMode
			approvalMode := plat.cfg.ApprovalMode
			if role.ID == "doc-arch" && plat.cfg.OrchestratorApprovalMode != "" {
				approvalMode = plat.cfg.OrchestratorApprovalMode
			}
			if approvalMode == "" {
				approvalMode = "auto-edit"
			}

			agentVars := map[string]string{
				"NAME":                role.ID,
				"DESCRIPTION":         role.Description,
				"TOOLS_YAML":          yamlList(tools),
				"USER_INVOCABLE":      boolText(role.UserInvocable),
				"USER_INVOCABLE_LINE": userInvocableLine,
				"AGENTS_BLOCK":        agentsBlock,
				"APPROVAL_MODE":       approvalMode,
				"BODY":                promptBody,
			}

			agentOut, err := renderTemplate(string(agentTmpl), agentVars)
			if err != nil {
				return Bundle{}, fmt.Errorf("render agent for %s/%s: %w", plat.id, role.ID, err)
			}
			agentPath := filepath.ToSlash(filepath.Join(plat.cfg.AgentDir, role.ID+plat.cfg.AgentExtension))
			files[agentPath] = ensureTrailingNewline([]byte(agentOut))
		}
	}

	// --- Commands (opencode only) ---
	for _, cmd := range contentManifest.Commands {
		cmdBodyRaw, err := embedded.ReadFile(path.Join("src/content", cmd.Content))
		if err != nil {
			return Bundle{}, fmt.Errorf("read command content %s: %w", cmd.Content, err)
		}

		cmdBody, err := renderTemplate(string(cmdBodyRaw), map[string]string{
			"BASE_PATH":       contentManifest.PlaceholderBasePath,
			"PATH_RESOLUTION": pathResolutionPreamble,
		})
		if err != nil {
			return Bundle{}, fmt.Errorf("render command body for %s: %w", cmd.ID, err)
		}

		cmdOut, err := renderTemplate(string(commandTmpl), map[string]string{
			"DESCRIPTION": cmd.Description,
			"AGENT":       cmd.Agent,
			"BODY":        cmdBody,
		})
		if err != nil {
			return Bundle{}, fmt.Errorf("render command %s: %w", cmd.ID, err)
		}

		cmdPath := filepath.ToSlash(filepath.Join(platformManifest.OpenCode.CommandDir, cmd.ID+".md"))
		files[cmdPath] = ensureTrailingNewline([]byte(cmdOut))
	}

	manifest := buildDistManifest(contentManifest, platformManifest)

	if err := collectSkills(files); err != nil {
		return Bundle{}, fmt.Errorf("copy skills: %w", err)
	}

	return Bundle{Manifest: manifest, Files: files}, nil
}

// Generate produces the dist/ directory from embedded src/ + skills/ content.
// outputDir is explicit; runtime commands must not call this implicitly.
func Generate(outputDir string) error {
	bundle, err := BuildBundle()
	if err != nil {
		return err
	}

	if err := os.RemoveAll(outputDir); err != nil {
		return fmt.Errorf("clean %s: %w", outputDir, err)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create %s: %w", outputDir, err)
	}

	for rel, data := range bundle.Files {
		absPath := filepath.Join(outputDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(absPath, data, 0644); err != nil {
			return err
		}
	}

	if err := writeManifest(outputDir, bundle.Manifest); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	return nil
}

func generate(outputDir string) error { return Generate(outputDir) }

// writeFile writes content to path, creating parent directories as needed.
// A trailing newline is added if missing (matching generate.js behavior).
func writeFile(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func ensureTrailingNewline(data []byte) []byte {
	if len(data) == 0 || data[len(data)-1] != '\n' {
		return append(data, '\n')
	}
	return data
}

// lintEmbeddedSkills walks the embedded skills/ tree and validates every
// SKILL.md frontmatter, returning the first violation found.
func lintEmbeddedSkills() error {
	return fs.WalkDir(embedded, "skills", func(embPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(embPath, "SKILL.md") {
			return nil
		}
		data, err := embedded.ReadFile(embPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", embPath, err)
		}
		return lintSkillFrontmatter(embPath, data)
	})
}

// lintSkillFrontmatter validates a single SKILL.md frontmatter block. It rejects
// unquoted scalar values that contain ": " (colon followed by space), because
// strict YAML parsers interpret those as nested mappings inside a compact value
// and refuse to load the file. Returns nil if the frontmatter is absent.
func lintSkillFrontmatter(filename string, data []byte) error {
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil
	}

	for i := 1; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			return nil
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			continue
		}
		value := strings.TrimSpace(line[colonIdx+1:])
		if value == "" {
			continue
		}
		if strings.HasPrefix(value, "'") || strings.HasPrefix(value, "\"") {
			continue
		}

		if strings.Contains(value, ": ") {
			return fmt.Errorf(
				"%s:%d: unquoted frontmatter value contains ': ' (colon+space) — strict YAML parsers will reject this as a nested mapping. Wrap the value in single quotes. Line: %q",
				filename, i+1, line,
			)
		}
	}

	return nil
}

// collectSkills recursively copies embedded skills/* into the bundle file map.
func collectSkills(files map[string][]byte) error {
	return fs.WalkDir(embedded, "skills", func(embPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		data, err := embedded.ReadFile(embPath)
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", embPath, err)
		}

		files[filepath.ToSlash(embPath)] = data
		return nil
	})
}

func buildDistManifest(content ContentManifest, platforms PlatformManifest) DistManifest {
	platSlice := []struct {
		id  string
		cfg PlatformConfig
	}{
		{"opencode", platforms.OpenCode},
		{"copilot", platforms.Copilot},
		{"claude", platforms.Claude},
		{"qwen", platforms.Qwen},
		{"pi", platforms.Pi},
	}

	distRoles := make([]DistRole, 0, len(content.Roles))
	for _, role := range content.Roles {
		dr := DistRole{
			ID:            role.ID,
			Description:   role.Description,
			Hidden:        role.Hidden,
			Mode:          role.Mode,
			OpenCodeTools: role.OpenCodeTools,
		}

		// Prompt files: all 5 platforms
		for _, p := range platSlice {
			path := filepath.ToSlash(filepath.Join(p.cfg.PromptDir, role.ID+".md"))
			switch p.id {
			case "opencode":
				dr.PromptFiles.OpenCode = path
			case "copilot":
				dr.PromptFiles.Copilot = path
			case "claude":
				dr.PromptFiles.Claude = path
			case "qwen":
				dr.PromptFiles.Qwen = path
			case "pi":
				dr.PromptFiles.Pi = path
			}
		}

		// Agent files: only platforms with agentDir (copilot, claude, qwen)
		for _, p := range platSlice {
			if p.cfg.AgentDir == "" {
				continue
			}
			path := filepath.ToSlash(filepath.Join(p.cfg.AgentDir, role.ID+p.cfg.AgentExtension))
			switch p.id {
			case "copilot":
				dr.AgentFiles.Copilot = path
			case "claude":
				dr.AgentFiles.Claude = path
			case "qwen":
				dr.AgentFiles.Qwen = path
			}
		}

		distRoles = append(distRoles, dr)
	}

	distCommands := make([]DistCommand, 0, len(content.Commands))
	for _, cmd := range content.Commands {
		distCommands = append(distCommands, DistCommand{
			ID:          cmd.ID,
			Description: cmd.Description,
			Agent:       cmd.Agent,
			File:        filepath.ToSlash(filepath.Join(platforms.OpenCode.CommandDir, cmd.ID+".md")),
		})
	}

	return DistManifest{
		GeneratedAt:         time.Now().UTC().Format(time.RFC3339Nano),
		PlaceholderBasePath: content.PlaceholderBasePath,
		Skills:              content.Skills,
		ConditionalSkills:   content.ConditionalSkills,
		Roles:               distRoles,
		Commands:            distCommands,
		LegacyCommandIds:    content.LegacyCommandIds,
		Platforms:           platforms,
	}
}
// writeManifest marshals and writes dist/manifest.json.
func writeManifest(outputDir string, manifest DistManifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	data = append(data, '\n')

	manifestPath := filepath.Join(outputDir, "manifest.json")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(manifestPath, data, 0644)
}
