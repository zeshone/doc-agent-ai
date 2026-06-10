# Changelog

Newer versions on top. Dates are when the tag was cut, not when the work landed on `main`.

For older releases without a section here, the GitHub Release notes have the details.

---

## v4.0.0 — Unreleased

### doc-reader conditional skill (PR 3 — FINAL SLICE)

Adds a new `doc-reader` skill installed exclusively in **in-project docs mode**.
It teaches agents in repos with `docs/doc-agent/` to use ONLY the compacted
`/doc-to-sdd` output as documentation context, avoiding context bloat from the
full normal-flow artifact tree.

#### doc-reader skill

`skills/doc-reader/SKILL.md` — LLM-first skill with:

- **Reads only**: `docs/doc-agent/agent_sdd_context_project/_sdd-context.md`
  (business layer) and `_sdd-tech-context.md` (technical layer).
- **Explicitly excludes**: `_prd.md`, `_tech-spec.md`, and all other files
  directly under `docs/doc-agent/` or its module/feature subdirectories.
- **Fallback**: if the context files are absent, the skill instructs the agent
  to suggest running `/doc-to-sdd` once — no fallback to the full docs tree.

#### Conditional install

`doc-reader` is marked via a new `conditionalSkills` field in `content.json`
and `DistManifest`. The install engine skips it when `--docs-mode vault`; it
is installed on every platform with a `SkillsDir` (opencode, Claude Code,
Copilot, Qwen, Pi) when in-project mode is selected.

#### Mode-switch cleanup

When reinstalling and switching from in-project → vault, the mode-switch hook
now calls `sweepDocReaderIfLeavingInProject`, which removes the `doc-reader`
skill dir from all installed platforms. The sweep is idempotent.

#### Registry and parity

A `doc-reader` row was added to `registryTemplate` (User Skills table) and a
`### doc-reader` compact rules block was added to the Compact Rules section.
Registry parity tests enforce the content.json ↔ registryTemplate invariant.

- `feat(skill)`: add doc-reader — in-project-only context skill
- `feat(registry)`: add doc-reader row + compact rules to registryTemplate (atomic with content.json)
- `feat(install)`: conditional install — skip doc-reader in vault mode
- `feat(install)`: mode-switch cleanup — sweepDocReaderIfLeavingInProject on in-project→vault

---

### 2b-remediation: overwrite confirmation, Go 1.25 pin, dead code removal

Resolves two CRITICAL and four WARNING/SUGGESTION items from the verify report.

#### C1 — Go version resolved: go.mod stays at `go 1.25.0`

`golang.org/x/sys v0.46.0` and `golang.org/x/term v0.44.0` (transitive deps of
charmbracelet) both declare `go 1.25.0` as their minimum. Lowering the directive
is not possible without downgrading those deps. CI and release workflows updated
to `go-version: '1.25'`; README badge updated to `1.25+`.
`go mod tidy` run — charmbracelet packages promoted from `// indirect` to direct
requires (W2 resolved).

#### C2 — Per-platform overwrite confirmation (spec F1 parity)

Restores the overwrite prompt from the old bufio flow. The wizard now inserts
a `stepOverwriteConfirm` step between platform selection and docs-mode when any
selected platform already has agents installed.

- **Wizard**: queues affected platforms in order; per-platform prompt "Overwrite
  X? (y/N)". `y` = consent (stored in `overwriteConsent` map); `n`/Enter = skip
  (platform deselected). Fresh installs skip this step entirely.
- **Headless**: `executeInstall` calls `checkAlreadyInstalled` per target; without
  consent (`plan.Overwrite[id]=true`) and without `--yes`, returns an error
  directing the user to pass `--yes` or use the TUI. `--yes` bypasses the check.
- `InstallPlan.Overwrite` is now populated by `BuildPlan()` and `runInstall()`.
- `checkAlreadyInstalled` was effectively dead (only `installInteractive` used it).
  This fix revives it on the live install paths (both TUI wizard and engine).

Tests (9 new, TDD RED → GREEN): wizard step appears/skips/proceeds, BuildPlan
populates Overwrite map, headless errors without `--yes`, headless proceeds with
`--yes`, engine respects the Overwrite map.

#### W4 — collectingReporter: progress output shown correctly in done step

The previous implementation stored a `*InstallModel` pointer in the reporter and
appended to it from the `tea.Cmd` goroutine. Because `handleKey` is a value
receiver, the pointer pointed to a detached copy; progress lines never reached
the rendered model. Fixed: `collectingReporter` now owns a `*[]string` inside the
`tea.Cmd` closure. Lines are returned via `installResultMsg.progressLines` and
merged into the live model by `Update`. Progress is displayed in the done step
(after the install completes) rather than during the progress step. No live
streaming during progress — that is the correct description of what ships.

#### S1 — Removed dead `installInteractive` and `normalizeBasePath`

