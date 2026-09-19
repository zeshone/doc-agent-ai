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

- [x] **T1 — The marker carries the node; `--node` becomes optional.**
  Add `Node string` to the `marker` struct. When `--node` is omitted, resolve it
  from `.doc-agent.json`. When neither exists, fail with a message naming the fix
  rather than guessing. Route: delegated writer (touches CLI + resolve + tests).
  Checks: `go test ./...`, `gofmt -l .`, `go vet ./...`.

- [x] **T2 — `doc-reader` routes through the program.**
  Remove the hardcoded `docs/doc-agent/agent_sdd_context_project/` path. Instruct
  the agent to ask the program for the resolved paths and read exactly what it
  returns, honoring `sddContext.state`. Keep the strict rule unchanged: only the
  compacted files, never the full docs tree. Update the assertions in
  `doc_reader_test.go` that pin the old path. Route: delegated writer.
  Checks: `go test ./...`, `gofmt -l .`, `go vet ./...`.

- [x] **T3 — Install `doc-reader` unconditionally.**
  Drop `conditionalSkills` from `src/manifests/content.json`, the gate at
  `internal/install/install.go:341`, and `sweepDocReaderIfLeavingInProject` with
  its mode-switch hook. Update the tests that assert the conditional behavior:
  `internal/install/doc_reader_install_test.go`,
  `internal/install/doc_reader_sweep_test.go`, and
  `TestConditionalSkillsSubsetOfSkills` in `doc_reader_test.go`.
  Route: delegated writer. Checks: `go test ./...`, `gofmt -l .`, `go vet ./...`.

