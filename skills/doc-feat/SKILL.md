---
name: doc-feat
description: 'Orchestrate the legacy feature documentation mini-flow. Trigger: feat, legacy feature, feature documentation, document a feature, /feat.'
author: Zesh-One
license: MIT
---

# Legacy Feature Documentation Orchestrator

Activated by `/doc-feat`. Never invoked standalone by the user.

## Activation Contract

Triggered only when the user invokes `/doc-feat <ruta-legacy> <descripcion> [--scope ...]`.
Not a subagent — this skill IS the orchestrator for the entire mini-flow.

## Argument Parsing

Parse `$ARGUMENTS` as a space-separated token list. Preserve quoted substrings as single tokens.

**Step A — Locate `--scope`**

```
i = index of token == "--scope"
IF i == -1:
  no flag. positional = all tokens. go to Step C.
ELSE:
  positional = tokens[0..i-1]
  mode_token = tokens[i+1]   (REQUIRED)
  arg_token  = tokens[i+2]   (REQUIRED if mode is local or cross; FORBIDDEN if mode is none)
```

**Step B — Validate flag**

- `mode_token` NOT IN `{local, cross, none}` → emit usage error naming valid modes. STOP.
- `mode_token` IN `{local, cross}` AND `arg_token` missing → emit usage error. STOP.
- `mode_token == none` AND `arg_token` present → emit usage error. STOP.

**Step C — Validate positional**

- `len(positional) < 2` → emit: `/doc-feat <ruta-legacy> <descripcion> [--scope local <path> | cross <pattern> | none]`. STOP.
- `ruta-legacy = positional[0]`
- `descripcion = join(positional[1..], " ")`

**Step D — Validate path**

- Canonicalize `ruta-legacy`. If it does not resolve to an existing directory → emit error naming the path. STOP.

**Error cases table**

| Condition | Message |
|-----------|---------|
| `< 2` positional args | Usage: `/doc-feat <ruta-legacy> <descripcion> [--scope ...]` |
| Path not found / not a dir | `<path> does not exist or is not a directory` |
| Unknown `--scope` mode | `Unknown mode '<x>'. Valid modes: local, cross, none` |
| `--scope local/cross` without arg | `--scope <mode> requires a following argument` |
| `--scope none` with extra arg | `--scope none takes no argument` |

## System Name (`<sistema>`) Derivation

Derive in this exact order. Stop at the first successful candidate.

**Step 1:** `candidate = kebab(lowercase(basename(canonicalize(ruta-legacy))))`

**Step 2:** If `len(candidate) >= 4` AND `candidate` NOT IN the low-information wordlist below → propose `candidate`, ask user to confirm.

Low-information wordlist:
- src, app, lib, legacy, code, repo, project, dist, build, out, target, tmp, temp, work, workspace

**Step 3:** Run `git -C <ruta-legacy> config --get remote.origin.url`. If success, derive basename without `.git`, kebab it. If it passes the same wordlist check → propose it (surfacing it came from git remote). Ask user to confirm.

**Step 4:** If both failed → surface the rejected candidate and failed fallbacks. Prompt user explicitly. No silent guess.

User's confirmed/overridden name = `<sistema>`.

## Slug Derivation + Uniqueness

`slug = kebab(first 4-6 significant words of descripcion)`, max 50 characters.

If `<BASE_PATH><sistema>-features/<slug>/` already exists → append `-2`, `-3`, ... until unique. Inform the user of the final slug used.

## Mini-Flow Sequence

| Step | Skill | Condition |
|------|-------|-----------|
| 1 | `doc-scope` | Only if `--scope` flag NOT given |
| 2 | `doc-rec-lite` | Always |
| 3 | `doc-prd-lite` | Always |
| 4 | `doc-tech` | Only if risk gate fires (see below) |
| 5 | `doc-pti` | Always |

Later phases MUST NOT start until the previous phase is confirmed complete.

## Risk Gate (before Step 4)

Trigger `doc-tech` if ANY of the following is true:

| Indicator | Strength |
|-----------|----------|
| `<slug>_prd.md` `## Risks` section is non-empty AND contains at least one row tagged `high` or `medium` (case-insensitive) | MUST trigger |
| Resolved scope mode == `cross` | MUST trigger |
| User answers `tech`, `yes`, `sí`, `s`, or `y` at the explicit pre-pti confirmation prompt | MUST trigger |
| User answers `skip`, `no`, or `n` AND no other MUST trigger fired | MUST skip |
| Risks empty AND mode is `local`/`none` AND user has not asked | MAY skip (default: skip) |

Always emit the confirmation prompt before pti — the user can opt in even when no automatic trigger fired.

## Output Layout

All files go under `<BASE_PATH><sistema>-features/<slug>/`.

Required output files:

| File | Phase |
|------|-------|
| `<slug>_requirements.md` | rec-lite |
| `<slug>_prd.md` | prd-lite |
| `<slug>_tech-spec.md` | doc-tech (only if risk gate triggered) |
| `<slug>_issues.md` | doc-pti |
| `<slug>.md` | master index — Obsidian `[[links]]` to all siblings |

The master index MUST link to `<sistema>/` IF that directory exists, and MUST NOT fail if it does not.

The output directory MUST be a sibling of `<BASE_PATH><sistema>/` if that path exists. MUST NOT write inside an existing `<BASE_PATH><sistema>/` tree.
