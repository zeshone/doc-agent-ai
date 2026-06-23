<p align="center">
  <img src="./docs/assets/brand/Zeen_Dark.png" alt="Zeen" width="860">
</p>

<h1 align="center">doc-agent-ai</h1>

<p align="center"><strong>Zeen-branded multi-platform documentation workflow agent.</strong><br>Single binary — download and run. No Node.js, no npm, no dependencies.</p>

<p align="center">
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="MIT License"></a>
  <a href="https://go.dev"><img src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go" alt="Go Version"></a>
</p>

<p align="center"><strong>Install once, document everywhere</strong> across opencode, Claude Code, GitHub Copilot, Qwen Code, and Pi.</p>

## Why it exists

`doc-agent-ai` gives teams one consistent documentation workflow across multiple AI coding platforms without dragging a JavaScript toolchain into the install path.

- One binary
- One command surface
- One documentation flow
- Multi-platform installation

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

The `doc-reader` skill is installed on every platform with a skills directory when **in-project docs mode** is selected. It teaches agents to use only the compacted `/doc-to-sdd` context files (`docs/doc-agent/agent_sdd_context_project/`) and to skip the full docs tree. Switching back to vault mode automatically removes it.

---

## Commands

| Subcommand | What it does |
|------------|--------------|
| `generate <dir>` | Write the rendered bundle to an explicit directory (dev/build only) |
| `install` | TUI installer — platform selection, docs mode (vault / in-project), overwrite confirmation |
| `uninstall` | TUI uninstaller — removes only doc-agent-ai artifacts, leaves your docs untouched |

### Headless install (CI / scripts)

Any install flag skips the TUI entirely:

```sh
doc-agent-ai install --platforms opencode,claude --docs-mode vault --path /home/you/docs/ --yes
```

| Flag | Meaning |
|------|---------|
| `--platforms <csv>` | Comma-separated platform IDs (`opencode,claude,copilot,qwen,pi`); omit for all detected |
| `--docs-mode <mode>` | `vault` (fixed base path) or `in-project` (`docs/doc-agent/` per repo) |
| `--path <path>` | Vault base path (required for vault mode unless saved in config) |
| `--yes` | Skip confirmations; alone, reuses saved config defaults — needs `--path` on a fresh machine |

The chosen mode and path persist in `~/.doc-agent-ai.json` and pre-fill the next install. A per-project `.doc-agent.json` marker (`{"mode": "in-project"}`) overrides the global mode for that repository.

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
2. Run `go run ./cmd/doc-agent-ai generate <dir>` to write a rendered bundle for inspection or packaging.
3. Run `go run ./cmd/doc-agent-ai` to open the installer UI, or `go run ./cmd/doc-agent-ai install --platforms ...` for headless install.
4. Commit. Generated output directories remain disposable — `src/` and `skills/` are still the source of truth.

### Build from source

```bash
git clone https://github.com/zeshone/doc-agent-ai.git
cd doc-agent-ai
go build -o doc-agent-ai .
go build -o doc-agent-ai ./cmd/doc-agent-ai
./doc-agent-ai
```

### Authoring conventions

- Canonical skill/role names: `doc-arch`, `doc-idea`, `doc-rec`, `doc-prd`, `doc-refinement`, `doc-tech`, `doc-ddd`, `doc-pti`, `doc-feat`, `doc-scope`, `doc-rec-lite`, `doc-prd-lite`, `doc-to-sdd`, `doc-reader` (conditional — in-project mode only)
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
./doc-agent-ai
```

Removes skills, prompts, commands, and agent registrations from all detected platforms. Your documentation files are never touched.

---

## License

MIT © 2026 [Zesh-One](https://github.com/zeshone)
