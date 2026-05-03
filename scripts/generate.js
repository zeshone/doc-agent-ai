#!/usr/bin/env node

import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const ROOT = path.resolve(__dirname, "..");
const SRC_DIR = path.join(ROOT, "src");
const DIST_DIR = path.join(ROOT, "dist");
const SKILLS_DIR = path.join(ROOT, "skills");

function readText(filePath) {
  return fs.readFileSync(filePath, "utf8").replace(/^\uFEFF/, "");
}

function readJson(relativePath) {
  return JSON.parse(readText(path.join(SRC_DIR, relativePath)));
}

function ensureDir(dirPath) {
  fs.mkdirSync(dirPath, { recursive: true });
}

function emptyDir(dirPath) {
  fs.rmSync(dirPath, { recursive: true, force: true });
  ensureDir(dirPath);
}

function writeFile(filePath, content) {
  ensureDir(path.dirname(filePath));
  fs.writeFileSync(filePath, content.endsWith("\n") ? content : `${content}\n`);
}

function copyDirSync(src, dest) {
  ensureDir(dest);
  for (const entry of fs.readdirSync(src, { withFileTypes: true })) {
    const srcPath = path.join(src, entry.name);
    const destPath = path.join(dest, entry.name);
    if (entry.isDirectory()) {
      copyDirSync(srcPath, destPath);
    } else {
      ensureDir(path.dirname(destPath));
      fs.copyFileSync(srcPath, destPath);
    }
  }
}

function render(template, variables) {
  return template.replace(/\{\{([A-Z0-9_]+)\}\}/g, (_, key) => {
    if (!(key in variables)) {
      throw new Error(`Missing template variable: ${key}`);
    }
    return variables[key];
  });
}

function yamlList(items) {
  return items.map((item) => `  - ${item}`).join("\n");
}

function boolText(value) {
  return value ? "true" : "false";
}

const contentManifest = readJson("manifests/content.json");
const platformManifest = readJson("manifests/platforms.json");
const promptTemplate = readText(path.join(SRC_DIR, "templates", "prompt.md.tmpl"));
const commandTemplate = readText(path.join(SRC_DIR, "templates", "command.md.tmpl"));

emptyDir(DIST_DIR);

for (const role of contentManifest.roles) {
  const bodySource = readText(path.join(SRC_DIR, "content", role.content));

  for (const [platformId, platform] of Object.entries(platformManifest)) {
    const promptBody = render(bodySource, {
      BASE_PATH: contentManifest.placeholderBasePath,
      SKILL_PATH: `${platform.skillRoot}/${role.skill}/SKILL.md`,
      RULES_SKILL_PATH: `${platform.skillRoot}/${role.rulesSkill}/SKILL.md`,
      TECH_TEMPLATE_PATH: `${platform.skillRoot}/doc-tech/references/template.md`
    });

    writeFile(
      path.join(DIST_DIR, platform.promptDir, `${role.id}.md`),
      render(promptTemplate, { BODY: promptBody })
    );

    if (platform.agentDir) {
      const agentTemplate = readText(path.join(SRC_DIR, "templates", platform.agentTemplate));
      const tools = role.id === "doc-arch" ? platform.orchestratorTools : platform.agentTools;
      const agentsBlock = platformId === "copilot" && role.copilotChildren?.length
        ? `agents:\n${yamlList(role.copilotChildren)}\n`
        : "";
      const userInvocableLine = platformId === "qwen"
        ? `user-invocable: ${boolText(role.userInvocable)}\n`
        : "";
      const approvalMode = role.id === "doc-arch"
        ? platform.orchestratorApprovalMode ?? platform.approvalMode ?? "auto-edit"
        : platform.approvalMode ?? "auto-edit";

      writeFile(
        path.join(DIST_DIR, platform.agentDir, `${role.id}${platform.agentExtension}`),
        render(agentTemplate, {
          NAME: role.id,
          DESCRIPTION: role.description,
          TOOLS_YAML: yamlList(tools),
          USER_INVOCABLE: boolText(role.userInvocable),
          USER_INVOCABLE_LINE: userInvocableLine,
          AGENTS_BLOCK: agentsBlock,
          APPROVAL_MODE: approvalMode,
          BODY: promptBody
        })
      );
    }
  }
}

for (const command of contentManifest.commands) {
  const bodySource = readText(path.join(SRC_DIR, "content", command.content));
  const commandBody = render(bodySource, {
    BASE_PATH: contentManifest.placeholderBasePath
  });

  writeFile(
    path.join(DIST_DIR, platformManifest.opencode.commandDir, `${command.id}.md`),
    render(commandTemplate, {
      DESCRIPTION: command.description,
      AGENT: command.agent,
      BODY: commandBody
    })
  );
}

copyDirSync(SKILLS_DIR, path.join(DIST_DIR, "skills"));

const distManifest = {
  generatedAt: new Date().toISOString(),
  placeholderBasePath: contentManifest.placeholderBasePath,
  skills: contentManifest.skills,
  roles: contentManifest.roles.map((role) => ({
    id: role.id,
    description: role.description,
    hidden: role.hidden,
    mode: role.mode,
    opencodeTools: role.opencodeTools,
    promptFiles: Object.fromEntries(
      Object.entries(platformManifest).map(([platformId, platform]) => [platformId, path.join(platform.promptDir, `${role.id}.md`)])
    ),
    agentFiles: Object.fromEntries(
      Object.entries(platformManifest)
        .filter(([, platform]) => platform.agentDir)
        .map(([platformId, platform]) => [platformId, path.join(platform.agentDir, `${role.id}${platform.agentExtension}`)])
    )
  })),
  commands: contentManifest.commands.map((command) => ({
    id: command.id,
    description: command.description,
    agent: command.agent,
    file: path.join(platformManifest.opencode.commandDir, `${command.id}.md`)
  })),
  platforms: platformManifest
};

writeFile(path.join(DIST_DIR, "manifest.json"), JSON.stringify(distManifest, null, 2));

console.log("dist generated from src canonical content");
