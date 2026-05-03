# doc-agent-ai

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**Multi-platform documentation workflow agent.** Install once, document everywhere — from initial idea to implementation-ready issues, across opencode, Claude Code, GitHub Copilot, and Qwen Code.

---

## Quick start

```bash
git clone https://github.com/zeshone/doc-agent-ai.git
cd doc-agent-ai
npm run generate
node install.js
```

Restart your AI tool. Then type `/arch my-system` to start documenting.

---

## Platforms

| Platform | Support |
|----------|---------|
| [opencode](https://opencode.ai) | Prompts, commands, agents, skill-registry |
| [Claude Code](https://claude.ai) | Prompts, agents, skill-registry |
| [GitHub Copilot](https://github.com/features/copilot) | Prompts, agents |
| [Qwen Code](https://tongyi.aliyun.com) | Prompts, agents |

---

## Phases

| Command | Phase | Output |
|---------|-------|--------|
| `/rec <system>` | Requirements elicitation | `_requirements.md` |
| `/prd <system>` | Product Requirements Document | `_prd.md` |
| `/tech <system>` | Technical specification | `_tech-spec.md` |
| `/pti <system>` | Issues breakdown | `_issues.md` |
| `/arch <system>` | Full flow (all 4) | All of the above |

### Modules

```text
/rec <system>/<module>           ← individual phase for a module
/prd <system>/<module>
/tech <system>/<module>
/pti <system>/<module>
/mod <system> <module>           ← full module flow
```

---

## Update

```bash
git pull
npm run generate
node install.js
```

No data loss. Your documentation files live in your chosen projects path — never touched by install or uninstall.

---

## Maintenance

This project uses a **single-source → generate → install** architecture.

### Source of truth

```text
src/
  content/          ← canonical agent/prompt/command content
  manifests/        ← declarative metadata (platforms, roles, commands)
  templates/        ← platform-specific wrappers (.tmpl)
skills/             ← skill definitions shared across platforms
scripts/
  generate.js       ← builds dist/ from src/ + skills/
```

### Modify the agent

1. Edit files under `src/` or `skills/`.
2. Run `npm run generate` to rebuild `dist/`.
3. Run `node install.js` to apply changes locally.
4. Commit. `dist/` is git-ignored — only `src/` is versioned.

### Authoring conventions

- Canonical names: `doc-arch`, `doc-rec`, `doc-prd`, `doc-tech`, `doc-pti`
- Progressive technical depth: `rec` (executive) → `prd` (technical, clear) → `tech` (maximum precision, legible)
- Local-first issues. GitHub only on explicit request.
- Language detection on first contact. Documentation language asked once per system.
- Never assume missing context — ask or mark `TBD`.

---

## Uninstall

```bash
node uninstall.js
```

Removes skills, prompts, commands, and agent registrations from all detected platforms. Your documentation files are never touched.

---

## License

MIT © 2026 [Zesh-One](https://github.com/zeshone)
