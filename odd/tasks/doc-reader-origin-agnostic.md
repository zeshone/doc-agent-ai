# doc-reader works regardless of where the documentation lives

## Objective

Make the `doc-reader` skill usable in both vault and in-project mode, so every
supported agent knows that a compacted context exists and that it is the primary
source, whatever the documentation origin.

## Problem

`doc-reader` tells an agent to read only the two `doc-to-sdd` compacted files and
never the full docs tree. It hardcodes `docs/doc-agent/agent_sdd_context_project/`,
which is only correct in in-project mode. The workaround was to not install it at
all in vault mode: `src/manifests/content.json` lists it under `conditionalSkills`
and `internal/install/install.go:341` skips it unless the resolved global mode is
`in-project`. `internal/install/execute_install.go:147`
(`sweepDocReaderIfLeavingInProject`) even deletes it on a mode switch to vault.

The gate is not laziness. It is the honest consequence of a real gap, confirmed by
probe on 2026-09-18:

- **in-project**: the node name does not affect resolution. `--node loquesea` and
  `--node otronombre` both resolve `docsRoot` to `<ProjectRoot>/docs/doc-agent`.
- **vault**: the node determines everything (`<vault>/<node>/`), and nothing in a
  code repository records which node it belongs to. `doc-agent-ai status` with no
  `--node` errors: "status needs --node <system[/module[/submodule]]>".

So an agent working inside a code repo cannot learn its vault node, and a skill
that cannot name the node cannot ask the program for paths.

## Why

The user is testing a cheap hypothesis: that the compacted context alone is enough
working context, with the strict no-fallback rule kept exactly as written. That
test cannot run today — the user's global mode is `vault`, and `doc-reader` is
installed on none of their three platforms (claude, opencode, pi). The experiment
would report failure for a reason unrelated to the hypothesis.

## Scope

Authorized: make the node resolvable from the project, route the skill through the
program instead of hardcoded paths, and install the skill unconditionally.

Out of scope, deliberately:
- The escalation/fallback design (read the full docs when the guide is not enough).
  The user chose to keep the strict rule and revisit only if the experiment fails.
- The addendum mechanism for decisions taken during development.
- Who writes `.doc-agent.json` long-term. Today the model writes it following
  prose at `skills/doc-arch/SKILL.md:105`; the binary only reads it. Left open.

## Constraints

- The program holds state; prose routes. No path may be baked into a skill.
- The marker is additive by design (`internal/pipeline/resolve.go:59-63`,
  "Other keys are ignored so the file stays additive"), so adding a key is safe.
- Artifacts in English. Conventional commits.

## Resolved configuration

- TDD: **strict, enabled** (source: CLAUDE.md session config). RED before GREEN.
- Test runner: `go test ./...`
- Additional checks: `gofmt -l .` (must be empty), `go vet ./...`
- Delivery strategy: `ask-on-risk` (default). Forecast ~300-500 authored changed
  lines across three tasks; ask before exceeding ~400 on the accumulated branch.

## Tasks

- [ ] **T1 — The marker carries the node; `--node` becomes optional.**
  Add `Node string` to the `marker` struct. When `--node` is omitted, resolve it
  from `.doc-agent.json`. When neither exists, fail with a message naming the fix
  rather than guessing. Route: delegated writer (touches CLI + resolve + tests).
  Checks: `go test ./...`, `gofmt -l .`, `go vet ./...`.

- [ ] **T2 — `doc-reader` routes through the program.**
  Remove the hardcoded `docs/doc-agent/agent_sdd_context_project/` path. Instruct
  the agent to ask the program for the resolved paths and read exactly what it
  returns, honoring `sddContext.state`. Keep the strict rule unchanged: only the
  compacted files, never the full docs tree. Update the assertions in
  `doc_reader_test.go` that pin the old path. Route: delegated writer.
  Checks: `go test ./...`, `gofmt -l .`, `go vet ./...`.

- [ ] **T3 — Install `doc-reader` unconditionally.**
  Drop `conditionalSkills` from `src/manifests/content.json`, the gate at
  `internal/install/install.go:341`, and `sweepDocReaderIfLeavingInProject` with
  its mode-switch hook. Update the tests that assert the conditional behavior:
  `internal/install/doc_reader_install_test.go`,
  `internal/install/doc_reader_sweep_test.go`, and
  `TestConditionalSkillsSubsetOfSkills` in `doc_reader_test.go`.
  Route: delegated writer. Checks: `go test ./...`, `gofmt -l .`, `go vet ./...`.

## Acceptance criteria

1. In vault mode, an agent in a code repo carrying a marker with its node can
   obtain the compacted context paths from the program without being told the node.
2. In in-project mode, behavior is unchanged.
3. `doc-reader` contains no filesystem path to the docs tree.
4. `doc-reader` is installed on every platform regardless of mode.
5. The strict compacted-only rule is preserved verbatim in meaning.
6. `go test ./...`, `gofmt -l .` and `go vet ./...` all clean.

## Progress

Not started. Branch `feat/doc-reader-origin-agnostic` cut from `main`.

## Next step

T1.