- [ ] **T4 — `doctor` repairs a pre-v5 compacted context.**
  Added 2026-09-18 at the user's request, after probing the real vault found
  three of four `Deze3.0` modules carrying bare-named context files
  (`_sdd-context.md`) with no manifest, written by the pre-v5 skill. In vault
  mode the program looks for `<prefix>_sdd-context.md`, so those report
  `absent` while the files sit there. `doctor` currently has zero awareness of
  the SDD context: `grep -n "sdd\|SDD" internal/pipeline/doctor.go` returns
  nothing, and it never inspects `agent_sdd_context_project/`.

  **What doctor CAN do**: detect legacy-named outputs with no manifest, rename
  them to the canonical `<prefix>_` names, and record an adoption for the
  context.

  **What doctor MUST NOT do, and why**: it cannot write a manifest claiming
  sources and fingerprints. The manifest records which artifacts were read and
  their hashes *at compaction time*; hashing today's sources would claim the
  compaction saw today's bytes. That is inventing provenance — the same failure
  the `Doctor` doc comment already refuses for answer records ("generating them
  from the artifacts would invent quotes attributed to them"). So it cannot
  produce `fresh`; it produces the SDD-context equivalent of `adopted`:
  present, provenance explicitly unverified.

  It also cannot *regenerate*. Compaction means authoring prose, which only a
  model can do through `/doc-to-sdd`. The adopted state is precisely the signal
  that `/doc-to-sdd` should be run.

  Route: delegated writer. Checks: `go test ./...`, `gofmt -l .`, `go vet ./...`.

  **Delivery note**: T4 is a real feature with its own tests and will likely
  push the branch past the ~400 authored-line budget. Strategy is `ask-on-risk`,
  so ask before starting T4 whether to chain it as a second PR.

## Acceptance criteria

1. In vault mode, an agent in a code repo carrying a marker with its node can
   obtain the compacted context paths from the program without being told the node.
2. In in-project mode, behavior is unchanged.
3. `doc-reader` contains no filesystem path to the docs tree.
4. `doc-reader` is installed on every platform regardless of mode.
5. The strict compacted-only rule is preserved verbatim in meaning.
6. `go test ./...`, `gofmt -l .` and `go vet ./...` all clean.
7. A node whose compacted context was written before v5 is visible to the
   program after `doctor --apply`, reported with its provenance unverified
   rather than as `fresh`.

## Progress

**T1 done** — commit `93f1767`. The marker gained a `node` key; `status` resolves
it when `--node` is omitted; explicit `--node` wins; neither present fails naming
both fixes. `readMarker` was extracted so mode and node parse the file once, and a
malformed marker stays an error rather than collapsing into a silent "no node".

Evidence, verified by the parent rather than taken from the writer's report:
- Behavioral RED isolated by reverting only the `RunStatus` wiring while keeping
  the new helper: 3 targeted tests failed for the right reasons, 3 pre-existing
  explicit-`--node` tests kept passing. The writer's own RED was a compile error,
  which proves a function is missing, not that behavior is.
- Acceptance criterion 1 proved against the real vault: a scratch repo carrying
  `{"mode":"vault","node":"Deze3.0"}` and no `--node` resolved `docsRoot` to
  `/home/zesh-one/src/Obsidian/DevZeshOne/Deze3.0`, `docsRootExists: true`.
  With the marker removed it failed with the new message naming both fixes.
- `go test ./...` 716 passed / 8 packages; `gofmt -l .` clean; `go vet ./...` clean.

## Open question deferred by the user

The compacted context is per node: a system and each of its modules have their
own, and `status --node Deze3.0/personas` returns module-scoped outputs while
`status --node Deze3.0` returns `absent`. But `docagent.status/v1` exposes no
`children` key, so an agent cannot enumerate a system's features through the
program. The index renders a child-module table as markdown in its managed
region; it is not in the JSON. For now the skill must ask the human rather than
guess a node. Adding `children` to status was NOT authorized and is not in scope.

**T2 done** — commit `e02b418`. The skill holds no filesystem path; it asks
`status` for `sddContext.outputs` and reads exactly those. Node scoping is
stated explicitly, with two stops: ask the human when the node is unknown, and
report an unreachable binary rather than working around it. The strict rule is
unchanged in meaning.

Evidence: the writer's RED was behavioral, not a compile error — 4 tests failed
naming the missing references, 5 passed. Parent read the final skill in full and
re-ran everything: `go test ./...` 719 passed / 8 packages, `gofmt -l .` clean,
`go vet ./...` clean. The pinned-path test became an absence test, which is the
stronger invariant.

Scope added to T3 by the parent: `registryTemplate` in
`internal/install/platform.go` carries a SECOND copy of doc-reader's rules —
line 786 ("in in-project mode" trigger row) and the Compact Rules block at 852
("Installed ONLY in in-project docs mode", plus the hardcoded paths). It now
contradicts the rewritten SKILL.md. Same class as issue #91.

**T3 done** — commit `8ccf2a1`. Gate, sweep and the whole `conditionalSkills`
mechanism removed (doc-reader was its only consumer, ever). The registry's
second copy of the rules now matches the skill, guarded by an absence test
scoped to the doc-reader block. Net -253 lines.

Evidence: RED was the two inverted vault-install tests plus the registry guard,
failing for the right reasons while the in-project test correctly stayed green.
`go test ./...` 713 passed / 8 packages (719 - 6 removed tests, exactly
accounted for), `gofmt -l .` clean, `go vet ./...` clean.

Parent correction applied on top: the writer removed `conditionalSkills` for
having no users, then kept an unused `platforms` parameter on the mode-switch
hook "for any future platform-scoped side effect", leaving the name
`runModeSwitchHookWithPlatforms` describing something it no longer did. One
call site, so it was renamed to `runModeSwitchHook` and the parameter dropped.
Keeping dead weight for a hypothetical is the opposite of the judgment applied
three files earlier in the same change.

## Delivery

Branch is at **1019 authored changed lines** against `main` (650 added, 369
deleted), well past the ~400 budget. Strategy is `ask-on-risk`, so this is the
point where the chain decision is due.

T1-T3 form one coherent change — doc-reader works regardless of documentation
origin — and T4 is a separate concern, repairing pre-v5 artifacts. Natural slice
boundary is right here.

## Next step

Chain decision, then T4.
