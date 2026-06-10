---
name: doc-reader
description: "Trigger: reading project docs, understanding repo architecture, working on features with in-project docs mode. Use ONLY the /doc-to-sdd compacted context files — never read the full docs tree."
license: Apache-2.0
metadata:
  author: gentleman-programming
  version: "1.0"
---

## Activation Contract

Load this skill when working on a project that uses in-project docs mode (`docs/doc-agent/`) and you need documentation context to inform architectural or feature decisions.

## Hard Rules

- Read ONLY files under `docs/doc-agent/agent_sdd_context_project/`:
  - `_sdd-context.md` — business layer (requirements, PRD, decisions)
  - `_sdd-tech-context.md` — technical layer (architecture, tech spec, DB design)
- Do NOT read any other file under `docs/doc-agent/` as documentation context. The following are EXCLUDED:
  - `_prd.md`
  - `_tech-spec.md`
  - Any `.md` file directly under `docs/doc-agent/` or its module/feature subdirectories
- If `agent_sdd_context_project/` does not exist or the context files are absent, suggest running `/doc-to-sdd` once to generate them. Do NOT fall back to reading the full docs tree.
- Never invent context from partial reads — if context is missing, stop and surface the gap.

## Decision Gates

| Situation | Action |
|-----------|--------|
| Context files exist | Read `_sdd-context.md` and/or `_sdd-tech-context.md` as needed |
| Context files absent | Suggest `/doc-to-sdd` — do not read the full docs tree |
| Only one context file exists | Read the available one; note the missing layer |
| Working on a pure code task with no doc dependency | Skip this skill entirely |

## Execution Steps

1. Check if `docs/doc-agent/agent_sdd_context_project/_sdd-context.md` exists.
2. Check if `docs/doc-agent/agent_sdd_context_project/_sdd-tech-context.md` exists.
3. Read whichever files exist — they are the ONLY authoritative documentation context.
4. If neither exists, stop and tell the user: "Run `/doc-to-sdd` once to generate the compacted context files."
5. Do NOT open `_prd.md`, `_tech-spec.md`, or any other file in the docs tree.

## Output Contract

Provide documentation context from the compacted files only. State which files you read. If context is incomplete, surface the gap and recommend running `/doc-to-sdd`.
