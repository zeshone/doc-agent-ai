# Changelog

Newer versions on top. Dates are when the tag was cut, not when the work landed on `main`.

For older releases without a section here, the GitHub Release notes have the details.

---

## v4.0.0 — 2026-06-10

A major release: a TUI installer, your choice of where documentation lives, a new context skill for in-project docs, and a breaking rename of the opencode commands. Non-opencode users upgrade transparently; opencode users have one rename to absorb (see below).

### BREAKING CHANGE — opencode commands now carry the `doc-` prefix

All 11 opencode commands were renamed so they are recognizable among the many commands agents expose. If you use opencode and customized any command triggers or keybindings, update them:

| Old command | New command |
|-------------|-------------|
| `/arch` | `/doc-arch` |
| `/idea` | `/doc-idea` |
| `/rec` | `/doc-rec` |
| `/prd` | `/doc-prd` |
| `/refine` | `/doc-refine` |
| `/tech` | `/doc-tech` |
| `/pti` | `/doc-pti` |
| `/mod` | `/doc-mod` |
| `/feat` | `/doc-feat` |
| `/ddd` | `/doc-ddd` |
| `/to-sdd` | `/doc-to-sdd` |

Reinstalling under v4 removes the old bare-name command files automatically — no orphans left behind. Non-opencode platforms (Claude Code, GitHub Copilot, Qwen Code, Pi) are unaffected; they already used `doc-`-prefixed roles and agent files.

### New: Zeen-branded welcome screen

