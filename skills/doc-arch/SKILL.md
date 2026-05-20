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

Always use the **node short name** (not full path): `<node>_requirements.md`, `<node>_prd.md`, `<node>_tech-spec.md`, `<node>_issues.md`.

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
| `pti <sistema>` | Step 6 — Issue breakdown | `doc-pti` |

`refine` without a system argument runs in standalone mode (user provides a story to refine).

### Module Commands

| Command | Equivalent |
|---------|-----------|
| `mod <sistema> <modulo>` | Full module workflow (pauses between phases) |
| `idea <sistema>/<modulo>` | Module idea refinement |
| `rec <sistema>/<modulo>` | Module requirements elicitation |
| `prd <sistema>/<modulo>` | Module PRD |
| `refine <sistema>/<modulo>` | Module story audit |
| `tech <sistema>/<modulo>` | Module tech spec |
| `pti <sistema>/<modulo>` | Module issue breakdown |

Sub-modules extend the path by one more level: `rec <sistema>/<modulo>/<submodulo>`.

## Node Statuses

| Status | Condition |
|--------|-----------|
| `started` | Index exists, no completed phases |
| `in progress` | 1–5 phases completed |
| `documented` | All 6 phases completed |
| `in review` | Issues generated, pending GitHub upload |

Recalculate automatically after each phase completion.

## Language Handling

On first contact for a NEW project: detect the user's language, ask which language they want the documentation written in, record the choice. All generated files use that language. Modules inherit the parent system's language.

## Cross-Phase Rules

### Workflow Order
Always: idea → rec → prd → refine → tech → pti. Between each phase in `arch`/`mod`: show summary and ask "Do we continue with the next step?"

### Archetype Detection
During `rec`, first question always: "Is this a single delivery or an evolving product?" Bounded → no Modules section in index.

### Module Context Inheritance
- Module `rec` reads the parent system's `_requirements.md`
- Module `prd` reads the parent system's `_prd.md`
- Module `tech` always asks: "Inherits parent architecture or diverges?" → delta mode (~95%) or full mode (~5%)

### Index Updates
Every phase completion updates the corresponding checkbox `[x]` in the master index. System index tracks all modules recursively.

### Issue Output
Issues are generated as local `.md` files by default. GitHub publishing only on explicit user request. Notify: "File generated. Whenever you want to publish to GitHub, let me know."

## Global Agent Rules

1. **Never invent context.** Missing prerequisite → stop and notify with the correct command.
2. **Ask before assuming.** `prd` and `tech` always ask clarification questions before generating.
3. **Idea = pure product discovery.** No stack, no APIs, no databases. Product-level only.
4. **PRD = more technical than rec, but clear.** Deeper flows, criteria, constraints — without jargon.
5. **Refine = quality gate.** Audit user stories against INVEST. Never add/delete/change scope without confirmation.
6. **Tech = maximum precision.** Explicit architecture, contracts, tradeoffs, rollout, validation. No empty claims ("robust", "scalable", "secure") without concrete mechanisms.
7. **Uncertainty → TBD.** When data is missing, ask or mark `TBD`/`Open Decision`. Never invent.
8. **PTI = grab-able vertical slices.** Each issue: end-to-end behavior, verifiable criteria, AFK/HITL type, explicit dependencies and TBDs. Avoid horizontal titles ("crear API", "hacer DB").
9. **Always update indexes.** Phases completed → checkboxes marked → status recalculated.
10. **Modules read parent context.** Never document a module in isolation.

11. **Surface gaps with options.** When a contradiction, unstated assumption, or ambiguity is found → point it out, present 2+ concrete options with pros/cons. Do not proceed until the user decides. Do not create unnecessary friction for minor issues.
