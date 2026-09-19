# Documentation for v5.1.0

## Objective

Bring the project's own documentation up to what it now is, before cutting
v5.1.0. The user framed this release as a foundation point: the docs have to be
good enough that someone arriving cold can use every command without reading Go.

## Problem

`main` carries six merged PRs since `v5.0.0` and none of them is described
anywhere a user reads:

| commit | what it changed |
|---|---|
| `46bf46a` (#93) | `doc-reader` asks the program for paths; works in both modes; installed everywhere; `.doc-agent.json` gained `node`; `status --node` optional |
| `0752764` (#94) | `doc-to-sdd` stops instead of hand-writing a context it could not submit |
| `1bd7038` (#96) | `doctor` adopts a pre-v5 compacted context; new `sddContext.state: "adopted"` |
| `4456308` (#104) | the English-only CI gate never ran; repaired, moved into a Go test, content fixed |
| `bdaf572` (#101) | six skills stopped instructing the write the program performs |
| `d433afd` (#102) | `status` exposes `children` so an agent can discover a system's modules |

Beyond the changelog gap, there is no usage guide at all. `docs/` holds one
brand asset. The README's `## Commands` section lists the pipeline subcommands
but does not document them as a reference: no full flag set, no worked example,
no sample output, and the eleven slash commands are not covered at the same
depth as the binary's ten subcommands.

## Scope

Authorized: CHANGELOG entry for v5.1.0, README updates, and a new usage guide
covering every command with examples. Then cut the tag.

Out of scope:
- Any behavior change. This is documentation only; if the work reveals a code
  defect, record it and keep going rather than fixing it here.
- The three deferred issues (#97, #99, #100).

## Constraints

- All documentation in English, per the project's own rule.
- Examples must be REAL: taken from actual command runs, not composed. A worked
  example that was never executed is the same class of claim this project spent
  v5 eliminating.
- The English-only content gate now actually runs. Anything added under `src/`
  or `skills/` is checked; a doc under `docs/` is not, but keep it English anyway.
- Version is `v5.1.0`: everything since v5.0.0 is backward compatible — the
  marker gained an optional key, `children` and the `adopted` state are additive
  JSON, and no skill changed its invocation contract.

## Resolved configuration

- TDD: strict, enabled. Documentation has no unit tests, so the applicable check
  is that every documented command is executed and its real output captured.
- Checks: `go test ./...`, `gofmt -l .`, `go vet ./...` must stay green.
- Delivery: `ask-on-risk`. Docs are additive; expect one PR.

## Tasks

- [x] **D1 — Inventory the real command surface.** Every binary subcommand with
  its exact flags, exit codes and output schema; every slash command with what
  it does and what it produces. Read the source, do not infer from the README —
  the README is one of the things being corrected. Route: delegated mapper.

- [ ] **D2 — Write the usage guide.** A new document covering all commands with
  worked examples captured from real runs. Route: delegated writer.

- [ ] **D3 — Update the README.** Reconcile it with what the tool now does, and
  point at the guide rather than duplicating it. Route: delegated writer.

- [ ] **D4 — CHANGELOG entry for v5.1.0.** Covering the six PRs above, in the
  file's established voice. Route: delegated writer.

- [ ] **D5 — Cut v5.1.0** per `RELEASING.md`, after the user approves the docs.

## Acceptance criteria

1. Every binary subcommand and every slash command is documented with at least
   one example whose output was actually captured from a run.
2. The CHANGELOG describes all six merged PRs.
3. The README contains nothing contradicted by the code.
4. All three checks stay green.
5. The user reviews the docs before the tag.

## Progress

Branch `docs/v5.1.0-usage-guide` cut from `main` at `d433afd`.

**D1 done.** Inventory complete, sourced from code and verified by real runs of a
freshly built binary in a scratch directory. It found **nine disagreements between
the README and the code**, each with file:line on both sides. The two that matter
most, re-verified by the parent:

1. `README.md:76` and `:228` still describe `doc-reader` as conditional on
   in-project mode and claim "switching back to vault mode automatically removes
   it". PR #93 made it unconditional and deleted that sweep. Two places in the
   README instruct users about behaviour that no longer exists.

2. `validate` accepted does NOT promise `commit-phase` will write. Proven at the
   signature level: `Validate(sub Submission, bank QuestionBank)` never receives an
   `Environment`, while `Commit(sub Submission, env Environment, bank QuestionBank)`
   does and calls `Resolve`. So validate cannot know whether the destination even
   resolves. Captured live: the same submission returns `"accepted"` exit 0 from
   validate, then `"undetermined"` exit 2 from commit-phase with "vault mode needs a
   base path but none is configured". The guide must say this plainly — it already
   cost a real run once.

Other findings: `status --node` optionality undocumented; the platforms table is
wrong about skills (universal, not Pi-specific) and about Pi's skill registry (it
does write one); `skills/doc-arch/SKILL.md:71-81` omits the `adopted (coverage
unverified)` node status; and `--help`'s exit-code line is inaccurate for
`sdd-commit`, whose environment failures surface as verdict 2 rather than usage 1.

## Next step

D2, D3 and D4 in parallel — they touch disjoint files.
