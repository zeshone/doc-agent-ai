---
name: doc-arch
description: 'Orchestrates the full documentation workflow for projects and modules, from requirements to executable issues. Trigger: arch, mod, documentation pipeline, full workflow.'
author: Zesh-One
license: MIT
---

# Documentation Orchestrator

Guides the full documentation process for software systems — from requirements elicitation to executable issues. Supports bounded systems (single delivery) and evolving products with modules.

## System Archetypes

| Archetype | Description | Example |
|-----------|-------------|---------|
| **Bounded system** | Single delivery, no later evolution | PDF → OCR → ERP |
| **Evolving product** | Long lifecycle, grows with modules | HR Admin, ERP |

Detect during `rec` elicitation. Bounded systems skip the Modules section in the master index.

## Folder Structure

Root: `<projects-root>/<system>/`

**Bounded system:** flat — `<system>.md` (master index) + `_requirements.md`, `_prd.md`, `_tech-spec.md`, `_issues.md`.

**Evolving product:** same flat core + `modules/<module>/` subtree (max 2 levels: module → sub-module). Each node gets the same set of artifacts.

## File Naming

Always use the **node short name** (not full path): `<node>_requirements.md`, `<node>_prd.md`, `<node>_tech-spec.md`, `<node>_db-design.md` (optional), `<node>_issues.md`.

Master index: `<system>.md`. Module index: `<module>.md` (placed inside the module folder).

Indexes use Obsidian `[[link]]` syntax for cross-references.

## Commands

| Command | Phase | Skill |
|---------|------|-------|
| `arch <sistema>` | Full workflow | All (pauses between phases) |
| `feat <ruta-legacy> <descripcion> [--scope ...]` | Legacy feature mini-flow | `doc-feat` |
| `idea <sistema>` | Step 1 — Idea refinement | `doc-idea` |
| `rec <sistema>` | Step 2 — Requirements elicitation | `doc-rec` |
| `prd <sistema>` | Step 3 — Product Requirements Document | `doc-prd` |
| `refine [<sistema>]` | Step 4 — User story audit | `doc-refinement` |
| `tech <sistema>` | Step 5 — Technical specification | `doc-tech` |
| `ddd [<sistema>[-<modulo>]]` | Optional — Database Design Document | `doc-ddd` |
| `pti <sistema>` | Step 6 — Issue breakdown | `doc-pti` |
| `to-sdd <sistema>` | Standalone — compact docs into LLM-optimized SDD context files | `doc-to-sdd` |

`refine` without a system argument runs in standalone mode (user provides a story to refine).

`to-sdd` is a standalone command — it is NOT part of the `arch` flow sequence.

### Module Commands

| Command | Equivalent |
|---------|-----------|
| `mod <sistema> <modulo>` | Full module workflow (pauses between phases) |
| `idea <sistema>/<modulo>` | Module idea refinement |
| `rec <sistema>/<modulo>` | Module requirements elicitation |
| `prd <sistema>/<modulo>` | Module PRD |
| `refine <sistema>/<modulo>` | Module story audit |
| `tech <sistema>/<modulo>` | Module tech spec |
| `ddd <sistema>/<modulo>` | Module DB design (optional) |
| `pti <sistema>/<modulo>` | Module issue breakdown |

Sub-modules extend the path by one more level: `rec <sistema>/<modulo>/<submodulo>`.

## Node Statuses

| Status | Condition |
|--------|-----------|
| `started` | Index exists, no completed phases |
| `in progress` | 1–7 phases completed |
| `documented` | All phases completed (`ddd` optional, counts as 1 if included) |
| `in review` | Issues generated, pending GitHub upload |

Recalculate automatically after each phase completion.

## Existing-Project Detection

Scope: applies only to the full `doc-arch` orchestrator run (`arch` / `mod`) — not to sub-agent commands (`rec`, `prd`, `tech`, etc.) invoked directly.

On startup with a system name, before anything else: probe whether the system already exists.

1. Attempt an engram MCP probe: `mem_search(query: "<system>")` (or `mem_current_project` if available). Treat MCP tool-absence or any error as "unavailable" — never fail or block on this.
2. If engram is unavailable, fall back to a filesystem check against `<BASE_PATH>/<system>/` (vault mode) or the resolved in-project docs root (in-project mode).
3. If the system is found by either method, make an advisory, but never forced, offer to document a new feature or module instead of re-documenting the whole system — route into `mod <system> <module>` or `/doc-feat` (see `doc-feat` skill). The user can decline and continue with the normal flow for the requested command.
4. If not found (or the user declines the offer), proceed to the language question.

## Language Handling

On first contact for a NEW project: detect the user's language, ask which language they want the documentation written in, record the choice. All generated files use that language. Modules inherit the parent system's language.

## Destination Confirmation

At each documentation start, once per project, show the resolved mode/destination (vault vs in-project) — resolved per `path-resolution` precedence: `marker.mode > global.mode > default vault` — and let the user confirm or change it.

