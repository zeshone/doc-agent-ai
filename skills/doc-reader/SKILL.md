---
name: doc-reader
description: "Trigger: reading project documentation, understanding repo architecture, or working on a feature/module that needs compacted SDD context — in either vault or in-project docs mode. Ask `doc-agent-ai status` for the resolved paths and read ONLY what it returns; never read the full docs tree."
license: Apache-2.0
metadata:
  author: gentleman-programming
  version: "2.0"
---

## Activation Contract

Load this skill whenever documentation context would inform an architectural or feature decision, regardless of whether this project stores documentation in-project or in a vault. The program — not this prompt — knows where a node's documentation lives; you ask it, you never compose the path yourself.

The compacted context is scoped per node. A system and each of its modules and submodules carry their own compacted context: reading the system's context while working on one of its features is reading the wrong document, not a lesser version of the right one.

## Hard Rules

- **Never compose a documentation path by hand, in either mode.** Ask `doc-agent-ai status` and read exactly the paths it returns. No directory or filename from the docs tree may appear anywhere in this skill or in your reasoning.
- **Read ONLY the paths named in `sddContext.outputs`** for the node you are working on — the business-layer file (requirements, PRD, decisions) and the technical-layer file (architecture, tech spec, DB design). These are the ONLY authoritative documentation context.
- **Never read anything else.** The full docs tree — including `_prd.md`, `_tech-spec.md`, or any other document under `target.docsRoot` — is EXCLUDED from agent context, no matter how thin `sddContext.outputs` looks.
- **The node selects which document you get; it is not optional context.** A project's `.doc-agent.json` marker names a default node for the repository as a whole. An explicit `--node <system>`, `--node <system>/<module>`, or `--node <system>/<module>/<submodule>` always overrides that default, and is required whenever you are working on a specific module, feature, or submodule rather than the project as a whole.
- **Never guess a node.** The program cannot enumerate a node's children — there is no way to ask it "what modules does this system have" — so if you do not already know the node for the feature or module you are on, ask the human. A guessed node silently returns somebody else's documentation, or none, and you cannot tell which.
- **If you cannot run `doc-agent-ai` at all** — no shell tool available, or the binary is not on `PATH` — stop and say exactly that. Do not fall back to reading the docs tree or guessing a path: a guessed path reads somebody else's documentation, or nothing, and you cannot tell which.
- Never invent context from a partial or stale read — if context is missing, absent, or reported stale, stop and surface the gap rather than filling it in.

## Decision Gates

| Situation | Action |
|---|---|
| You cannot run a shell command, or `doc-agent-ai` is not on `PATH` | Stop; say exactly that. Never read the docs tree or guess a path instead. |
| You don't know which node covers the feature/module you're on | Ask the human for the node (`<system>`, `<system>/<module>`, or `<system>/<module>/<submodule>`). Never guess. |
| `status` refuses, needing `--node`, and you already know the node | Re-run with `--node <the node>` |
| `status` refuses, needing `--node`, and you do NOT know the node | Stop; ask the human — do not guess and do not fall back to the docs tree |
| `sddContext` is missing from the response (node/docsRoot unresolved) | Relay `nextAction.reason` (and `blockedReasons`) verbatim; do not guess a path |
| `sddContext.state` is `"fresh"` | Read every path in `sddContext.outputs` |
| `sddContext.state` is `"stale"` | Still read every path in `sddContext.outputs`, but tell the user the program reports them out of date, naming what changed from `sddContext.drifted` / `sddContext.appeared` |
| `sddContext.state` is `"absent"` | Stop; suggest running `/doc-to-sdd` once for this node. Do NOT fall back to the docs tree. |
| Working on a pure code task with no doc dependency | Skip this skill entirely |

## Execution Steps

1. **Decide the node.** Working on the project/repository as a whole: omit `--node` and let the marker's default resolve. Working on a specific feature, module, or submodule: pass its node explicitly, e.g. `--node <system>/<module>` — an explicit `--node` always wins over the marker's default. If you don't know which node the feature/module you're on maps to, ask the human before doing anything else — never guess.
2. **Confirm you can run the program.** If you have no shell tool, or `doc-agent-ai` is not on `PATH`, stop now and say so. Do not proceed to the steps below.
3. **Ask for status:**
   ```
   doc-agent-ai status [--node <system[/module[/submodule]]>]
   ```
4. **Read the JSON on stdout** and branch on `sddContext.state`:
   - Missing entirely → the node or its docs root could not be resolved. Report `nextAction.reason` (and `blockedReasons`, if any) and stop.
   - `"absent"` → no compacted context exists for this node yet. Suggest `/doc-to-sdd` once, and stop. Do not open `target.docsRoot`.
   - `"stale"` → read every path in `sddContext.outputs`, verbatim, but tell the user they are out of date and name what changed (`sddContext.drifted`, `sddContext.appeared`).
   - `"fresh"` → read every path in `sddContext.outputs`, verbatim.
5. **Never open a path you composed yourself.** Only the paths the program returned in `sddContext.outputs` are documentation context — never `target.docsRoot` directly, and never `_prd.md`, `_tech-spec.md`, or any other file in the tree.

## Output Contract

Provide documentation context sourced only from the paths in `sddContext.outputs`. State the node you asked for and which files you read. If the context was stale, say so and name what drifted or appeared. If the context is absent, or the node could not be resolved, surface that plainly and recommend the corrective step — `/doc-to-sdd`, or asking the human for the node — instead of guessing.
