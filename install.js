#!/usr/bin/env node

/**
 * doc-agent-ai installer
 * Installs generated artifacts from dist/ into supported AI platforms.
 */

import fs from "fs";
import path from "path";
import readline from "readline";
import os from "os";
import { execSync } from "child_process";

const VERSION = "2.0.0";

const INSTALLER_DIR = path
  .dirname(new URL(import.meta.url).pathname)
  .replace(/^\/([A-Z]:)/, "$1");

const DIST_DIR = path.join(INSTALLER_DIR, "dist");
const DIST_MANIFEST = path.join(DIST_DIR, "manifest.json");

const HOME = os.homedir();
const OPENCODE_DIR = path.join(HOME, ".config", "opencode");
const OPENCODE_JSON = path.join(OPENCODE_DIR, "opencode.json");
const QWEN_DIR = path.join(HOME, ".qwen");
const COPILOT_DIR = path.join(HOME, ".copilot");
const CLAUDE_DIR = path.join(HOME, ".claude");

const TARGETS = {
  opencode: {
    home: OPENCODE_DIR,
    promptsDir: path.join(OPENCODE_DIR, "prompts", "doc"),
    commandsDir: path.join(OPENCODE_DIR, "commands")
  },
  qwen: {
    home: QWEN_DIR,
    promptsDir: path.join(QWEN_DIR, "prompts", "doc"),
    agentsDir: path.join(QWEN_DIR, "agents")
  },
  copilot: {
    home: COPILOT_DIR,
    promptsDir: path.join(COPILOT_DIR, "prompts", "doc"),
    agentsDir: path.join(COPILOT_DIR, "agents")
  },
  claude: {
    home: CLAUDE_DIR,
    promptsDir: path.join(CLAUDE_DIR, "prompts", "doc"),
    agentsDir: path.join(CLAUDE_DIR, "agents")
  }
};

const c = {
  reset: "\x1b[0m",
  bold: "\x1b[1m",
  green: "\x1b[32m",
  yellow: "\x1b[33m",
  red: "\x1b[31m",
  cyan: "\x1b[36m",
  blue: "\x1b[34m",
  gray: "\x1b[90m"
};

const ok = (msg) => console.log(`  ${c.green}✔${c.reset} ${msg}`);
const warn = (msg) => console.log(`  ${c.yellow}⚠${c.reset}  ${msg}`);
const err = (msg) => console.log(`  ${c.red}✖${c.reset} ${msg}`);
const info = (msg) => console.log(`  ${c.blue}→${c.reset} ${msg}`);
const dim = (msg) => console.log(`${c.gray}  ${msg}${c.reset}`);
const head = (msg) => console.log(`\n${c.bold}  ${msg}${c.reset}`);

function ask(rl, question) {
  return new Promise((resolve) => rl.question(question, resolve));
}

function ensureDir(dirPath) {
  fs.mkdirSync(dirPath, { recursive: true });
}

function copyFileSync(src, dest) {
  ensureDir(path.dirname(dest));
  fs.copyFileSync(src, dest);
}

function copyDirSync(src, dest) {
  ensureDir(dest);
  for (const entry of fs.readdirSync(src, { withFileTypes: true })) {
    const srcPath = path.join(src, entry.name);
    const destPath = path.join(dest, entry.name);
    if (entry.isDirectory()) copyDirSync(srcPath, destPath);
    else copyFileSync(srcPath, destPath);
  }
}

function replaceInFile(filePath, search, replace) {
  const content = fs.readFileSync(filePath, "utf8");
  if (!content.includes(search)) return;
  fs.writeFileSync(filePath, content.replaceAll(search, replace));
}

function readJson(filePath, label) {
  try {
    return JSON.parse(fs.readFileSync(filePath, "utf8"));
  } catch {
    err(`${label} is missing or invalid JSON.`);
    process.exit(1);
  }
}

function normalizeBasePath(raw) {
  const trimmed = raw.trim();
  const normalized = (trimmed || process.cwd()).replace(/[\\/]+/g, "/");
  return normalized.endsWith("/") ? normalized : `${normalized}/`;
}

