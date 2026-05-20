# Changelog

Newer versions on top. Dates are when the tag was cut, not when the work landed on `main`.

For older releases without a section here, the GitHub Release notes have the details.

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
