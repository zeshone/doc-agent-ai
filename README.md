# doc-agent-ai

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)](https://go.dev)

**Multi-platform documentation workflow agent.** Single binary — download and run. No Node.js, no npm, no dependencies. Install once, document everywhere across opencode, Claude Code, GitHub Copilot, and Qwen Code.

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

- Canonical names: `doc-arch`, `doc-rec`, `doc-prd`, `doc-tech`, `doc-pti`
- Progressive technical depth: `rec` (executive) → `prd` (technical, clear) → `tech` (maximum precision, legible)
- Local-first issues. GitHub only on explicit request.
- Language detection on first contact. Documentation language asked once per system.
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