function detectPlatforms() {
  let copilot = false;
  if (fs.existsSync(COPILOT_DIR)) {
    try {
      execSync("code --version", { stdio: "ignore" });
      copilot = true;
    } catch {
      copilot = false;
    }
  }

  let claude = false;
  if (fs.existsSync(CLAUDE_DIR)) {
    try {
      execSync("claude --version", { stdio: "ignore" });
      claude = true;
    } catch {
      claude = false;
    }
  }

  return {
    opencode: fs.existsSync(OPENCODE_JSON),
    qwen: fs.existsSync(QWEN_DIR),
    copilot,
    claude
  };
}

function validateDist() {
  if (!fs.existsSync(DIST_MANIFEST)) {
    err("Generated dist/ is missing.");
    err("Run `npm run generate` before installing.");
    process.exit(1);
  }

  const manifest = readJson(DIST_MANIFEST, "dist/manifest.json");
  const missing = [];

  for (const role of manifest.roles) {
    for (const relativePath of Object.values(role.promptFiles ?? {})) {
      if (!fs.existsSync(path.join(DIST_DIR, relativePath))) missing.push(relativePath);
    }
    for (const relativePath of Object.values(role.agentFiles ?? {})) {
      if (!fs.existsSync(path.join(DIST_DIR, relativePath))) missing.push(relativePath);
    }
  }

  for (const command of manifest.commands) {
    if (!fs.existsSync(path.join(DIST_DIR, command.file))) missing.push(command.file);
  }

  for (const skill of manifest.skills) {
    const skillDir = path.join(DIST_DIR, "skills", skill);
    if (!fs.existsSync(skillDir)) missing.push(path.join("skills", skill));
  }

  if (missing.length > 0) {
    err("dist/ is incomplete. Missing generated artifacts:");
    missing.forEach((item) => err(`  ${item}`));
    err("Run `npm run generate` and try again.");
    process.exit(1);
  }

  return manifest;
}

function installSkills(skills, targetHome) {
  const targetSkillsDir = path.join(targetHome, "skills");
  for (const skill of skills) {
    copyDirSync(path.join(DIST_DIR, "skills", skill), path.join(targetSkillsDir, skill));
    ok(`skill: ${skill}`);
  }
}

function installFiles(relativePaths, targetDir, basePath, placeholderBasePath, labelPrefix) {
  ensureDir(targetDir);
  for (const relativePath of relativePaths) {
    const sourcePath = path.join(DIST_DIR, relativePath);
    const targetPath = path.join(targetDir, path.basename(relativePath));
    copyFileSync(sourcePath, targetPath);
    replaceInFile(targetPath, placeholderBasePath, basePath);
    ok(`${labelPrefix}: ${path.basename(targetPath)}`);
  }
}

