# doc-agent-ai

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://go.dev)

**Multi-platform documentation workflow agent.** Single binary — download and run. No Node.js, no npm, no dependencies. Install once, document everywhere across opencode, Claude Code, GitHub Copilot, Qwen Code, and Pi.

> **v4.0.0** — All opencode commands now carry the `doc-` prefix (e.g. `/doc-arch`, `/doc-prd`). This is a breaking change for opencode users — see the [migration table in CHANGELOG →](./CHANGELOG.md).

---

## Quick start

**macOS (Homebrew):**

```sh
brew tap zeshone/tap
brew install doc-agent-ai
doc-agent-ai install
```

**Windows (Scoop):**

```powershell
scoop bucket add zeshone https://github.com/zeshone/scoop-bucket
scoop install doc-agent-ai
doc-agent-ai install
```

**Linux or direct download:** grab the binary for your platform from [Releases](https://github.com/zeshone/doc-agent-ai/releases/latest), then:

```sh
chmod +x doc-agent-ai   # not needed on Windows
./doc-agent-ai install
```

Restart your AI tool. Then type `/doc-arch my-system` to start documenting.

---

## Platforms

| Platform | Support |
|----------|---------|
| [opencode](https://opencode.ai) | Prompts, commands, agents, skill-registry |
| [Claude Code](https://claude.ai) | Prompts, agents, skill-registry |
| [GitHub Copilot](https://github.com/features/copilot) | Prompts, agents |
| [Qwen Code](https://tongyi.aliyun.com) | Prompts, agents |
| [Pi](https://github.com/earendil-works/pi) | Prompts, skills, skill-registry |

Pi consumes role definitions as prompt templates (no separate agent registry). The installer detects `~/.pi/agent` or `pi` on `PATH`. Override with `--pi-path <path>` or set `PI_CODING_AGENT_DIR`.

---

## Commands

| Subcommand | What it does |
|------------|--------------|
| `generate` | Build `dist/` from embedded canonical content |
| `install` | Interactive installer — copies agents, skills, and commands to detected platforms |
| `uninstall` | Interactive uninstaller — removes only doc-agent-ai artifacts, leaves your docs untouched |

---

## Phases

The documentation workflow has **7 phases** (one optional):

```text
idea -> rec -> prd -> refine -> tech -> [ddd] -> pti
              └─────────────────────────────────┘
                            ddd is optional
```

| Command | Phase | Output |
|---------|-------|--------|
| `/doc-idea <system>` | Idea refinement — turn a vague concept into a clear product direction | Master index description (+ optional `_idea-brief.md`) |
| `/doc-rec <system>` | Requirements elicitation | `_requirements.md` |
| `/doc-prd <system>` | Product Requirements Document | `_prd.md` |
| `/doc-refine <system>` | User story audit against INVEST criteria | Updated `_prd.md` user stories |
| `/doc-refine` | Standalone refinement of a single user story | Inline refined story |
| `/doc-tech <system>` | Technical specification | `_tech-spec.md` |
| `/doc-ddd <system>` | Optional — Database Design Document | `_db-design.md` |
| `/doc-pti <system>` | Issues breakdown | `_issues.md` |
| `/doc-arch <system>` | Full flow (all 6 + optional ddd) | All of the above |
| `/doc-to-sdd <system>` | Standalone — compact docs into LLM-optimized SDD context files | One or both: `agent_sdd_context_project/_sdd-context.md`, `agent_sdd_context_project/_sdd-tech-context.md` (depends on available source artifacts) |

### Modules

```text
/doc-idea <system>/<module>      ← individual phase for a module
/doc-rec <system>/<module>       ← individual phase for a module
/doc-prd <system>/<module>
/doc-refine <system>/<module>
/doc-tech <system>/<module>
/doc-ddd <system>/<module>       ← optional DB design for a module
/doc-pti <system>/<module>
/doc-mod <system> <module>       ← full module flow (all phases, ddd optional)
```

---

## Update

```sh
brew upgrade doc-agent-ai     # macOS
scoop update doc-agent-ai     # Windows
```

Or download the new binary from [Releases](https://github.com/zeshone/doc-agent-ai/releases/latest) and run `install` again. Your documentation files live in your chosen projects path — never touched by install or uninstall.

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

- Canonical skill/role names: `doc-arch`, `doc-idea`, `doc-rec`, `doc-prd`, `doc-refinement`, `doc-tech`, `doc-ddd`, `doc-pti`, `doc-feat`, `doc-scope`, `doc-rec-lite`, `doc-prd-lite`, `doc-to-sdd`
- Command names mirror their skill with the `doc-` prefix, with two deliberate divergences: `/doc-refine` triggers the `doc-refinement` skill, and `/doc-mod` (module flow) is handled by `doc-arch`
- Progressive workflow depth: `idea` (product framing) → `rec` (executive/business elicitation) → `prd` (technical but clear) → `refine` (story quality gate) → `tech` (maximum precision, still legible) → [`ddd` (structured data design, ERD, constraints, rationale)] → `pti` (executable issues)
- `ddd` is optional. Triggered explicitly, by hard signals (schema files, DBMS mentions), or by orchestrator prompt between `tech` and `pti`. Dismissed for in-memory or ephemeral systems.
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
