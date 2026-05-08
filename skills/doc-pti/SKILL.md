---
name: doc-pti
description: Break a PRD into independently-grabbable local issues using tracer-bullet vertical slices. Use when user wants to convert a PRD to executable work items, create implementation tickets, or break down a PRD into work.
author: Zesh-One
license: MIT
---

# PRD to Issues

Break a PRD into independently-grabbable issues using vertical slices (tracer bullets).

Default output is a local `.md` file. GitHub issues are OPTIONAL and only created if the user explicitly asks.

## Process

### 1. Locate the PRD

Read the PRD artifact provided by the calling flow.

If the PRD is missing, STOP and tell the user the exact prerequisite command to run first.

### 2. Explore the codebase (optional)

Optional. Only do this if it materially helps clarify dependencies or current constraints.

Do NOT invent missing implementation details from the codebase if the PRD leaves a product decision open.

### 3. Extract execution context from the PRD

Before drafting slices, identify:

- user stories covered by the PRD
- acceptance criteria already defined
- explicit non-goals
- dependencies and rollout constraints
- open decisions / TBDs / missing definitions that affect execution

If a slice cannot become executable without missing context, do NOT fill the gap yourself. Either:

- ask the user,
- mark the slice as `HITL`, or
- leave an explicit `TBD` / dependency in the slice.

### 4. Draft vertical slices

Break the PRD into **tracer bullet** issues. Each issue is a thin vertical slice that cuts through ALL integration layers end-to-end, NOT a horizontal slice of one layer.

Slices may be HITL (Human In The Loop — requires human interaction, such as an architectural decision or design review) or AFK (Away From Keyboard — fully automatable, can be implemented and merged without human interaction). Prefer AFK over HITL where possible.

<vertical-slice-rules>
- Each slice delivers a narrow but COMPLETE path through every layer (schema, API, UI, tests)
- A completed slice is demoable or verifiable on its own
- Prefer many thin slices over few thick ones
- Title the slice by user-visible behavior or business outcome, NOT by layer work
- Avoid vague or horizontal titles like "crear API", "hacer DB", "armar frontend", "conectar backend"
- Each slice must be clear enough that someone can understand what to build, how to validate it, and what blocks it
- If the PRD is underspecified, make the uncertainty visible instead of hiding it
</vertical-slice-rules>

### 5. Present the breakdown for approval

Present the proposed breakdown as a numbered list. For each slice, show:

- **Title**: short descriptive name
- **Type**: HITL / AFK
- **Blocked by**: which other slices (if any) must complete first
- **User stories covered**: which user stories from the PRD this addresses
- **End-to-end behavior**: what becomes true for the user or system when the slice is done
- **Validation**: how the slice will be verified
- **Open items**: explicit `TBD`, missing definition, or human decision if applicable

Ask the user:

- Does the granularity feel right? (too coarse / too fine)
- Are the dependency relationships correct?
- Should any slices be merged or split further?
- Are the correct slices marked as HITL and AFK?

Iterate until the user approves the breakdown.

### 6. Generate the local issues file

Write the approved slices to the local `_issues.md` artifact first.

Create issues in dependency order (blockers first).

<issue-template>
# Issues — <System or Module Name>

> Breakdown of the PRD into executable, reviewable vertical slices.
> Default output: local file.
> GitHub only if the user explicitly requests it.

---

## Issue 01 — <Behavior-oriented title>

**Type:** AFK / HITL
**Blocked by:** None / Issue 0N / TBD
**Covered user stories:** Story 1, Story 3

### What end-to-end behavior is being built

Explain what the user can do or which system flow becomes operational end-to-end when this slice is complete.

### Verifiable acceptance criteria

- [ ] Verifiable criterion 1
- [ ] Verifiable criterion 2
- [ ] Verifiable criterion 3

### Validation

- How this slice is demonstrated or tested

### Dependencies and blockers

- Concrete dependency or "None"

### Open decisions / TBD

- Pending decision, missing information, or "None"

### Implementation notes

- Only if they add operational clarity without turning the issue into a horizontal layer-by-layer breakdown
</issue-template>

### 7. Optional: publish to GitHub only on explicit request

If the user explicitly asks to publish to GitHub, then transform the approved local issues into GitHub issues.

Rules for GitHub publishing:

- Never make GitHub the default path
- Preserve the same titles, dependencies, AFK/HITL type, and acceptance criteria from the local file
- Create issues in dependency order
- Do NOT modify or close the parent PRD artifact unless explicitly asked

- If there is no GitHub publishing request, STOP after generating the local file