`installInteractive` was fully dead (zero live callers since the Bubbletea wizard
replaced the bufio flow in PR 2b). Removed along with `normalizeBasePath` (only
used by `installInteractive`). `ask()` is kept — `uninstallInteractive` is its
only live caller. `checkAlreadyInstalled` is now live via the overwrite fix above.

#### S2 — Documented install vs. uninstall no-TTY asymmetry

Added a rationale comment in `main.go` at the uninstall routing block explaining
why install errors on no-TTY (needs user input: mode, path) while uninstall falls
back to the bufio flow (only needs a yes/no, safe to default).

#### W3 — gofmt applied

`tui_model.go`, `tui_styles.go`, `tui_model_test.go`, `overwrite_test.go`, and
`install.go` reformatted. `gofmt -l .` is now clean.

---

### Bubbletea installer TUI (PR 2b)

Replaces the hand-rolled `bufio` interactive flow with a Bubbletea wizard. The
wizard is isolated in `tui_*.go` files; the engine remains charm-free (enforced
by the purity guard from PR 2a).

#### Charm dependencies added

```
github.com/charmbracelet/bubbletea v1.3.10
github.com/charmbracelet/bubbles  (textinput only)
github.com/charmbracelet/lipgloss v1.1.0
golang.org/x/term                  (TTY detection)
```

**Build requirement:** `go 1.25.0` minimum (see C1 note above).

**Binary size (T2-1 measurement):** with deps, `CGO_ENABLED=0 go build -ldflags="-s -w"` produces
**≈5.3–5.4 MiB depending on platform** (up from 4.0 MiB before charm; CI gate: 6 MiB — gate is safe).

#### Install wizard

Seven-step wizard: platform selection → overwrite confirm (existing installs only) →
docs-mode → vault path (vault only) → confirm → progress → done.

- Config defaults pre-fill mode, vault path, and platform selection on reinstall.
- In-project mode skips the path entry step.
- Mode-switch notice displayed in the confirm step when mode changes from config.
- Engine output collected via `collectingReporter` and displayed in the done step.

#### Uninstall wizard

Three-step wizard: confirm (lists what will be removed) → progress → done.
Enter defaults to no; explicit `y` is required (destructive action guard).

**no-TTY behavior:** uninstall without a TTY falls back to the bufio
`uninstallInteractive` flow. Install without a TTY (and without flags) errors with
an actionable flag example. See `main.go` comment for the rationale.

#### TTY detection routing (main.go)

Decision order:
1. **Flags present** → headless path (existing, from PR 2a).
2. **No flags + TTY** → Bubbletea wizard (this PR).
3. **No flags + no TTY** → actionable error with flag examples; exit 1.

#### Tests

- `tui_model_test.go`: 15 direct `Model.Update()` state-transition tests (platform toggle,
  cursor nav, step advance, in-project skips path, vault includes path, config pre-fill,
  plan correctness, mode-switch notice display, TTY-error message components).
- `tui_flow_test.go`: 9 golden view snapshots (one per step + mode-switch + uninstall confirm)
  + 4 teatest full-flow tests (in-project, vault, config-prefilled, uninstall-cancel).
- `overwrite_test.go`: 9 overwrite-specific tests (see C2 above).

- `feat(tui)`: add Bubbletea install wizard (tui.go, tui_model.go, tui_steps.go, tui_styles.go)
- `feat(tui)`: add Bubbletea uninstall wizard (tui_uninstall.go)
- `feat(main)`: wire TTY detection — flags > TTY-TUI > no-TTY-error
- `test(tui)`: add 15 unit tests, 9 golden snapshots, 4 teatest flow tests

### Headless flags + engine seam (PR 2a)

The install command now accepts four flags that bypass the interactive/TUI flow
entirely and run a fully non-interactive install (CI-friendly):

```
doc-agent-ai install \
  --platforms opencode,claude \
  --docs-mode vault \
  --path /my/docs \
  --yes
```

**Decision order:** explicit flags > TTY-interactive (TUI in slice 2b) > error (no TTY, no flags).

#### Reporter seam

The engine (`installToPlatformWithReporter`) now routes all user-visible output
through a `Reporter` interface (Ok / Warn / ErrOut / Info / Dim / Head). The
default `stdoutReporter` is byte-identical to the former package-level helpers so
headless and interactive-without-TUI output is unchanged. Slice 2b will supply a
`bufferReporter` variant that collects structured results for the Bubbletea TUI
without corrupting its frame.

The original `installToPlatform(manifest, plat, basePath, distDir, globalMode...)` 
signature is preserved as a backward-compatible wrapper; all existing tests 
compile and pass unchanged.

#### executeInstall orchestrator

A new `executeInstall(manifest, plan, distDir, allPlatforms, reporter)` function
orchestrates a full install from an `InstallPlan`:

