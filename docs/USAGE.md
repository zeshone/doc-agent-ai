# doc-agent-ai — Usage Guide

Every command this binary and its installed skills expose, with the exact
flags, the exit codes, whether it touches disk, and worked examples. Every
fenced `console`/`json` example in this guide — every command line, every
JSON body, every error line — was captured verbatim from a binary built from
this repository and run in a scratch directory outside it; none of it was
composed from the struct definitions. Reference tables (flags, arguments)
are read from the source, not printed by any command, and are cited against
the file they came from. Where a command could not reasonably be run end to
end (the interactive Bubbletea TUI, an actual GitHub Copilot install), that
is said explicitly instead of a fabricated transcript.

This is a reference, not a tutorial. Use the table of contents to jump to one
command; read [Four things worth knowing first](#four-things-worth-knowing-first)
once, because each entry there has already cost a real run someone made.

## Contents

- [Orientation](#orientation)
- [Four things worth knowing first](#four-things-worth-knowing-first)
- [Binary subcommands](#binary-subcommands)
  - [generate](#generate)
  - [install](#install)
  - [uninstall](#uninstall)
  - [topics](#topics)
  - [status](#status)
  - [validate](#validate)
  - [commit-phase](#commit-phase)
  - [decide-phase](#decide-phase)
  - [sdd-commit](#sdd-commit)
  - [doctor](#doctor)
  - [--version](#--version)
  - [--help](#--help)
- [Slash commands](#slash-commands)
  - [/doc-arch](#doc-arch)
  - [/doc-idea](#doc-idea)
  - [/doc-rec](#doc-rec)
  - [/doc-prd](#doc-prd)
  - [/doc-refine](#doc-refine)
  - [/doc-tech](#doc-tech)
  - [/doc-ddd](#doc-ddd)
  - [/doc-pti](#doc-pti)
  - [/doc-mod](#doc-mod)
  - [/doc-to-sdd](#doc-to-sdd)
  - [/doc-feat](#doc-feat)

---

## Orientation

`doc-agent-ai` has two layers:

- **Binary subcommands** — what you type at a shell, or what the installed
  agent runs on your behalf. `generate`, `install` and `uninstall` manage the
  tool itself. The other seven — `topics`, `status`, `validate`,
  `commit-phase`, `decide-phase`, `sdd-commit`, `doctor` — are the pipeline:
  they print versioned JSON on stdout and hold the authority over which phase
  a node is in. You will rarely type these yourself; the installed agent
  calls them so its claims about progress are computed, not remembered.
- **Slash commands** — what you type inside an installed AI coding platform
  (Claude Code, opencode, GitHub Copilot, Qwen Code, Pi). Each one delegates
  to a skill, and the canonical seven (`idea`, `rec`, `prd`, `refine`, `tech`,
  `ddd`, `pti`) drive the binary's pipeline commands underneath, gated on
  `status`.

**Exit codes**, for every pipeline subcommand:

| Code | Meaning |
|---|---|
| `0` | The command ran and its verdict is affirmative. |
| `1` | The invocation or the environment was wrong (bad flags, unresolvable destination for `status`, missing required arguments). |
| `2` | The command ran and refused: rejected or undetermined. This is a successful run with a negative verdict, not a broken invocation. |

One exception worth internalizing rather than trusting the general rule: see
[sdd-commit](#sdd-commit) below — its environment failures surface as exit
`2`, not `1`, because an unresolvable destination is folded into the same
`undetermined` result as a refused verdict.

**Where documentation lands** (vault vs. in-project mode, the `.doc-agent.json`
marker, the global `~/.doc-agent-ai.json` config) is covered in the
[README's Headless install section](../README.md#headless-install-ci--scripts).
This guide assumes you already know which mode you're in and focuses on what
each command does once that's resolved.

---

## Four things worth knowing first

### 1. `validate` accepting a submission does not mean `commit-phase` will write it

Look at the two function signatures this is built on
(`internal/pipeline/validate.go`, `internal/pipeline/commit.go`):

```go
func Validate(sub Submission, bank QuestionBank) ValidationResult
func Commit(sub Submission, env Environment, bank QuestionBank) CommitResult
```

`Validate` never receives an `Environment`. It checks the submission's shape —
answer coverage, provenance, section content — entirely in memory. `Commit`
does receive one, and the first thing it does is call `Resolve(sub.Node, env)`
to find out where the artifact would actually be written. `Validate` has no
way to know whether that resolves, so an `"accepted"` validation is not a
promise that `commit-phase` will write anything.

Here is the same submission run through both, in an environment with no vault
configured (no `~/.doc-agent-ai.json`, no `.doc-agent.json` marker — the
default falls back to vault mode with an empty base path):

```console
$ doc-agent-ai validate --node acme-hr --phase idea --answers answers.json --sections sections.json
```

```json
{
  "schemaName": "docagent.validation/v1",
  "node": "acme-hr",
  "phase": "idea",
  "result": "accepted",
  "checks": [
    { "id": "answer-record-present", "result": "pass" },
    { "id": "required-topics-covered", "result": "pass", "detail": "5 answered, 0 explicitly deferred, of 5 required" },
    { "id": "verbatim-provenance", "result": "pass", "detail": "5 of 5 recorded answers carry a source; 0 unattributed" },
    { "id": "sections-known", "result": "pass" },
    { "id": "section-content-present", "result": "pass", "detail": "every required topic for a system node carries prose or an explicit TBD" }
  ]
}
```

`validate` exits `0`. The exact same `--answers` and `--sections` files, same
node and phase, through `commit-phase`, in that same unconfigured environment:

```console
$ doc-agent-ai commit-phase --node acme-hr --phase idea --answers answers.json --sections sections.json
```

```json
{
  "schemaName": "docagent.commit/v1",
  "result": "undetermined",
  "node": "acme-hr",
  "phase": "idea",
  "written": [],
  "detail": "vault mode needs a base path but none is configured: run `doc-agent-ai install --docs-mode vault --path <path>` or set a project marker"
}
```

`commit-phase` exits `2`. Nothing was written — `written` is empty — but the
submission was never rejected on its own merits either: it is
`"undetermined"`, a distinct result from `"rejected"`, because the destination
itself could not be resolved. `validate` cannot catch this class of failure by
construction, because it is never told where the write would land. If your
own tooling treats "validate passed" as "commit-phase will succeed," this is
the case that breaks that assumption — it has already cost a real
documentation run once.

### 2. `doctor` reports by default; it only writes with `--apply`

```console
$ doc-agent-ai doctor --node legacy-sys --check
```

```json
{
  "schemaName": "docagent.doctor/v1",
  "node": "legacy-sys",
  "mode": "vault",
  "docsRoot": "/…/vault/legacy-sys",
  "applied": false,
  "findings": [
    { "kind": "adopt-phase", "node": "legacy-sys", "phase": "rec", "detail": "legacy-sys_requirements.md exists with no answer record: adopt as unverified" },
    { "kind": "create-index", "node": "legacy-sys", "detail": "/…/vault/legacy-sys/legacy-sys.md does not exist and will be created" }
  ],
  "blocked": [
    "legacy-sys: the archetype could not be determined from /…/vault/legacy-sys/legacy-sys.md. Re-run with --archetype bounded or --archetype evolving"
  ]
}
```

`"applied": false` — a bare `--check` (or omitting both flags; `--check` is
the default) changes nothing on disk. Confirmed by listing the docs root
right after the run above: only the original hand-written
`legacy-sys_requirements.md` is there, no index, no adoption record.

The same node, `--apply`'d with the archetype it asked for supplied on the
command line:

```console
$ doc-agent-ai doctor --node legacy-sys --apply --archetype evolving
```

```json
{
  "schemaName": "docagent.doctor/v1",
  "node": "legacy-sys",
  "mode": "vault",
  "docsRoot": "/…/vault/legacy-sys",
  "applied": true,
  "findings": [
    { "kind": "adopt-phase", "node": "legacy-sys", "phase": "rec", "detail": "legacy-sys_requirements.md exists with no answer record: adopt as unverified" },
    { "kind": "archetype", "node": "legacy-sys", "detail": "archetype \"evolving\", supplied with --archetype" },
    { "kind": "create-index", "node": "legacy-sys", "detail": "/…/vault/legacy-sys/legacy-sys.md does not exist and will be created" }
  ],
  "blocked": []
}
```

Now `legacy-sys.md` exists with a `[~] adopted` row for `rec`, and
`.doc-agent-state/adoption.json` is written. `--apply` and `--check` are
mutually exclusive (`--apply --check` together is a usage error, exit `1`,
see [doctor](#doctor)); reporting is the default posture, never the other way
around, because this command touches documentation a human wrote by hand.

### 3. `status --node` is the one optional `--node`, and it is optional in exactly one way

Every other node-taking command (`validate`, `commit-phase`, `decide-phase`,
`sdd-commit`, `doctor`) requires `--node` explicitly — there is no fallback.
`status` alone falls back to the `"node"` key in the project's
`.doc-agent.json` marker when `--node` is omitted, and an explicit `--node`
always wins over whatever the marker says.

With a marker present (`{"mode": "in-project", "node": "acme-hr"}`) and no
`--node` flag:

```console
$ doc-agent-ai status
```

resolves and prints `acme-hr`'s full status (`"target": {"node": "acme-hr", ...}`,
`"modeResolvedBy": "marker"`), exit `0`.

With neither `--node` nor a marker:

```console
$ doc-agent-ai status
```

```
Error: status needs --node <system[/module[/submodule]]>, or a "node" recorded in .doc-agent.json
```

Exit `1`. The error names both fixes — pass `--node`, or write a marker — in
one line, rather than only telling you what's missing.

### 4. The legacy `doc-feat` mini-flow sits outside the v5 pipeline

`/doc-feat`, and the skills it drives — `doc-scope`, `doc-rec-lite`,
`doc-prd-lite` — never invoke the binary. No answer records are written, no
topic coverage is counted, none of `status`, `validate`, `commit-phase` or
`doctor` ever sees these artifacts, and `doctor` cannot adopt them later
because they carry none of the shapes it recognizes. Output lands in a
separate namespace, `<system>-features/<slug>/`, a sibling of the real node
tree rather than a node inside it — not something `status --node
<system>/<slug>` could ever resolve to, because the pipeline's own path
resolution (`internal/pipeline/resolve.go`) does not produce that shape for
any node string.

Concretely: grep for `doc-agent-ai`, `commit-phase`, `--node` or `status
--node` inside `skills/doc-feat/`, `skills/doc-scope/`, `skills/doc-rec-lite/`
or `skills/doc-prd-lite/` and you will find nothing. Compare that to
`skills/doc-arch/SKILL.md`, which is the only skill that references the
pipeline commands, and is the shared rules skill for every canonical phase
role (`doc-idea`, `doc-rec`, `doc-prd`, `doc-refinement`, `doc-tech`,
`doc-pti`, `doc-ddd`, `doc-to-sdd` — see `src/manifests/content.json`'s
`rulesSkill` field). `doc-feat` is not on that list, and its own `SKILL.md`
defines a fully self-contained mini-flow with its own output layout.

A reader must not assume a feature documented through `/doc-feat` carries the
guarantees the seven canonical phases do — no coverage was counted, so none
can be claimed. What this mini-flow should become, if anything, is tracked in
[issue #97](https://github.com/zeshone/doc-agent-ai/issues/97); until that is
resolved, treat its output as informal.

---

## Binary subcommands

### generate

Writes the rendered bundle (skills, prompts, agents, commands, manifest — the
same content `install` copies into a platform's home directory) to an
explicit directory. Dev/build tooling, not something an end user runs day to
day.

**Flags:** positional only.

| Argument | Required | Meaning |
|---|---|---|
| `<dir>` | Yes | Destination directory. Created if it does not exist. |

**Writes to disk:** yes, unconditionally, to `<dir>`.

**Exit codes:** `0` on success. `1` if `<dir>` is missing, or if generation
itself fails (an error from the embedded content).

```console
$ doc-agent-ai generate ./bundle-out
bundle generated at ./bundle-out
$ find bundle-out -maxdepth 1
bundle-out
bundle-out/agents-claude
bundle-out/agents-copilot
bundle-out/agents-qwen
bundle-out/commands
bundle-out/manifest.json
bundle-out/prompts
bundle-out/prompts-claude
bundle-out/prompts-copilot
bundle-out/prompts-pi
bundle-out/prompts-qwen
bundle-out/skills
```

Missing the argument:

```console
$ doc-agent-ai generate
Usage: doc-agent-ai generate <dir>
```

Exit `1`.

### install

Installs the tool's skills, prompts, agents and commands to every detected
(or explicitly requested) platform. With no install flags and a TTY present,
it launches the Bubbletea wizard (not capturable here — this guide only
documents the headless path, which is fully scriptable and was run for real
below). With no flags and no TTY, it prints guidance and exits `1` rather
than guessing.

**Decision order:** explicit install flags present → headless install.
Otherwise: TTY present → TUI wizard. Otherwise: error.

**Install flags** (any one of these present triggers headless mode):

| Flag | Required | Meaning |
|---|---|---|
| `--platforms <csv>` | No | Comma-separated platform IDs: `opencode`, `claude`, `copilot`, `qwen`, `pi`. Omit for every detected platform. Unknown IDs are a usage error. |
| `--docs-mode <mode>` | No | `vault` (default) or `in-project`. |
| `--path <path>` | Conditionally | Vault base path. Required when the effective mode is `vault` and no path is already saved in `~/.doc-agent-ai.json`. |
| `--yes` | No | Skip interactive overwrite confirmation. Alone, it still needs a resolvable mode/path from a prior install's saved config. |

**Platform-path overrides**, consumed before the subcommand switch and usable
with any subcommand, but relevant to `install`/`uninstall`:

| Flag | Meaning |
|---|---|
| `--copilot-path <path>` | Overrides the GitHub Copilot home directory, bypassing auto-detection. |
| `--pi-path <path>` | Overrides the Pi agent home directory, bypassing auto-detection and `PI_CODING_AGENT_DIR`. |

**Writes to disk:** yes — skills, prompts, agent files and (opencode/Claude/Pi
only) `.atl/skill-registry.md` under each target platform's home directory,
plus `~/.doc-agent-ai.json` recording the chosen mode/path/platforms for next
time.

**Exit codes:** `0` on success. `1` on a bad flag combination, an unresolvable
mode/path, or an already-installed platform without `--yes`/TUI consent.

Real headless run — `HOME` pointed at a scratch directory with fake `.claude`
and `.qwen` markers so nothing outside the scratch tree was touched:

```console
$ doc-agent-ai install --platforms claude,qwen --docs-mode vault --path /scratch/vault/ --yes

  Installing for Claude Code...
  ✔ skill: doc-arch
  ✔ skill: doc-idea
  … (12 more skills)
  ✔ skill: doc-reader
  ✔ prompt: doc-arch.md
  … (8 more prompts)
  ✔ agent: doc-arch.md
  … (8 more agents)
  ✔ skill-registry.md written

  Installing for Qwen Code...
  ✔ skill: doc-arch
  … (13 more skills, doc-reader included)
  ✔ prompt: doc-arch.md
  … (8 more prompts)
$ echo $?
0
```

Both platforms received `doc-reader` — confirmed by listing each platform's
`skills/` directory afterward — and only Claude got a
`skill-registry.md written` line; Qwen's `WriteSkillRegistry` is a no-op (see
the corrected platform table in the [README](../README.md#platforms)).
`~/.doc-agent-ai.json` afterward:

```json
{
  "version": 1,
  "mode": "vault",
  "path": "/scratch/vault/",
  "platforms": ["claude", "qwen"]
}
```

A flag combination that cannot resolve a path:

```console
$ doc-agent-ai install --yes
Error: invalid flags: vault mode requires a documentation base path; provide --path or run without --docs-mode to use the TUI
```

Exit `1`.

### uninstall

Removes every doc-agent-ai artifact (skills, prompts, agents, commands,
`.atl/skill-registry.md`) from detected platforms. Your documentation files
are never touched — this only ever looks under each platform's home
directory. With a TTY, it launches the Bubbletea TUI. Without one, it falls
back to a `bufio`-based yes/no prompt on stdin — asymmetric with `install`,
which errors rather than falling back, because uninstall only needs a
confirmation, not undecidable input like a mode or a path.

**Flags:** none of its own; `--copilot-path`/`--pi-path` overrides apply here
too.

**Writes to disk:** deletes files (never documentation) after confirmation.

**Exit codes:** `0` whether anything was removed, declined, or there was
nothing to do. Uninstall never signals failure through the exit code for a
normal "nothing installed" or "user declined" outcome.

Real run, non-TTY fallback, confirmed with `y` on stdin, against the same
scratch install from above:

```console
$ echo y | doc-agent-ai uninstall

  doc-agent-ai vdev — uninstaller

  Detecting platforms...
  ⚠  opencode not found  (opencode.json missing)
  ✔ Qwen Code detected  (/scratch/home/.qwen)
  ⚠  GitHub Copilot not found  (~/.copilot missing or 'code' not in PATH)
  ✔ Claude Code detected  (/scratch/home/.claude)

  The following will be removed:
  ─────────────────────────────────
  qwen:
    → Skills: doc-arch, doc-idea, … , doc-reader
    → Prompts: prompts/doc/
    → Agents: doc-arch, doc-ddd, … , doc-to-sdd
  claude:
    → Skills: doc-arch, doc-idea, … , doc-reader
    → Prompts: prompts/doc/
    → Agents: doc-arch, doc-ddd, … , doc-to-sdd
    → Registry: .atl/skill-registry.md

  ⚠  Your documentation files are NOT affected.

  Uninstall from all detected platforms? (y/N)
  Removing from qwen...
  ✔ removed: skill: doc-arch
  … (every skill, prompt and agent, individually)

  Removing from claude...
  ✔ removed: skill: doc-arch
  … (every skill, prompt and agent, individually)
  ✔ removed: .atl/skill-registry.md

  ✔ Uninstall complete.
  Restart your AI tool if it is currently running.
$ echo $?
0
```

Afterward, `<home>/.claude/skills/` no longer exists.

### topics

Prints the question bank — every phase's required topics, in the machine
vocabulary the model routes on. This is how an agent learns what a phase
needs to ask about without that list living in skill prose. With no flags,
prints the whole bank; narrow with `--phase`, further with `--node-type`.

**Flags:**

| Flag | Required | Meaning |
|---|---|---|
| `--phase <id>` | No | One of `idea`, `rec`, `prd`, `refine`, `tech`, `ddd`, `pti`. Omit for the entire bank. |
| `--node-type <t>` | No | `system`, `module` or `submodule`. Only meaningful with `--phase`; narrows topics whose bank entry declares `appliesWhen`. Omitting it returns every declared topic for the phase regardless of node type. |

**Writes to disk:** no.

**Exit codes:** `0` on success. `1` for an unknown phase or node type.

Full bank (truncated here; the real output is the whole `docagent.questionbank/v1` document, one entry per phase):

```console
$ doc-agent-ai topics
{
  "schemaName": "docagent.questionbank/v1",
  "phases": [
    {
      "phase": "idea",
      "kind": "interview",
      "artifact": "{node}_idea-brief.md",
      "legacyArtifacts": ["{node}_idea.md"],
      "documentTitle": "Idea Brief",
      "artifactOptional": true,
      "requiredTopics": [
        { "id": "target-users", "title": "Target Users" },
        { "id": "problem-solved", "title": "Problem Solved", "note": "…" },
        { "id": "success-definition", "title": "Definition of Success", "note": "…" },
        { "id": "out-of-scope", "title": "Out of Scope" },
        { "id": "why-now", "title": "Why Now" }
      ]
    },
    … 6 more phases …
  ]
}
```

`--node-type` actually filtering a conditional topic — `tech`'s
`inheritance-mode` topic only applies to `module`/`submodule` nodes
(`appliesWhen: {nodeType: [module, submodule]}` in `questionbank.yaml`).
For a `system` node it is absent from the result:

```console
$ doc-agent-ai topics --phase tech --node-type system
```

```json
{
  "schemaName": "docagent.questionbank/v1",
  "phase": "tech",
  "kind": "interview",
  "optional": false,
  "artifact": "{node}_tech-spec.md",
  "documentTitle": "Technical Specification",
  "requiredTopics": [
    { "id": "architecture", "title": "Architecture" },
    { "id": "component-contracts", "title": "Component Contracts" },
    { "id": "tradeoffs", "title": "Tradeoffs" },
    { "id": "rollout", "title": "Rollout" },
    { "id": "validation-strategy", "title": "Validation Strategy" }
  ]
}
```

For `--node-type module`, the same call includes a sixth topic,
`inheritance-mode` / "Architecture Inheritance", with its `appliesWhen` echoed
back in the JSON.

### status

Prints a node's computed position: which phases are complete, adopted,
pending or blocked; topic coverage counts; the compacted-context freshness;
discovered child nodes; and the single next action to take. This is the
command an agent calls to decide what to do next — never inferred from
conversation, files it believes exist, or a checkbox.

**Flags:**

| Flag | Required | Meaning |
|---|---|---|
| `--node <n>` | **No — the only node-taking command where this is true.** | `system`, `system/module` or `system/module/submodule`. Falls back to the `"node"` key in `.doc-agent.json` when omitted. An explicit flag always wins over the marker. |

**Writes to disk:** no.

**Exit codes:** `0` for any computed status, including a blocked one — a
blocked status is a correct answer, not a failed invocation. `1` when neither
`--node` nor a usable marker is available. `2` only when the status itself
comes back `"cannot-determine"` (an unresolvable destination, a corrupt
adoption/decisions record, or a question-bank/binary mismatch).

Real run, vault mode with no base path configured at all (default mode,
`Resolve` fails):

```console
$ doc-agent-ai status --node acme-hr
```

```json
{
  "schemaName": "docagent.status/v1",
  "target": { "node": "acme-hr", "nodeType": "system", "shortName": "acme-hr", "docsRootExists": false },
  "phases": [ /* all 7 phases, "state": "undetermined" */ ],
  "completed": [],
  "adopted": [],
  "missing": ["idea", "rec", "prd", "refine", "tech", "ddd", "pti"],
  "blockedReasons": [
    { "code": "docs-root-unresolved", "detail": "vault mode needs a base path but none is configured: run `doc-agent-ai install --docs-mode vault --path <path>` or set a project marker" }
  ],
  "nextAction": {
    "kind": "cannot-determine",
    "command": null,
    "reason": "vault mode needs a base path but none is configured: run `doc-agent-ai install --docs-mode vault --path <path>` or set a project marker",
    "operatorHint": "resolve the documentation destination, then run status again"
  }
}
```

Exit `2`. Missing both `--node` and a marker:

```console
$ doc-agent-ai status
Error: status needs --node <system[/module[/submodule]]>, or a "node" recorded in .doc-agent.json
```

Exit `1`. One real, single `status --node acme-hr` call, later in the same
vault, after the `idea` phase was committed ([commit-phase](#commit-phase)),
`ddd` was declined ([decide-phase](#decide-phase)), an `sdd-commit` had run
and then a source drifted ([sdd-commit](#sdd-commit)), and a module directory
had been created underneath it — everything below came out of that one
invocation, not assembled from several:

```json
{
  "schemaName": "docagent.status/v1",
  "target": {
    "node": "acme-hr", "nodeType": "system", "shortName": "acme-hr",
    "mode": "vault", "modeResolvedBy": "global",
    "docsRoot": "/…/vault/acme-hr", "docsRootExists": true
  },
  "phases": [
    { "id": "idea", "state": "complete", "artifact": "acme-hr_idea-brief.md", "artifactExists": true, "requiredTopics": 5, "answeredTopics": 5, "deferredTopics": [], "unansweredTopics": [] },
    { "id": "rec", "state": "pending", "artifact": "acme-hr_requirements.md", "artifactExists": false, "requiredTopics": 7, "answeredTopics": 0, "deferredTopics": [], "unansweredTopics": ["archetype", "stakeholders", "business-events", "current-process", "business-rules", "exceptions-edge-cases", "stakeholder-conflicts"] },
    { "id": "prd", "state": "blocked", "artifact": "acme-hr_prd.md", "artifactExists": false, "requiredTopics": 9, "answeredTopics": 0, "deferredTopics": [], "unansweredTopics": ["primary-user-flows", "user-stories", "acceptance-criteria", "dependencies-integrations", "technical-constraints", "security-privacy", "risks-roadmap", "open-decisions", "success-metrics"] },
    { "id": "refine", "state": "blocked", "artifact": "acme-hr_prd.md", "artifactExists": false, "requiredTopics": 0, "answeredTopics": 0, "deferredTopics": [], "unansweredTopics": [] },
    { "id": "tech", "state": "blocked", "artifact": "acme-hr_tech-spec.md", "artifactExists": false, "requiredTopics": 5, "answeredTopics": 0, "deferredTopics": [], "unansweredTopics": ["architecture", "component-contracts", "tradeoffs", "rollout", "validation-strategy"] },
    { "id": "ddd", "state": "not-applicable", "artifact": "acme-hr_db-design.md", "artifactExists": false, "requiredTopics": 0, "answeredTopics": 0, "deferredTopics": [], "unansweredTopics": [] },
    { "id": "pti", "state": "blocked", "artifact": "acme-hr_issues.md", "artifactExists": false, "requiredTopics": 5, "answeredTopics": 0, "deferredTopics": [], "unansweredTopics": ["slice-boundaries", "verifiable-criteria", "execution-type", "issue-dependencies", "open-tbds"] }
  ],
  "completed": ["idea"],
  "adopted": [],
  "missing": ["rec", "prd", "refine", "tech", "pti"],
  "nextRecommended": "rec",
  "blockedReasons": [],
  "sddContext": { "state": "stale", "drifted": ["acme-hr_idea-brief.md"], "coverageVerified": true, "outputs": ["/…/vault/acme-hr/agent_sdd_context_project/acme-hr_sdd-context.md", "/…/vault/acme-hr/agent_sdd_context_project/acme-hr_sdd-tech-context.md"] },
  "children": [ { "node": "acme-hr/payroll", "shortName": "payroll" } ],
  "nextAction": { "kind": "start-phase", "phase": "rec", "command": "/doc-rec acme-hr", "reason": "phase \"rec\" has no recorded work yet" }
}
```

Every field a single node's status can carry is visible here at once: mixed
phase states (`complete`, `pending`, `blocked`, `not-applicable`), a
`children` entry, and a `stale` `sddContext`. See
[sdd-commit](#sdd-commit) for `sddContext`'s other two states,
`fresh` and `absent`, and [doctor](#doctor) for its fourth,
`adopted`, which only ever appears on a node `doctor` adopted, never
here.

### validate

Checks a phase submission's shape — without an `Environment`, so without ever
resolving where it would be written. See
[#1 above](#1-validate-accepting-a-submission-does-not-mean-commit-phase-will-write-it)
for why that distinction matters.

**Flags** (shared with `commit-phase`):

| Flag | Required | Meaning |
|---|---|---|
| `--node <n>` | Yes | Node identifier. |
| `--phase <p>` | Yes | One of the seven canonical phases. |
| `--answers <file>` | Conditionally | Path to a `docagent.answers/v1` file. Required for interview phases (everything except `refine`). |
| `--audit <file>` | Conditionally | Path to a `docagent.audit/v1` file. Required for the `refine` audit phase, used instead of `--answers`. |
| `--sections <file>` | Conditionally | Path to a `docagent.sections/v1` file. Required for interview phases; not used for audits, which write no artifact of their own. |

**Writes to disk:** never.

**Exit codes:** `0` when `result` is `"accepted"`. `2` when `"rejected"` or
`"undetermined"` (a malformed record, an unknown phase, coverage gaps). `1`
for a plain usage error (missing `--node`/`--phase`, or a phase missing its
required `--answers`/`--audit`/`--sections`).

See the accepted example under
[#1 above](#1-validate-accepting-a-submission-does-not-mean-commit-phase-will-write-it) —
same command, real output, not repeated here.

### commit-phase

Validates a submission and writes it only if it passes — the program holds
the pen. A rejected or undetermined submission touches no file.

**Flags:** identical to `validate`, above.

**Writes to disk:** yes, but only on a `"written"` result: the answer record
(or audit record), the section input, the rendered artifact (interview
phases) or the rendered report (audit phases), and the node's index. Nothing
on `"rejected"` or `"undetermined"`.

**Exit codes:** `0` on `"written"`. `2` on `"rejected"` or `"undetermined"`
(see [#1](#1-validate-accepting-a-submission-does-not-mean-commit-phase-will-write-it)
for the case where validate already said `"accepted"`). `1` for a plain usage
error.

Real successful write — vault mode configured, same `idea` submission as the
`validate` example above:

```console
$ doc-agent-ai commit-phase --node acme-hr --phase idea --answers answers.json --sections sections.json
```

```json
{
  "schemaName": "docagent.commit/v1",
  "result": "written",
  "node": "acme-hr",
  "phase": "idea",
  "written": [
    "/…/vault/acme-hr/.doc-agent-state/answers/acme-hr.idea.json",
    "/…/vault/acme-hr/.doc-agent-state/sections/acme-hr.idea.json",
    "/…/vault/acme-hr/acme-hr_idea-brief.md"
  ],
  "indexUpdated": { "file": "/…/vault/acme-hr/acme-hr.md", "phaseMarked": "idea", "nodeStatusRecomputed": "in progress" },
  "nextRecommended": "rec"
}
```

Exit `0`. The resulting index (`acme-hr.md`) carries the machine-owned region:

```
| Phase | Done | State | Coverage |
|---|---|---|---|
| idea | [x] | complete | 5/5 |
| rec | [ ] | pending | 0/7 |
…
**Node status:** in progress
**Next:** rec
```

And the rendered artifact (`acme-hr_idea-brief.md`) has canonical English
headings (`## Target Users`, `## Problem Solved`, …) rendered by the program,
with the submitted prose underneath each one.

An **audit phase** (`refine`) uses `--audit` instead of `--answers`, and
writes no interview artifact of its own — only the audit record and a
rendered report next to the PRD it judged:

```console
$ doc-agent-ai commit-phase --node audit-node --phase refine --audit audit.json
```

```json
{
  "schemaName": "docagent.commit/v1",
  "result": "written",
  "node": "audit-node",
  "phase": "refine",
  "written": [
    "/…/vault/audit-node/.doc-agent-state/audits/audit-node.refine.json",
    "/…/vault/audit-node/audit-node_refinement.md"
  ],
  "indexUpdated": { "file": "/…/vault/audit-node/audit-node.md", "phaseMarked": "refine", "nodeStatusRecomputed": "in progress" },
  "nextRecommended": "idea"
}
```

The rendered `audit-node_refinement.md` includes a computed summary table
(subjects audited, passing/failing counts) and a per-criterion verdict
matrix (✅/❌) — computed from the record, so the report and what the
pipeline counts cannot disagree.

For the undetermined/rejected side of this command, see
[#1 above](#1-validate-accepting-a-submission-does-not-mean-commit-phase-will-write-it).

### decide-phase

Records the user's accept/decline choice about an optional phase (currently
only `ddd`), so it is not re-asked in a later session, and returns the
node's recomputed status.

**Flags:**

| Flag | Required | Meaning |
|---|---|---|
| `--node <n>` | Yes | Node identifier. |
| `--phase <p>` | Yes | Must be a phase the bank marks `optional: true` (only `ddd` today). |
| `--decision <d>` | Yes | `accepted` or `declined`. |

**Writes to disk:** yes — `.doc-agent-state/decisions.json`.

**Exit codes:** `0` on success. `1` for missing flags, an unknown phase, or a
phase that is not optional.

```console
$ doc-agent-ai decide-phase --node acme-hr --phase ddd --decision declined
```

Prints the node's full recomputed `docagent.status/v1`, exit `0`. In it,
`ddd`'s phase entry now reads `"state": "not-applicable"`, and it stops
counting toward "applicable" phases anywhere status derives node-level
completion (see [doctor](#doctor)'s adopted-node-status example, which
relies on exactly this to reach `"adopted (coverage unverified)"`). Before
any `/doc-to-sdd` compaction has ever run for this node, the same response
also carries `sddContext`'s first state, real and unedited:

```json
"sddContext": { "state": "absent", "coverageVerified": false }
```

### sdd-commit

Verifies a compacted agent context (the two files `/doc-to-sdd` produces) and
writes it — plus a fingerprinting manifest — only if it holds. It cannot
judge whether the compaction is faithful to its sources; it verifies what is
mechanically checkable: sources are named and hashed, every open question
claimed as preserved is actually present in both a source and the output, no
claimed open question is invented, and a compaction is refused outright if it
records zero decisions.

**Flags:**

| Flag | Required | Meaning |
|---|---|---|
| `--node <n>` | Yes | Node identifier. |
| `--input <file>` | Yes | Path to a `docagent.sddinput/v1` file: `business` and `technical` markdown, `preservedTbds`, `decisions`. |

**Writes to disk:** yes, but only on a `"written"` result:
`agent_sdd_context_project/<prefix>_sdd-context.md`,
`..._sdd-tech-context.md`, and `manifest.json` beside them.

**Exit codes:** `0` on `"written"`. `2` on `"rejected"` (failed checks) or
`"undetermined"` (destination cannot be resolved — see the note below).

Rejected — no decisions recorded:

```console
$ doc-agent-ai sdd-commit --node acme-hr --input sdd-input-rejected.json
```

```json
{
  "schemaName": "docagent.sddmanifest/v1",
  "result": "rejected",
  "node": "acme-hr",
  "written": [],
  "checks": [
    { "id": "sdd-sources-present", "result": "pass", "detail": "1 of 6 source artifacts present" },
    { "id": "sdd-tbds-not-invented", "result": "pass" },
    { "id": "sdd-tbds-preserved", "result": "pass", "detail": "0 open question(s) verified present in both a source and the compaction" },
    { "id": "sdd-decisions-bounded", "result": "fail", "detail": "no decisions were recorded; an agent reading this would still have to open the source documents to learn what was settled and why" },
    { "id": "sdd-coverage-stated", "result": "pass", "detail": "every source carries counted coverage" }
  ],
  "rejectedBecause": ["sdd-decisions-bounded"]
}
```

Exit `2`. The same input with one decision added, accepted and written:

```json
{
  "schemaName": "docagent.sddmanifest/v1",
  "result": "written",
  "node": "acme-hr",
  "written": [
    "/…/vault/acme-hr/agent_sdd_context_project/acme-hr_sdd-context.md",
    "/…/vault/acme-hr/agent_sdd_context_project/acme-hr_sdd-tech-context.md",
    "/…/vault/acme-hr/agent_sdd_context_project/manifest.json"
  ],
  "checks": [
    { "id": "sdd-sources-present", "result": "pass", "detail": "1 of 6 source artifacts present" },
    { "id": "sdd-tbds-not-invented", "result": "pass" },
    { "id": "sdd-tbds-preserved", "result": "pass", "detail": "0 open question(s) verified present in both a source and the compaction" },
    { "id": "sdd-decisions-bounded", "result": "pass", "detail": "1 decision(s), each with what, why, so-that and how it was decided" },
    { "id": "sdd-coverage-stated", "result": "pass", "detail": "every source carries counted coverage" }
  ]
}
```

Exit `0`. `status --node acme-hr` right after reports `"sddContext": {"state": "fresh", "coverageVerified": true, "outputs": [...]}`.
Editing a source artifact afterward flips it: `status` then reports
`"state": "stale", "drifted": ["acme-hr_idea-brief.md"]`.

**The exit-code exception worth its own line:** run `sdd-commit` with a
completely valid node and input, but no vault base path configured —
an *environment* failure, not a content one:

```console
$ doc-agent-ai sdd-commit --node acme-hr --input sdd-input.json
```

```json
{
  "schemaName": "docagent.sddmanifest/v1",
  "result": "undetermined",
  "node": "acme-hr",
  "written": [],
  "checks": null,
  "detail": "vault mode needs a base path but none is configured: run `doc-agent-ai install --docs-mode vault --path <path>` or set a project marker"
}
```

This exits `2`, the same code as a refused verdict — not `1`, even though the
general "usage or environment error → 1" rule you'd expect from `--help`
would predict `1` here. `Environment` failures for `sdd-commit` are folded
into `CommitUndetermined`, which this codebase always maps to
`ExitVerdict` (`2`). Script against the JSON body (`"result"`), not against
the exit code alone, if you need to tell this apart from a genuine content
rejection.

### doctor

Aligns documentation written before answer records existed. It never
fabricates an answer record — the user's words from an interview months ago
exist nowhere, and generating them from the artifact would be the exact
fabricated-completeness failure this whole tool exists to prevent. It can
only mark a phase **adopted**: present and usable, coverage explicitly
unverified.

**Flags:**

| Flag | Required | Meaning |
|---|---|---|
| `--node <n>` | Yes | Node identifier. |
| `--check` | No | Report only. This is the default even when omitted. |
| `--apply` | No | Write the plan. Mutually exclusive with `--check`. |
| `--recursive` | No | Also adopt nodes nested under this one, deepest first. |
| `--archetype <bounded\|evolving>` | No | Supplies the archetype when it cannot be determined from the (not-yet-existing) index. Rejected if it's anything else. |

**Writes to disk:** only with `--apply`: the adoption record
(`.doc-agent-state/adoption.json`) and the index, if either changed.

**Exit codes:** `0` when nothing is blocked. `2` when `blocked` is non-empty
(commonly: the archetype could not be determined and `--archetype` was not
supplied). `1` for `--apply --check` together, an unknown `--archetype`
value, or a missing `--node`.

See [#2 above](#2-doctor-reports-by-default-it-only-writes-with---apply) for
the report-vs-apply pair of real runs.

A fully adopted node reaches the fourth node-status value,
`"adopted (coverage unverified)"` — every applicable phase adopted (with
`ddd` declined via `decide-phase` first, so it doesn't count as
"applicable"):

```console
$ doc-agent-ai doctor --node legacy-sys --apply
… findings for idea, rec, prd, refine, tech, pti, each "kind": "adopt-phase" …
$ grep 'Node status' vault/legacy-sys/legacy-sys.md
**Node status:** adopted (coverage unverified)
```

`doctor` can also adopt a compacted `/doc-to-sdd` context that predates the
manifest — a `agent_sdd_context_project/` directory with output files on disk
but no `manifest.json` beside them, the pre-v5 shape. This is where
`sddContext`'s fourth state, `adopted`, comes from — the only one of the four
`status` cannot reach on its own, because it depends on a `doctor` finding,
not on anything `sdd-commit` writes:

```console
$ doc-agent-ai doctor --node legacy2 --apply --archetype bounded
```

```json
{
  "schemaName": "docagent.doctor/v1",
  "node": "legacy2",
  "mode": "vault",
  "docsRoot": "/…/vault/legacy2",
  "applied": true,
  "findings": [
    { "kind": "adopt-sdd-context", "node": "legacy2", "detail": "legacy2_sdd-context.md, legacy2_sdd-tech-context.md present with no manifest: adopt as unverified" },
    { "kind": "archetype", "node": "legacy2", "detail": "archetype \"bounded\", supplied with --archetype" },
    { "kind": "create-index", "node": "legacy2", "detail": "/…/vault/legacy2/legacy2.md does not exist and will be created" }
  ],
  "blocked": []
}
```

```console
$ doc-agent-ai status --node legacy2
```

```json
"sddContext": {
  "state": "adopted",
  "coverageVerified": false,
  "outputs": [
    "/…/vault/legacy2/agent_sdd_context_project/legacy2_sdd-context.md",
    "/…/vault/legacy2/agent_sdd_context_project/legacy2_sdd-tech-context.md"
  ]
}
```

`--apply --check` together:

```console
$ doc-agent-ai doctor --node x --apply --check
Error: --apply and --check contradict each other; pick one
```

Exit `1`.

### --version

Prints the build version and exits `0`.

```console
$ doc-agent-ai --version
doc-agent-ai dev
```

`dev` is what an unflagged `go build` produces (`internal/build.Version`
defaults to the string `"dev"`); release builds set it via `-ldflags
-X`, so a distributed binary prints a real semver like `doc-agent-ai v5.1.0`.

### --help

Prints the full command summary shown in the [Orientation](#orientation)
section above and exits `0`. Its own exit-codes line ("0 affirmative, 1
usage or environment error, 2 refused verdict") is the general rule; see
[sdd-commit](#sdd-commit) for the one documented place it does not hold
literally.

An unknown subcommand also prints the same help text, to stderr's sibling
stdout mixed with an error line, and exits `1`:

```console
$ doc-agent-ai frobnicate
Unknown subcommand: frobnicate
doc-agent-ai — Multi-platform documentation workflow agent installer
…
```

---

## Slash commands

These run inside an installed AI platform. Each delegates to a skill; the
seven canonical ones share `doc-arch`'s rules skill and are gated by
`status`/`commit-phase` underneath (see
[#4](#4-the-legacy-doc-feat-mini-flow-sits-outside-the-v5-pipeline) for the
one command family that is not). "Phase" below is the canonical phase id
from `questionbank.yaml`, where one applies.

### /doc-arch

**Does:** runs the complete documentation workflow for a system or module:
idea → rec → prd → refine → tech → [ddd] → pti, pausing between each step
for confirmation, asking about `ddd` after `tech`, updating index checkboxes
as phases complete.

**Argument:** `<system>` (or, per its skill's Existing-Project Detection
step, offers to route into `/doc-mod` or `/doc-feat` instead if the system
already exists).

**Produces:** the master index plus every canonical phase's artifact for the
system.

**Phase:** none of its own — it is the orchestrator that drives all seven.

**Standalone or flow:** *is* the flow. `/doc-mod` and every individual
`/doc-idea`, `/doc-rec`, etc. route through the same underlying `doc-arch`
agent (`src/manifests/content.json`'s `commands[].agent` field is
`"doc-arch"` for all of them except `/doc-ddd` and `/doc-to-sdd`), with
command-specific content telling it which step, or which full sequence, to
run.

### /doc-idea

**Does:** idea refinement — turns a vague concept into a clear product
direction. Pure product discovery: no stack, no APIs, no databases.

**Argument:** `<system>`, `<system>/<module>` or `<system>/<module>/<submodule>`.

**Produces:** a master index description, and optionally
`<node>_idea-brief.md` (artifact is optional for this phase — the interview
still counts, but the write can legitimately be skipped).

**Phase:** `idea` (step 1 of 7; canonical order: idea → rec → prd → refine →
tech → [ddd] → pti).

**Standalone or flow:** both — invoked as the first step of `/doc-arch`, or
directly on its own for one phase.

### /doc-rec

**Does:** requirements elicitation.

**Argument:** `<system>`, `<system>/<module>` or `<system>/<module>/<submodule>`.

**Produces:** `<node>_requirements.md`.

**Phase:** `rec` (step 2 of 7).

**Standalone or flow:** both.

### /doc-prd

**Does:** Product Requirements Document authoring.

**Argument:** `<system>`, `<system>/<module>` or `<system>/<module>/<submodule>`.

**Produces:** `<node>_prd.md`. Prerequisite: `_requirements.md` must exist,
or the command instructs the user to run `/doc-rec` first.

**Phase:** `prd` (step 3 of 7).

**Standalone or flow:** both.

### /doc-refine

**Does:** audits existing user stories against INVEST criteria. Never adds,
deletes or changes story scope without explicit confirmation. With no
argument, runs standalone: refines a single story the user pastes in,
independent of any node or phase.

**Argument:** `<system>`, `<system>/<module>`, `<system>/<module>/<submodule>`,
or empty for standalone mode.

**Produces:** `<node>_refinement.md`, and an updated `_prd.md` user-stories
section if the user approves changes. In standalone mode: an inline refined
story, no file.

**Phase:** `refine` — the only `kind: audit` phase; it elicits nothing new
and owns no topics of its own (step 4 of 7).

**Standalone or flow:** both — and uniquely, can run with no node argument at
all in its standalone form.

### /doc-tech

**Does:** technical specification. For a module, always asks whether it
inherits the parent system's architecture or diverges from it.

**Argument:** `<system>`, `<system>/<module>` or `<system>/<module>/<submodule>`.

**Produces:** `<node>_tech-spec.md`. Prerequisite: `_prd.md`.

**Phase:** `tech` (step 5 of 7). Its `inheritance-mode` topic is required
only for `module`/`submodule` node types (see the [`topics`](#topics)
example above).

**Standalone or flow:** both.

### /doc-ddd

**Does:** optional Database Design Document — entities, ERD, schema,
relationships, integrity constraints, design rationale. Triggered by an
explicit invocation, by hard signals in the project (`.sql`, `migrations/`,
`schema.prisma`, `models/`), or by the orchestrator asking after `tech`.

**Argument:** `<system>`, `<system>/<module>` or `<system>/<module>/<submodule>`.

**Produces:** `<node>_db-design.md`. Prerequisite: `doc-tech` should be
complete first (or schema/migration files provided manually).

**Phase:** `ddd` — the only `optional: true` phase; declining it is recorded
via `decide-phase` and never re-asked (between `tech` and `pti` in the
canonical order).

**Standalone or flow:** both. Unlike every other canonical-phase command,
`/doc-ddd`'s own agent is `doc-ddd` in the command manifest, not `doc-arch`.

### /doc-pti

**Does:** breaks the PRD into an executable issue list. Generates a local
`.md` file by default; GitHub publishing only on explicit user request.

**Argument:** `<system>`, `<system>/<module>` or `<system>/<module>/<submodule>`.

**Produces:** `<node>_issues.md`. Prerequisite: `_prd.md`.

**Phase:** `pti` (step 6 of 7, last in the canonical order — `sdd-commit`'s
`agent_sdd_context_project` compaction deliberately excludes it: an agent
compacting context is normally already working one issue, and folding the
whole issue list back in fights the point of compacting).

**Standalone or flow:** both.

### /doc-mod

**Does:** initializes a new module (or sub-module) inside an evolving
product and runs its full workflow: idea → rec → prd → refine → tech → [ddd]
→ pti, pausing between steps, verifying the parent system exists and uses
the evolving archetype first.

**Argument:** `<system> <module>` (creates `modules/<module>/`), or
`<system>/<module> <submodule>` (creates `modules/<submodule>/` inside the
module).

**Produces:** the module's index (bidirectionally linked to the parent), the
parent index updated to list it, and every canonical phase artifact for the
module.

**Phase:** none of its own — same orchestrator as `/doc-arch`, scoped to a
new module instead of a new system.

**Standalone or flow:** part of the flow — it is how a module enters it.

### /doc-to-sdd

**Does:** compacts existing documentation (business-layer:
`_requirements.md`/`_prd.md`; technical-layer: `_tech-spec.md`/`_db-design.md`)
into two LLM-optimized context files, so an agent reads two documents instead
of up to seven. Submits the compaction via `sdd-commit`
(step 5 in its own skill: "Submit the compacted business and technical
markdown, plus the decisions and preserved TBDs, via `sdd-commit`").

**Argument:** `<system>`, `<system>/<module>` or `<system>/<module>/<submodule>`.
Requires at least one business-layer or one technical-layer artifact to
already exist.

**Produces:** `agent_sdd_context_project/<prefix>_sdd-context.md` and
`..._sdd-tech-context.md`, plus `manifest.json`. Output is always in English
regardless of the source documentation's language.

**Phase:** none — not one of the seven canonical phases; it reads across
whichever of them are present.

**Standalone or flow:** standalone. Explicitly **not** part of the
`/doc-arch` sequence; can run after any combination of completed phases.

### /doc-feat

**Does:** the legacy feature documentation mini-flow, for documenting one
feature inside an existing (often pre-doc-agent-ai) codebase without
bringing the whole system into the pipeline. Parses a legacy path, a
description, and an optional `--scope local|cross|none`; runs scope
classification (unless `--scope` was given), then `rec-lite` → `prd-lite` →
optionally `tech` (risk-gated: fires on high/medium risks in the PRD, a
`cross`-scope mode, or explicit user opt-in) → `pti`.

**Argument:** `<legacy-path> <description> [--scope local <path> | cross
<pattern> | none]`.

**Produces:** `<system>-features/<slug>/`, containing `<slug>_requirements.md`,
`<slug>_prd.md`, optionally `<slug>_tech-spec.md`, `<slug>_issues.md`, and a
local master index — a sibling of `<system>/`, never written inside it.

**Phase:** none. See
[#4 above](#4-the-legacy-doc-feat-mini-flow-sits-outside-the-v5-pipeline) —
this whole mini-flow never invokes the binary, carries no answer records, and
is invisible to `status`, `validate`, `commit-phase` and `doctor`.

**Standalone or flow:** standalone, and deliberately outside `/doc-arch`'s
flow. Tracked for a possible redesign in
[issue #97](https://github.com/zeshone/doc-agent-ai/issues/97).