function patchOpencodeJson(manifest) {
  const config = readJson(OPENCODE_JSON, "opencode.json");
  if (!config.agent) config.agent = {};

  let promptsBase = path.join(OPENCODE_DIR, "prompts", "doc");
  if (process.platform === "win32") promptsBase = promptsBase.replace(/\//g, "\\");

  for (const role of manifest.roles) {
    config.agent[role.id] = {
      description: role.description,
      mode: role.mode,
      prompt: `{file:${path.join(promptsBase, `${role.id}.md`)}}`,
      tools: role.opencodeTools,
      ...(role.hidden ? { hidden: true } : {})
    };
    ok(`agent registered: ${role.id}`);
  }

  fs.writeFileSync(OPENCODE_JSON, JSON.stringify(config, null, 2));
}

function buildSkillRegistry(basePath, skillsBase, style) {
  const docArchTrigger = style === "claude"
    ? "Documenting a project, /arch, /rec, /prd, /tech, /pti, /mod"
    : "`/arch`, `/rec`, `/prd`, `/tech`, `/pti`, `/mod` — project documentation workflow";

  return `# Skill Registry — doc-agent-ai

**Auto-generated by doc-agent-ai installer v${VERSION}**
**Base path:** ${basePath}

## User Skills

| Trigger | Skill | Path |
|---------|-------|------|
| ${docArchTrigger} | doc-arch | ${path.join(skillsBase, "doc-arch", "SKILL.md")} |
| Generating PRD for a project | doc-prd | ${path.join(skillsBase, "doc-prd", "SKILL.md")} |
| Gathering requirements, elicitation session, stakeholder interview | doc-rec | ${path.join(skillsBase, "doc-rec", "SKILL.md")} |
| Creating technical specification, new feature needing documented architecture | doc-tech | ${path.join(skillsBase, "doc-tech", "SKILL.md")} |
| Converting PRD into local issues, breaking down PRD into work items | doc-pti | ${path.join(skillsBase, "doc-pti", "SKILL.md")} |

## Compact Rules

### doc-arch
- Base path for all projects: \`${basePath}\`
- Commands: \`arch <s>\` (full flow), \`rec\`, \`prd\`, \`tech\`, \`pti\` (individual), \`mod <s> <m>\` (module)
- Module paths: \`<sistema>/<modulo>\` and \`<sistema>/<modulo>/<submodulo>\` — max 2 levels deep
- Archetype detected in \`rec\`: acotado (single delivery) or evolutivo (long lifecycle with modules)
- Index file \`<sistema>.md\` uses Obsidian \`[[...]]\` links and checkboxes — update after every phase
- Tech spec for modules: ALWAYS ask if inherits parent architecture (delta) or diverges (full spec)
- Modules read parent context: rec reads parent requirements, prd reads parent prd
- Issues generated as local .md only — never push to GitHub unless user explicitly asks
- Never invent context — if prerequisite missing, stop and show exact command to run

### doc-rec
- Start in executive/business language and increase technical depth progressively
- Do not open with BABOK jargon or implementation detail unless stakeholder already speaks that way

### doc-prd
- PRD is more technical than rec, but clear and pedagogical
- Ask before assuming; unknowns become \`TBD\`

### doc-tech
- Highest technical precision, still legible
- Ask before assuming; unknowns become \`TBD\` / \`Open Decision\`

### doc-pti
- Local-first issues, end-to-end slices, visible blockers and TBDs
`;
}

function writeSkillRegistry(targetHome, basePath, style) {
  const registryPath = path.join(targetHome, ".atl", "skill-registry.md");
  ensureDir(path.dirname(registryPath));
  fs.writeFileSync(registryPath, buildSkillRegistry(basePath, path.join(targetHome, "skills"), style));
  ok("skill-registry.md written");
}

function checkOpencodeAlreadyInstalled(roleIds) {
  try {
    const config = JSON.parse(fs.readFileSync(OPENCODE_JSON, "utf8"));
    return roleIds.filter((id) => config.agent?.[id]);
  } catch {
    warn("opencode.json is not valid JSON — cannot detect existing agents.");
    return [];
  }
}

function checkAgentsAlreadyInstalled(platformId, manifest) {
  const agentsDir = TARGETS[platformId].agentsDir;
  if (!agentsDir || !fs.existsSync(agentsDir)) return [];

  return manifest.roles
    .filter((role) => {
      const relativePath = role.agentFiles?.[platformId];
      return relativePath && fs.existsSync(path.join(agentsDir, path.basename(relativePath)));
    })
    .map((role) => role.id);
}

async function main() {
  console.log();
  console.log(`${c.bold}${c.cyan}  doc-agent-ai${c.reset} ${c.gray}v${VERSION}${c.reset}`);
  console.log(`${c.gray}  Documentation workflow agent installer${c.reset}`);
  console.log();

  const manifest = validateDist();
  const roleIds = manifest.roles.map((role) => role.id);

  head("Detecting platforms...");
  const platforms = detectPlatforms();

  if (platforms.opencode) ok(`opencode detected  ${c.gray}(${OPENCODE_DIR})${c.reset}`);
  else warn(`opencode not found  ${c.gray}(${OPENCODE_JSON} missing)${c.reset}`);

  if (platforms.qwen) ok(`Qwen Code detected  ${c.gray}(${QWEN_DIR})${c.reset}`);
  else warn(`Qwen Code not found  ${c.gray}(${QWEN_DIR} missing)${c.reset}`);

  if (platforms.copilot) ok(`GitHub Copilot detected  ${c.gray}(${COPILOT_DIR} + code CLI)${c.reset}`);
  else warn(`GitHub Copilot not found  ${c.gray}(${COPILOT_DIR} missing or 'code' not in PATH)${c.reset}`);

  if (platforms.claude) ok(`Claude Code detected  ${c.gray}(${CLAUDE_DIR})${c.reset}`);
  else warn(`Claude Code not found  ${c.gray}(${CLAUDE_DIR} missing)${c.reset}`);

  if (!platforms.opencode && !platforms.qwen && !platforms.copilot && !platforms.claude) {
    console.log();
    err("No supported platform detected.");
    err("Install opencode, Qwen Code, GitHub Copilot, or Claude Code before running this installer.");
    process.exit(1);
  }

  const rl = readline.createInterface({ input: process.stdin, output: process.stdout });

  try {
    let installOpencode = platforms.opencode;
    let installQwen = platforms.qwen;
    let installCopilot = platforms.copilot;
    let installClaude = platforms.claude;

    const detected = [platforms.opencode, platforms.qwen, platforms.copilot, platforms.claude].filter(Boolean).length;
    if (detected > 1) {
      head("Platform selection");
      console.log(`${c.gray}  Multiple platforms detected. Choose which to install:${c.reset}`);
      let idx = 1;
      const labelMap = {};
      if (platforms.opencode) { console.log(`${c.gray}  [${idx}] opencode only${c.reset}`); labelMap[idx++] = "opencode"; }
      if (platforms.qwen) { console.log(`${c.gray}  [${idx}] Qwen Code only${c.reset}`); labelMap[idx++] = "qwen"; }
      if (platforms.copilot) { console.log(`${c.gray}  [${idx}] GitHub Copilot only${c.reset}`); labelMap[idx++] = "copilot"; }
      if (platforms.claude) { console.log(`${c.gray}  [${idx}] Claude Code only${c.reset}`); labelMap[idx++] = "claude"; }
      console.log(`${c.gray}  [${idx}] All (default)${c.reset}`);
      console.log();
      const choice = await ask(rl, `  Selection ${c.gray}(Enter = all)${c.reset}: `);
      const sel = parseInt(choice.trim(), 10);
      if (labelMap[sel]) {
        installOpencode = labelMap[sel] === "opencode";
        installQwen = labelMap[sel] === "qwen";
        installCopilot = labelMap[sel] === "copilot";
        installClaude = labelMap[sel] === "claude";
      }
    }

    if (installOpencode) {
      const existing = checkOpencodeAlreadyInstalled(roleIds);
      if (existing.length > 0) {
        console.log();
        warn("The following agents are already registered in opencode.json:");
        existing.forEach((id) => dim(`  - ${id}`));
        const answer = await ask(rl, `\n  ${c.yellow}Overwrite opencode installation?${c.reset} (y/N) `);
        if (answer.trim().toLowerCase() !== "y") {
          info("Skipping opencode.");
          installOpencode = false;
        }
      }
    }

    if (installQwen) {
      const existing = checkAgentsAlreadyInstalled("qwen", manifest);
      if (existing.length > 0) {
        console.log();
        warn("The following agents are already installed in Qwen Code:");
        existing.forEach((id) => dim(`  - ${id}`));
        const answer = await ask(rl, `\n  ${c.yellow}Overwrite Qwen Code installation?${c.reset} (y/N) `);
        if (answer.trim().toLowerCase() !== "y") {
          info("Skipping Qwen Code.");
          installQwen = false;
        }
      }
    }

    if (installCopilot) {
      const existing = checkAgentsAlreadyInstalled("copilot", manifest);
      if (existing.length > 0) {
        console.log();
        warn("The following agents are already installed in GitHub Copilot:");
        existing.forEach((id) => dim(`  - ${id}`));
        const answer = await ask(rl, `\n  ${c.yellow}Overwrite GitHub Copilot installation?${c.reset} (y/N) `);
        if (answer.trim().toLowerCase() !== "y") {
          info("Skipping GitHub Copilot.");
          installCopilot = false;
        }
      }
    }

    if (installClaude) {
      const existing = checkAgentsAlreadyInstalled("claude", manifest);
      if (existing.length > 0) {
        console.log();
        warn("The following agents are already installed in Claude Code:");
        existing.forEach((id) => dim(`  - ${id}`));
        const answer = await ask(rl, `\n  ${c.yellow}Overwrite Claude Code installation?${c.reset} (y/N) `);
        if (answer.trim().toLowerCase() !== "y") {
          info("Skipping Claude Code.");
          installClaude = false;
        }
      }
    }

    if (!installOpencode && !installQwen && !installCopilot && !installClaude) {
      info("Nothing to install. Exiting.");
      rl.close();
      process.exit(0);
    }

    head("Configuration");
    console.log(`${c.gray}  Where should the agent save your project documentation?${c.reset}`);
    console.log(`${c.gray}  This is the root folder where all systems, PRDs and specs will be created.${c.reset}`);
    console.log(`${c.yellow}  ⚠  If you skip this, files will be saved in the current directory: ${process.cwd()}${c.reset}`);
    console.log();

    const rawBase = await ask(rl, `  Documentation path ${c.gray}(press Enter to use current dir)${c.reset}: `);
    const basePath = normalizeBasePath(rawBase);

    if (!fs.existsSync(basePath)) {
      warn(`Path does not exist yet: ${basePath}`);
      warn("The agent will still work — create the folder before first use.");
    }

    console.log();
    head("Ready to install");
    if (installOpencode) info(`opencode config:        ${OPENCODE_DIR}`);
    if (installQwen) info(`Qwen Code config:       ${QWEN_DIR}`);
    if (installCopilot) info(`GitHub Copilot config:  ${COPILOT_DIR}`);
    if (installClaude) info(`Claude Code config:     ${CLAUDE_DIR}`);
    info(`projects base:          ${basePath}`);
    info(`artifact source:        ${DIST_DIR}`);
    console.log();

    const confirm = await ask(rl, `  ${c.bold}Proceed?${c.reset} (Y/n) `);
    if (confirm.trim().toLowerCase() === "n") {
      info("Installation cancelled.");
      rl.close();
      process.exit(0);
    }

    head("Installing skills...");
    if (installOpencode) installSkills(manifest.skills, TARGETS.opencode.home);
    if (installQwen) installSkills(manifest.skills, TARGETS.qwen.home);
    if (installCopilot) installSkills(manifest.skills, TARGETS.copilot.home);
    if (installClaude) installSkills(manifest.skills, TARGETS.claude.home);

    if (installOpencode) {
      head("Installing for opencode...");
      installFiles(manifest.roles.map((role) => role.promptFiles.opencode).filter(Boolean), TARGETS.opencode.promptsDir, basePath, manifest.placeholderBasePath, "prompt");
      installFiles(manifest.commands.map((command) => command.file).filter(Boolean), TARGETS.opencode.commandsDir, basePath, manifest.placeholderBasePath, "command");
      patchOpencodeJson(manifest);
      writeSkillRegistry(TARGETS.opencode.home, basePath, "opencode");
    }

    if (installQwen) {
      head("Installing for Qwen Code...");
      installFiles(manifest.roles.map((role) => role.promptFiles.qwen).filter(Boolean), TARGETS.qwen.promptsDir, basePath, manifest.placeholderBasePath, "prompt");
      installFiles(manifest.roles.map((role) => role.agentFiles.qwen).filter(Boolean), TARGETS.qwen.agentsDir, basePath, manifest.placeholderBasePath, "agent");
    }

    if (installCopilot) {
      head("Installing for GitHub Copilot...");
      installFiles(manifest.roles.map((role) => role.promptFiles.copilot).filter(Boolean), TARGETS.copilot.promptsDir, basePath, manifest.placeholderBasePath, "prompt");
      installFiles(manifest.roles.map((role) => role.agentFiles.copilot).filter(Boolean), TARGETS.copilot.agentsDir, basePath, manifest.placeholderBasePath, "agent");
    }

    if (installClaude) {
      head("Installing for Claude Code...");
      installFiles(manifest.roles.map((role) => role.promptFiles.claude).filter(Boolean), TARGETS.claude.promptsDir, basePath, manifest.placeholderBasePath, "prompt");
      installFiles(manifest.roles.map((role) => role.agentFiles.claude).filter(Boolean), TARGETS.claude.agentsDir, basePath, manifest.placeholderBasePath, "agent");
      writeSkillRegistry(TARGETS.claude.home, basePath, "claude");
    }

    console.log();
    console.log(`${c.bold}${c.green}  ✔ Installation complete!${c.reset}`);
    console.log();
  } finally {
    rl.close();
  }
}

main().catch((e) => {
  err(`Unexpected error: ${e.message}`);
  process.exit(1);
});