1. Resolves platform targets (nil = all detected).
2. Passes `plan.Mode` to `installToPlatformWithReporter` so `__DOC_AGENT_GLOBAL_MODE__` is correctly substituted in all installed files.
3. Writes `AppConfig` after a successful install (mode, path, platforms) so subsequent runs pre-fill defaults.
4. Fires the mode-switch hook when `plan.PrevMode != plan.Mode` — emits a non-migration notice and (in slice 3) sweeps `doc-reader` when switching in-project → vault.

#### Engine purity CI guard

A new `TestEnginePurity_NoCharmImports` test uses `go/parser` to assert that the
13 engine-layer files never import `charmbracelet/*`. This test is permanently
active and will fail immediately if charm code leaks into the engine. Charm imports
are allowed only in `tui_*.go` files (introduced in slice 2b).

`--yes` used alone (without other flags) runs a fully headless install using saved config defaults (vault mode and last-used path/platforms); it does not launch the interactive flow. It requires a prior install to have saved a config: on a fresh machine with no `~/.doc-agent-ai.json`, `--yes` alone fails with "vault mode requires a documentation base path" — pass `--path` (and optionally `--docs-mode`) explicitly in that case.

- `feat(install)`: add Reporter seam (`Reporter` interface, `stdoutReporter`, `bufferReporter`, `installToPlatformWithReporter`)
- `feat(install)`: add `executeInstall` orchestrator (platform resolution, mode wiring, config persistence, mode-switch hook seam)
- `feat(main)`: wire `--platforms`, `--docs-mode`, `--path`, `--yes` flags; route headless vs interactive by `hasInstallFlags`
- `test(engine)`: add purity guard asserting no charmbracelet imports in engine files

### Content sweep — preamble injection (PR 1b)

All 9 role prompts and all 11 opencode commands now carry a terse resolution preamble at generate time. The preamble teaches agents to:

1. Check `.doc-agent.json` at the project root first; read its `mode` field if present.
2. Fall back to the global mode (`__DOC_AGENT_GLOBAL_MODE__`, substituted at install time).
3. Use vault layout (`<resolved-base>/<system>/...`) in vault mode — the base is substituted from `__DOC_AGENT_GLOBAL_BASE__` (no trailing slash) at install time.
4. Use the simplified `docs/doc-agent/` layout in in-project mode (no `<system>` folder):
   `docs/doc-agent/_prd.md`, `docs/doc-agent/<module>/`, `docs/doc-agent/features/<slug>/`, `docs/doc-agent/agent_sdd_context_project/`.

The preamble is a single canonical block rendered from `src/templates/path-resolution.md.tmpl` via the `{{PATH_RESOLUTION}}` generate-time token. Existing body `{{BASE_PATH}}` tokens (`__DOC_AGENT_BASE_PATH__/`) are unchanged; the preamble uses the complementary `__DOC_AGENT_GLOBAL_BASE__` token (vault base without trailing slash) reserved for this purpose in the 1a install engine.

A post-install no-leak regression guard (`TestInstallNoRawTokenLeak`) was also added to permanently assert that no installed file retains any bare `__DOC_AGENT_` token after substitution.

- `feat(content)`: insert `{{PATH_RESOLUTION}}` preamble into all 9 role files and all 11 command files

### Resolution engine (PR 1a)

- `feat(config)`: add `~/.doc-agent-ai.json` persistent config (mode, path, platforms)
- `feat(resolve)`: add dual-mode path resolution engine (vault / in-project) with `.doc-agent.json` per-project marker support
- `feat(plan)`: add `InstallPlan` value object decoupling TUI from engine; headless `parsePlanFromFlags` for `--platforms`, `--docs-mode`, `--path`, `--yes`
- `feat(generate)`: inject `{{PATH_RESOLUTION}}` token into `buildBodyVars` and commands render loop (opt-in per content file; vault output byte-identical when token absent)
- `feat(install)`: extend `installToPlatform` placeholder substitution to ordered multi-token list adding `__DOC_AGENT_GLOBAL_MODE__`

### BREAKING CHANGE — opencode command rename

All 11 opencode commands now carry the `doc-` prefix. If you are using opencode and have customized any command triggers or keybindings pointing to the old bare names, update them to the new names below.

**Migration table:**

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

Non-opencode platforms (Claude Code, GitHub Copilot, Qwen Code, Pi) are unaffected — they use role IDs and agent files, which already carried the `doc-` prefix.

### Legacy file cleanup (install and uninstall)

`doc-agent-ai install` now automatically removes any bare-name legacy command files (e.g. `arch.md`, `prd.md`) from the opencode commands directory after copying the new `doc-*` files. This sweep is idempotent — missing legacy files produce no error.

`doc-agent-ai uninstall` removes both current `doc-*` files and any surviving legacy bare-name files, so users who never reinstalled under v4 are fully cleaned up.

### Rollback caveat

Downgrading to v3.x after a v4 install will leave `doc-*.md` files orphaned in your opencode commands directory (v3.x has no knowledge of the new names). To avoid this: run `doc-agent-ai uninstall` under v4 before downgrading.

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
