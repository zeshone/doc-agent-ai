# doc-agent-ai

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)](https://go.dev)

**Multi-platform documentation workflow agent.** Single binary — download and run. No Node.js, no npm, no dependencies. Install once, document everywhere across opencode, Claude Code, GitHub Copilot, and Qwen Code.

> **v3.2.0** — Surgical skill sanitization. Skills slimmed from 2,086 to 870 lines (58% reduction). ~70% fewer input tokens for agent instructions, ~20% fewer output tokens between sub-agents. Transformed from human manuals into precise, kitchen-specific instructions for AI agents. [Full changelog →](https://github.com/zeshone/doc-agent-ai/releases/tag/v3.2.0)

---

## Quick start

Download the latest binary for your platform from [Releases](https://github.com/zeshone/doc-agent-ai/releases/latest).

**Linux / macOS:**
```bash
chmod +x doc-agent-ai
./doc-agent-ai install
```

**Windows:**
```powershell
.\doc-agent-ai.exe install
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

## Commands

| Subcommand | What it does |
|------------|--------------|
| `generate` | Build `dist/` from embedded canonical content |
| `install` | Interactive installer — copies agents, skills, and commands to detected platforms |
| `uninstall` | Interactive uninstaller — removes only doc-agent-ai artifacts, leaves your docs untouched |

---

## Phases

The documentation workflow now has **6 phases**:

```text
idea -> rec -> prd -> refine -> tech -> pti
```

| Command | Phase | Output |
|---------|-------|--------|
| `/idea <system>` | Idea refinement — turn a vague concept into a clear product direction | Master index description (+ optional `_idea-brief.md`) |
| `/rec <system>` | Requirements elicitation | `_requirements.md` |
| `/prd <system>` | Product Requirements Document | `_prd.md` |
| `/refine <system>` | User story audit against INVEST criteria | Updated `_prd.md` user stories |
| `/refine` | Standalone refinement of a single user story | Inline refined story |
| `/tech <system>` | Technical specification | `_tech-spec.md` |
| `/pti <system>` | Issues breakdown | `_issues.md` |
| `/arch <system>` | Full flow (all 6) | All of the above |

### Modules

```text
/idea <system>/<module>          ← individual phase for a module
/rec <system>/<module>           ← individual phase for a module
/prd <system>/<module>
/refine <system>/<module>
/tech <system>/<module>
/pti <system>/<module>
/mod <system> <module>           ← full module flow (all 6 phases)
```

---

## Update

Download the latest binary from [Releases](https://github.com/zeshone/doc-agent-ai/releases/latest) and run `install` again. Your documentation files live in your chosen projects path — never touched by install or uninstall.

---

## Maintenance

This project uses a **single-source → embed → generate → install** architecture. All canonical content lives in `src/` and `skills/`, embedded into the binary at compile time via `//go:embed`.

### Source of truth

```text
src/
  content/          ← canonical agent/prompt/command content
  manifests/        ← declarative metadata (platforms, roles, commands)
  templates/        ← platform-specific wrappers (.tmpl)
skills/             ← skill definitions shared across platforms
```

### Modify the agent

1. Edit files under `src/` or `skills/`.
2. Run `go run . generate` to rebuild `dist/`.
3. Run `go run . install` to apply changes locally.
4. Commit. `dist/` is git-ignored — only `src/` is versioned.

### Build from source

```bash
git clone https://github.com/zeshone/doc-agent-ai.git
cd doc-agent-ai
go build -o doc-agent-ai .
./doc-agent-ai install
```

### Authoring conventions

- Canonical names: `doc-arch`, `doc-idea`, `doc-rec`, `doc-prd`, `doc-refinement`, `doc-tech`, `doc-pti`
- Progressive workflow depth: `idea` (product framing) → `rec` (executive/business elicitation) → `prd` (technical but clear) → `refine` (story quality gate) → `tech` (maximum precision, still legible) → `pti` (executable issues)
- Local-first issues. GitHub only on explicit request.
- Language detection on first contact. Documentation language asked once per system.
- **All skills in English.** The author works primarily in Spanish — occasional Spanish fragments in skill instructions are unintentional artifacts of the authoring workflow. All skill content must be delivered in English for global accessibility. PRs with non-English skill content will be flagged.
- Never assume missing context — ask or mark `TBD`.

---

## Uninstall

```bash
./doc-agent-ai uninstall
```

Removes skills, prompts, commands, and agent registrations from all detected platforms. Your documentation files are never touched.

---

## License

MIT © 2026 [Zesh-One](https://github.com/zeshone)
