#!/usr/bin/env node

/**
 * doc-agent-ai uninstaller
 * Removes generated artifacts from supported AI platforms.
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
const skip = (msg) => console.log(`  ${c.gray}–${c.reset} ${msg} ${c.gray}(not found, skipped)${c.reset}`);
const subInfo = (msg) => console.log(`    ${c.blue}→${c.reset} ${msg}`);

function ask(rl, question) {
  return new Promise((resolve) => rl.question(question, resolve));
}

function readJson(filePath, label) {
  try {
    return JSON.parse(fs.readFileSync(filePath, "utf8"));
  } catch {
    err(`${label} is missing or invalid JSON.`);
    process.exit(1);
  }
}

function removeDirIfExists(dirPath, label) {
  if (fs.existsSync(dirPath)) {
    fs.rmSync(dirPath, { recursive: true, force: true });
    ok(`removed: ${label}`);
  } else {
    skip(label);
  }
}

function removeFileIfExists(filePath, label) {
  if (fs.existsSync(filePath)) {
    fs.unlinkSync(filePath);
    ok(`removed: ${label}`);
  } else {
    skip(label);
  }
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
    err("Run `npm run generate` before uninstalling.");
    process.exit(1);
  }

  return readJson(DIST_MANIFEST, "dist/manifest.json");
}

function cleanupEmptyDirs(startDir, stopDir) {
  let current = startDir;
  while (
    current
    && current !== stopDir
    && current.startsWith(stopDir)
    && fs.existsSync(current)
    && fs.statSync(current).isDirectory()
    && fs.readdirSync(current).length === 0
  ) {
    fs.rmdirSync(current);
    current = path.dirname(current);
  }
}

function getInstalledPromptIds(platformId, manifest) {
  return manifest.roles
    .filter((role) => {
      const relativePath = role.promptFiles?.[platformId];
      return relativePath && fs.existsSync(path.join(TARGETS[platformId].promptsDir, path.basename(relativePath)));
    })
    .map((role) => role.id);
}

function getInstalledAgentIds(platformId, manifest) {
  if (platformId === "opencode") {
    if (!fs.existsSync(OPENCODE_JSON)) return [];

    try {
      const config = JSON.parse(fs.readFileSync(OPENCODE_JSON, "utf8"));
      return manifest.roles
        .filter((role) => config.agent?.[role.id])
        .map((role) => role.id);
    } catch {
      warn("opencode.json is not valid JSON — agent detection will be skipped.");
      return [];
    }
  }

  const agentsDir = TARGETS[platformId].agentsDir;
  if (!agentsDir || !fs.existsSync(agentsDir)) return [];

  return manifest.roles
    .filter((role) => {
      const relativePath = role.agentFiles?.[platformId];
      return relativePath && fs.existsSync(path.join(agentsDir, path.basename(relativePath)));
    })
    .map((role) => role.id);
}

function getInstalledSkillIds(platformId, manifest) {
  return manifest.skills.filter((skill) => fs.existsSync(path.join(TARGETS[platformId].home, "skills", skill)));
}

function getInstalledCommandIds(manifest) {
  return manifest.commands
    .filter((command) => fs.existsSync(path.join(TARGETS.opencode.commandsDir, path.basename(command.file))))
    .map((command) => command.id);
}

function getRegistryState(platformId) {
  if (platformId !== "opencode" && platformId !== "claude") return false;
  return fs.existsSync(path.join(TARGETS[platformId].home, ".atl", "skill-registry.md"));
}

function checkWhatIsInstalled(manifest, platforms) {
  const installed = {};

  for (const platformId of Object.keys(TARGETS)) {
    if (!platforms[platformId]) continue;

    const skills = getInstalledSkillIds(platformId, manifest);
    const prompts = getInstalledPromptIds(platformId, manifest);
    const agents = getInstalledAgentIds(platformId, manifest);
    const commands = platformId === "opencode" ? getInstalledCommandIds(manifest) : [];
    const registry = getRegistryState(platformId);

    installed[platformId] = {
      skills,
      prompts,
      agents,
      commands,
      registry,
      any: skills.length > 0 || prompts.length > 0 || agents.length > 0 || commands.length > 0 || registry
    };
  }

  return installed;
}

function formatPromptDir(platformId) {
  return `${path.relative(TARGETS[platformId].home, TARGETS[platformId].promptsDir).replace(/\\/g, "/")}/`;
}

function removeSkillsForPlatform(platformId, skillIds) {
  const skillsDir = path.join(TARGETS[platformId].home, "skills");
  for (const skillId of skillIds) {
    removeDirIfExists(path.join(skillsDir, skillId), `skill: ${skillId}`);
  }
  cleanupEmptyDirs(skillsDir, TARGETS[platformId].home);
}

function removePromptFilesForPlatform(platformId, promptIds, manifest) {
  const target = TARGETS[platformId];
  const promptFiles = manifest.roles
    .filter((role) => promptIds.includes(role.id))
    .map((role) => role.promptFiles?.[platformId])
    .filter(Boolean);

  for (const relativePath of promptFiles) {
    removeFileIfExists(path.join(target.promptsDir, path.basename(relativePath)), `prompt: ${path.basename(relativePath)}`);
  }

  cleanupEmptyDirs(target.promptsDir, target.home);
}

function removeCommandFiles(commandIds, manifest) {
  const commandFiles = manifest.commands
    .filter((command) => commandIds.includes(command.id))
    .map((command) => command.file);

  for (const relativePath of commandFiles) {
    const commandId = path.basename(relativePath, path.extname(relativePath));
    removeFileIfExists(path.join(TARGETS.opencode.commandsDir, path.basename(relativePath)), `command: /${commandId}`);
  }

  cleanupEmptyDirs(TARGETS.opencode.commandsDir, TARGETS.opencode.home);
}

function removeAgentsFromOpencode(agentIds) {
  if (!fs.existsSync(OPENCODE_JSON)) {
    warn("opencode.json not found — skipping agent cleanup.");
    return;
  }

  let config;
  try {
    config = JSON.parse(fs.readFileSync(OPENCODE_JSON, "utf8"));
  } catch {
    err("opencode.json is not valid JSON — skipping agent cleanup.");
    return;
  }

  if (!config.agent) {
    warn("opencode.json has no agent section — skipping agent cleanup.");
    return;
  }

  let removed = 0;
  for (const agentId of agentIds) {
    if (config.agent[agentId]) {
      delete config.agent[agentId];
      ok(`agent removed: ${agentId}`);
      removed++;
    } else {
      skip(`agent: ${agentId}`);
    }
  }

  if (removed > 0) {
    fs.writeFileSync(OPENCODE_JSON, JSON.stringify(config, null, 2));
  }
}

function removeAgentFilesForPlatform(platformId, agentIds, manifest) {
  const target = TARGETS[platformId];
  const agentFiles = manifest.roles
    .filter((role) => agentIds.includes(role.id))
    .map((role) => role.agentFiles?.[platformId])
    .filter(Boolean);

  for (const relativePath of agentFiles) {
    removeFileIfExists(path.join(target.agentsDir, path.basename(relativePath)), `agent: ${path.basename(relativePath)}`);
  }

  cleanupEmptyDirs(target.agentsDir, target.home);
}

function removeSkillRegistry(platformId) {
  const registryPath = path.join(TARGETS[platformId].home, ".atl", "skill-registry.md");
  removeFileIfExists(registryPath, ".atl/skill-registry.md");
  cleanupEmptyDirs(path.dirname(registryPath), TARGETS[platformId].home);
}

function uninstallPlatform(platformId, details, manifest) {
  head(`Removing from ${platformId}...`);

  if (details.skills.length > 0) removeSkillsForPlatform(platformId, details.skills);
  if (details.prompts.length > 0) removePromptFilesForPlatform(platformId, details.prompts, manifest);

  if (platformId === "opencode") {
    if (details.commands.length > 0) removeCommandFiles(details.commands, manifest);
    if (details.agents.length > 0) removeAgentsFromOpencode(details.agents);
    if (details.registry) removeSkillRegistry(platformId);
    return;
  }

  if (details.agents.length > 0) removeAgentFilesForPlatform(platformId, details.agents, manifest);
  if (details.registry) removeSkillRegistry(platformId);
}

async function main() {
  console.log();
  console.log(`${c.bold}${c.cyan}  doc-agent-ai${c.reset} ${c.gray}v${VERSION} — uninstaller${c.reset}`);
  console.log();

  const manifest = validateDist();

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
    warn("No supported platform detected.");
    info("Nothing to uninstall.");
    process.exit(0);
  }

  const installed = checkWhatIsInstalled(manifest, platforms);
  const installedPlatforms = Object.entries(installed)
    .filter(([, details]) => details.any)
    .map(([platformId]) => platformId);

  const rl = readline.createInterface({ input: process.stdin, output: process.stdout });

  try {
    if (installedPlatforms.length === 0) {
      console.log();
      warn("doc-agent-ai does not appear to be installed on detected platforms.");
      info("Nothing to uninstall.");
      rl.close();
      process.exit(0);
    }

    head("The following will be removed:");
    console.log(`${c.gray}  ─────────────────────────────────${c.reset}`);

    for (const platformId of installedPlatforms) {
      const details = installed[platformId];
      console.log(`  ${platformId}:`);
      if (details.skills.length > 0) subInfo(`Skills: ${details.skills.join(", ")}`);
      if (details.prompts.length > 0) subInfo(`Prompts: ${formatPromptDir(platformId)}`);
      if (details.commands.length > 0) subInfo(`Commands: ${details.commands.map((id) => `/${id}`).join(", ")}`);
      if (details.agents.length > 0) subInfo(`Agents: ${details.agents.join(", ")}`);
      if (details.registry) subInfo("Registry: .atl/skill-registry.md");
    }

    console.log();
    warn("Your documentation files are NOT affected.");
    console.log();

    const confirm = await ask(rl, `  ${c.bold}${c.red}Uninstall from all detected platforms?${c.reset} (y/N) `);
    if (confirm.trim().toLowerCase() !== "y") {
      info("Uninstall cancelled.");
      rl.close();
      process.exit(0);
    }

    for (const platformId of installedPlatforms) {
      uninstallPlatform(platformId, installed[platformId], manifest);
    }

    console.log();
    console.log(`${c.bold}${c.green}  ✔ Uninstall complete.${c.reset}`);
    dim("Restart your AI tool if it is currently running.");
    console.log();
  } finally {
    rl.close();
  }
}

main().catch((e) => {
  err(`Unexpected error: ${e.message}`);
  process.exit(1);
});