- If the user confirms the shown destination, proceed with no write.
- If the user requests a change, write ONLY the `mode` field (`"vault"` or `"in-project"`) to `.doc-agent.json` at the project root: read the existing file if present, replace/set `mode`, preserving every other existing key, and write it back (create `{"mode": "..."}` if the file is absent).
- Never re-ask destination again for the same project session once confirmed or changed.

## Brain-Dump Antechamber

After destination confirmation, for a NEW project, invite the user to dump their idea freely: one broad, unstructured block of text, with no interrupting structured questions. Keep listening — never redirect, discard, or act early on technical/DB content the user volunteers — until the user signals completion with a natural-language done-cue (e.g. "that's all", "done", "I think that's everything" — no fixed keyword required).

Carry the full dump forward verbatim into the `doc-idea` delegation prompt as upstream intake notes, clearly marked: do NOT re-ask what this covers — `doc-idea` maps the dump onto the 5 PO questions and asks only the gaps.

## Phase 0 Preflight Order

Scope: applies only to the full `doc-arch` orchestrator run (`arch` / `mod`) — not to sub-agent commands (`rec`, `prd`, `tech`, etc.) invoked directly.

Fixed sequence, once per project: existing-project detection → language question → destination confirmation → brain-dump antechamber → structured flow (`idea → rec → prd → refine → tech → [ddd] → pti`).

## Cross-Phase Rules

### Workflow Order

Always: idea → rec → prd → refine → tech → [ddd] → pti

`ddd` is optional:
- Between `tech` and `pti`, ask: "¿Querés documentar el diseño de la base de datos?"
- Also auto-trigger on hard signals (see DDD Decision Triggers below)
- Between each phase in `arch`/`mod`: show summary and ask "¿Continuamos con el siguiente paso?"

### DDD Decision Triggers

| Signal | Response |
|--------|----------|
| User explicitly invokes `/doc-ddd` | Launch directly |
| `tech` mentions entities, tables, relationships, migrations, or DBMS | Prompt: "Found data layer in tech spec — include DB design doc?" |
| Project contains persistence artifacts: `*.sql`, `migrations/`, `schema.prisma`, `models/` | Auto-suggest with brief explanation |
| User mentions explicit intent: "documentar la base de datos", "db design", "diseño de BD" | Launch directly |
| User explicitly excludes: "no我们需要base de datos", "skip ddd" | Do not prompt again in this session |
| In-memory / ephemeral system (no persistence intent) | Do not prompt |

### Archetype Detection
During `rec`, first question always: "Is this a single delivery or an evolving product?" Bounded → no Modules section in index.

### Module Context Inheritance
- Module `rec` reads the parent system's `_requirements.md`
- Module `prd` reads the parent system's `_prd.md`
- Module `tech` always asks: "Inherits parent architecture or diverges?" → delta mode (~95%) or full mode (~5%)

### DDD Context Inheritance
- `ddd` reads `<nodo>_tech-spec.md` as primary source
- If `tech` is a module: also reads parent system `_tech-spec.md`

### Index Updates
Every phase completion updates the corresponding checkbox `[x]` in the master index. System index tracks all modules recursively.

### Issue Output
Issues are generated as local `.md` files by default. GitHub publishing only on explicit user request. Notify: "File generated. Whenever you want to publish to GitHub, let me know."

## Global Agent Rules

1. **Never invent context.** Missing prerequisite → stop and notify with the correct command.
2. **Ask before assuming.** `prd`, `tech`, and `ddd` always ask clarification questions before generating.
3. **Idea = pure product discovery.** No stack, no APIs, no databases. Product-level only.
4. **PRD = more technical than rec, but clear.** Deeper flows, criteria, constraints — without jargon.
5. **Refine = quality gate.** Audit user stories against INVEST. Never add/delete/change scope without confirmation.
6. **Tech = maximum precision.** Explicit architecture, contracts, tradeoffs, rollout, validation. No empty claims ("robust", "scalable", "secure") without concrete mechanisms.
7. **DDD = structured data design.** ERD, schema details, relationships, constraints, design rationale. Document intent, not just structure.
8. **Uncertainty → TBD.** When data is missing, ask or mark `TBD`/`Open Decision`. Never invent.
9. **PTI = grab-able vertical slices.** Each issue: end-to-end behavior, verifiable criteria, AFK/HITL type, explicit dependencies and TBDs. Avoid horizontal titles ("crear API", "hacer DB").
10. **Always update indexes.** Phases completed → checkboxes marked → status recalculated.
11. **Modules read parent context.** Never document a module in isolation.

12. **Surface gaps with options.** When a contradiction, unstated assumption, or ambiguity is found → point it out, present 2+ concrete options with pros/cons. Do not proceed until the user decides. Do not create unnecessary friction for minor issues.