The installer wizard opens with a Zeen block-art logo rendered from the brand palette (#00A5E7 blue, #0D1012 dark, #E5E8EA grey), a one-line agent description, and footer buttons `[Continue]` and `[Quit]`. This replaces the plain text banner and gives the installer a recognizable branded entry point.

### New: TUI installer

`doc-agent-ai install` and `uninstall` now run an interactive wizard instead of line-by-line prompts:

- **Install** — a button-driven wizard with full Back navigation: welcome → select platforms (already-installed ones are tagged) → choose docs mode → enter the vault path (vault mode only) → if some selected platforms already have Zeen, one consolidated screen to *overwrite all* or *install only missing* → animated progress with a per-platform checklist → done (waits for a keypress so you can read the summary). Your previous choices pre-fill the next run.
- **Uninstall**: shows exactly what will be removed (your documentation is never touched) and requires an explicit `y`.

The wizard appears when you run from a terminal. In a non-interactive environment (CI, pipes), pass flags instead — see *Headless install* below.

### New: choose where documentation lives

Two documentation modes, selectable in the wizard:

- **Vault** — the original behavior: all docs under one configurable base path, organized as `<base>/<system>/...`. Ideal for an Obsidian vault that spans many projects.
- **In-project** — docs live inside the repository under `docs/doc-agent/` (simplified layout, no `<system>` folder): `docs/doc-agent/_prd.md`, `docs/doc-agent/<module>/`, `docs/doc-agent/features/<slug>/`, `docs/doc-agent/agent_sdd_context_project/`.

Your choice persists in `~/.doc-agent-ai.json` and pre-fills future installs. A per-project `.doc-agent.json` marker (`{"mode": "in-project"}`) overrides the global mode for that repository — so one machine can use a vault by default and a specific repo can keep its docs in-tree. Installed prompts resolve the mode automatically: they check the project marker first, then fall back to the global mode.

### New: `doc-reader` skill for in-project mode

When you install in in-project mode, a new `doc-reader` skill ships to every platform that supports skills. It tells agents working in that repository to use **only** the compacted `/doc-to-sdd` output (`docs/doc-agent/agent_sdd_context_project/_sdd-context.md` and `_sdd-tech-context.md`) as documentation context — the full human-readable artifacts (`_prd.md`, `_tech-spec.md`, …) are explicitly excluded to keep agent context lean. If the compacted files don't exist yet, the skill suggests running `/doc-to-sdd` once rather than loading the whole docs tree.

The skill is installed only in in-project mode; switching a reinstall back to vault mode removes it automatically.

### New: headless install for CI and scripts

Any install flag bypasses the wizard for a fully non-interactive run:

```sh
doc-agent-ai install \
  --platforms opencode,claude \
  --docs-mode vault \
  --path /home/you/docs/ \
  --yes
```

| Flag | Meaning |
|------|---------|
| `--platforms <csv>` | Platform IDs (`opencode,claude,copilot,qwen,pi`); omit for all detected |
| `--docs-mode <mode>` | `vault` or `in-project` |
| `--path <path>` | Vault base path (required for vault mode unless saved in config) |
| `--yes` | Skip confirmations (required to overwrite existing installs non-interactively) |

`--yes` alone reuses saved config defaults; on a fresh machine with no config, pass `--path` (and optionally `--docs-mode`) explicitly.

### Build requirement

doc-agent-ai now requires **Go 1.25+** to build from source (a transitive dependency of the TUI libraries sets the floor). Prebuilt binaries from Releases, Homebrew, and Scoop are unaffected. The stripped binary is ≈5.4–5.5 MiB depending on platform.

### Rollback caveat

Downgrading to v3.x after a v4 install leaves `doc-*.md` command files orphaned in your opencode commands directory (v3.x doesn't know the new names). To avoid this, run `doc-agent-ai uninstall` under v4 before downgrading.

---

## v3.6.0 — 2026-06-03

### Added

- **`/to-sdd` command** — standalone compaction of human-readable docs into LLM-optimized SDD context.

  Usage: `/to-sdd <system>`

  Produces one or both files in `agent_sdd_context_project/` (depending on available source artifacts):

  - `_sdd-context.md` — business layer context
  - `_sdd-tech-context.md` — technical layer context

  Positioning: English-only output, token-efficient, clarity-first, and directly routed to `doc-to-sdd` (not part of the `/arch` flow).

  Tests / registry parity: embedded content validation covers direct `/to-sdd` → `doc-to-sdd` routing, and registry parity guards keep the skill manifest and registry template in sync.

## v3.5.0 — 2026-05-29

### Added

- **`/ddd` command** — optional Database Design Documentation step.

  Usage: `/ddd <system>` or `/ddd <system>/<module>`

  Produces `<sistema>_db-design.md` with:

  - Mermaid `erDiagram` ERD
  - Schema details per entity (columns, types, constraints, PK/FK)
  - Indexes, domain constraints, referential integrity rules
  - Design rationale table (normalization choices, trade-offs)
  - Migrations and lifecycle policy
  - Security model (audit fields, access control, encryption)
  - Visible `TBD` / `Open Decisions` section

  DBMS-agnostic: relational, document, key-value, embedded — all supported.

  Accepts a normalization sanity check (1NF → 2NF → 3NF) and surfaces violations as tagged **Design Issues**.


  New skill registered: `doc-ddd`.


- **Optional step in `arch` / `mod` flows** — between `tech` and `pti`.

  After `/tech`, the orchestrator asks `"¿Querés documentar el diseño de la base de datos?"` unless the user already excluded it or hard triggers apply.

  Hard triggers (launch without asking):

  - User explicitly invokes `/ddd`
  - Project contains persistence artifacts: `.sql`, `migrations/`, `schema.prisma`, `models/`
  - Tech spec mentions DBMS technology (PostgreSQL, MySQL, MongoDB, SQLite, etc.)
  - User states explicit intent: "documentar la base de datos", "db design"

  Soft trigger: prompt between `tech` and `pti` when data layer is present in the tech spec.


  Dismiss triggers: explicit exclusion ("no necesitamos base de datos", "in-memory only", "skip ddd") or ephemeral systems with no persistence intent.


- **DDD Decision Triggers table** in `doc-arch` skill — documents the full activation matrix for the optional step.

- **All 5 platforms** (opencode, copilot, claude, qwen, pi) generate `doc-ddd` prompts, agents, and skill registry entries on every `go run . generate`.


- **Registry parity test extended** — `doc-ddd` is now part of the manifest ↔ registryTemplate guard.

---


## v3.4.0 — 2026-05-20

### Added

- **`/feat` command** — documents a legacy feature end-to-end in a single invocation.

  Usage: `/feat <ruta-legacy> <description> [--scope local <path> | cross <pattern> | none]`

  The command runs a four-phase mini-flow:

  1. **Scope classification** (`doc-scope`) — reads the top-level structure of the legacy directory (bounded, no recursion) and proposes a scope mode (`local`, `cross`, or `none`). Skipped when `--scope` is provided explicitly.
  2. **Requirements** (`doc-rec-lite`) — Volere-lite elicitation, single-feature scope only.
  3. **PRD** (`doc-prd-lite`) — executive summary, flows, user stories with G/W/T acceptance criteria, non-goals, and a risks section.
  4. **Issues** (`doc-pti`, always) — executable local issues; optional `doc-tech` fires first when the PRD risks are high/medium or scope is `cross`.

  Output lands under `<BASE_PATH><sistema>-features/<slug>/` with a master index file using `[[links]]`.

  Four new skills registered: `doc-feat`, `doc-scope`, `doc-rec-lite`, `doc-prd-lite`.

- **Registry parity test** — `TestRegistryParity_AllManifestSkillsInRegistry` and `TestRegistryParity_NoOrphanEntriesInRegistry` enforce that `content.json` and `registryTemplate()` stay in sync. Adding a skill to the manifest without updating the registry now fails CI immediately.

---

## v3.3.1 — 2026-05-19

### Fixed

- `doc-arch` SKILL.md frontmatter failed to parse on strict YAML readers because the `description` value contained an unquoted `Trigger: ...` (colon + space), which the YAML 1.2 spec treats as a nested mapping inside a compact scalar. The value is now single-quoted.

### Added

- Build-time skill frontmatter lint. `generate()` now runs `lintEmbeddedSkills` as its first step and aborts with a precise `file:line` error before touching `dist/` if any embedded `SKILL.md` repeats the same class of bug. Covered by unit tests in `generate_test.go`.

---

## v3.3.0 — 2026-05-18

### Pi is now a first-class platform

doc-agent-ai installs documentation roles into [Pi](https://github.com/earendil-works/pi). Pi reads role definitions from prompt templates under `~/.pi/agent/prompts/` — it has no native agent registry like opencode and no `agents/<role>.md` directory like Claude/Qwen/Copilot, so we treat it as a prompts-only platform and skip the agent-file generation step.

Detection looks for `~/.pi/agent` first, then falls back to `pi` on `PATH` so a fresh install (where the home dir doesn't exist yet) is still recognised.

Three ways to point at a non-standard Pi install, in priority order:

1. `--pi-path <path>` on the command line
2. `PI_CODING_AGENT_DIR` environment variable (Pi itself honours this)
3. Default `~/.pi/agent`

### Install from Homebrew and Scoop

Pre-3.3.0 the only install path was downloading a binary from GitHub Releases. Now:

**macOS:**

```sh
brew tap zeshone/tap
brew install doc-agent-ai
```

**Windows:**

```powershell
scoop bucket add zeshone https://github.com/zeshone/scoop-bucket
scoop install doc-agent-ai
```

Both are wired into the release pipeline. Tagging `v*` on the repo publishes the binary to GitHub Releases, updates the Homebrew formula at [zeshone/homebrew-tap](https://github.com/zeshone/homebrew-tap), and updates the Scoop manifest at [zeshone/scoop-bucket](https://github.com/zeshone/scoop-bucket) — all in one workflow run.

Direct download of `doc-agent-ai_<version>_<os>_<arch>.{tar.gz,zip}` from Releases still works.

### Documentation workflow grew to 6 phases

Two new phases sit alongside the original four:

```text
idea → rec → prd → refine → tech → pti
```

- **`/idea`** (new) — pure product discovery. No stack, no APIs, no databases. Produces the master index description and optionally `_idea-brief.md`.
- **`/refine`** (new) — quality gate. Audits user stories against INVEST criteria, never changes scope without confirmation.

The README, the `doc-arch` skill, and the manifest were already in sync on 6 phases internally, but the `arch` command template and a couple of step numbers in the manifest still advertised 4. That mismatch is fixed.

### Platform detection: XDG, portable VS Code, `--copilot-path`

OpenCode now honours `XDG_CONFIG_HOME` on Linux. If `$XDG_CONFIG_HOME` is set, OpenCode is detected at `$XDG_CONFIG_HOME/opencode/opencode.json` instead of `~/.config/opencode/opencode.json`. macOS and Windows behaviour is unchanged.

GitHub Copilot detection gained two paths:

- If `~/.copilot` is missing but `code` is on `PATH`, doc-agent-ai queries `code --locate-shell-integration-path bash` and looks for the Copilot Chat extension data under a portable VS Code install root. If found, install proceeds with that path.
- A new `--copilot-path <path>` flag overrides both auto-detection paths. Useful for non-standard installs.

The standard `~/.copilot` path now also requires `code` to be on `PATH` to be considered detected, so stale `~/.copilot` directories left over from uninstalled VS Code don't cause false positives.

### Under the hood

- **CI gates** on every push and pull request to `main`: `go test ./...`, binary size < 6 MB, and a content-language check that flags Spanish fragments or CJK/Arabic/Cyrillic in embedded `src/` and `skills/` files (with explicit allowlist for the few intentionally bilingual examples).
- **Release pipeline** built on goreleaser. Tag `v*`, get six binaries (linux/darwin/windows × amd64/arm64), `.deb`/`.rpm` for Linux, archives, checksums, GitHub Release, Homebrew formula, and Scoop manifest. Pre-release tags (`-rc`, `-beta`) are detected automatically.

### Fixes

- Cross-platform archive naming uses lowercase OS names consistently (`windows_amd64.zip`, not `Windows_amd64.zip`).
- Single archive per OS/Arch combination — earlier the config produced both a `.tar.gz` and a `.zip` for every target, which confused Homebrew.

### Removed

- `TestGenerate_Integration` — dead test from the Node→Go migration. It compared the Go generator's output against a `dist/` produced by `npm run generate`, which hasn't existed since `b24330d`.

---

## v3.2.0 — 2026-03

Surgical skill sanitization. Skill files slimmed from 2,086 to 870 lines (58% reduction) by rewriting them as precise, agent-targeted instructions instead of human-readable manuals. Net effect: ~70% fewer input tokens for agent instructions, ~20% fewer output tokens between sub-agents.

[Full notes →](https://github.com/zeshone/doc-agent-ai/releases/tag/v3.2.0)

---

## v3.1.x and earlier

See [git history](https://github.com/zeshone/doc-agent-ai/commits/main) and [GitHub Releases](https://github.com/zeshone/doc-agent-ai/releases).
