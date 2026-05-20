---
name: doc-prd-lite
description: 'Generate a Product Requirements Document for a single legacy feature. Trigger: lite PRD, feature PRD, single-feature product requirements, invoked by doc-feat.'
author: Zesh-One
license: MIT
---

# Product Requirements Document — Feature Scope (Lite)

Invoked ONLY by `doc-feat`. Not user-invokable directly.

## Activation Contract

Inputs from `doc-feat`:
- `<sistema>` — confirmed system name
- `<slug>` — confirmed feature slug
- `<slug>_requirements.md` — completed output from `doc-rec-lite`
- `(mode, anchor_or_pattern)` — resolved scope

## Hard Rules

| Rule | Detail |
|------|--------|
| Single-feature only | The PRD covers `<slug>` exclusively. No multi-feature scope creep. |
| No multi-persona matrix | Skip the full user-persona matrix from `doc-prd`. One or two personas max, described inline. |
| Measurable criteria | Avoid "fast", "easy", "intuitive" unless immediately quantified. |
| Risks section is mandatory | MUST include a `## Risks` section. This feeds the risk gate in `doc-feat`. Tag each risk with `high`, `medium`, or `low`. |
| No TBD burial | Open decisions stay visible as `TBD` with a note. Never buried or paraphrased away. |

## Required Sections

Generate in this order. Do not omit any section.

### 1. Executive Summary
- **Problem Statement** (1-2 sentences for this feature)
- **Proposed Solution** (1-2 sentences)
- **Success Criteria** (2-4 measurable KPIs)

### 2. User Flows
- 1-2 primary flows for this feature (happy path + one exception path)
- User stories in `As a [user], I want [action] so that [benefit]` format
- Acceptance criteria per story in **Given/When/Then** format
- **Non-Goals** — what this feature deliberately does NOT do

### 3. Risks
Mandatory. Tag each risk with severity: `high`, `medium`, or `low`.

If no risks exist, write: `No risks identified at this time.` and leave the section non-empty so the risk gate in `doc-feat` can inspect it reliably.

### 4. Open Decisions
Any `TBD` items with a note explaining what is unknown and why.

## Output Contract

Write `<BASE_PATH><sistema>-features/<slug>/<slug>_prd.md`.

The `## Risks` section MUST be present and MUST contain at minimum the text `No risks identified at this time.` if truly empty — never omit the section header.
