---
name: doc-refinement
description: >
  Audits and professionally refines user stories. In automatic mode (triggered by the arch flow),
  it audits the user stories in the previously generated PRD against INVEST criteria. In standalone
  mode, it takes a user story written by the user and refines it into a professional, verifiable story.
  Trigger: refine, story audit, user story refinement, INVEST audit, story quality.
author: Zesh-One
license: MIT
---

# User Story Refinement — Audit and Sanitization

## Positioning

Runs **immediately after** `doc-prd` and **before** `doc-tech`.

Two modes:
1. **Audit mode** (`refine <sistema>`): reads the PRD, audits its user stories, presents findings, applies corrections only with user approval.
2. **Standalone mode** (`refine`): user provides a story, skill refines it inline — no files touched.

Focus is exclusively **user story quality**. Does not modify requirements, add features, or touch technical sections.

## INVEST — Operational Checklist

- **Independent**: deliverable without other stories? No circular dependencies.
- **Negotiable**: WHAT is clear, HOW is flexible. No implementation prescribed.
- **Valuable**: real user/business outcome in "so that". Not a technical task.
- **Small**: ~3-5 acceptance criteria max. Completable in one iteration.
- **Testable**: binary pass/fail criteria. No "properly", "correctly", "fast".
- **Format**: "As a [user], I want [action] so that [benefit]."

## Mode 1 — Automatic Audit (`refine <sistema>`)

1. Read the PRD from the previous phase (`<node>_prd.md`).
2. Extract all user stories from the "User Stories" section.
3. Audit each against the INVEST checklist. Classify: ✅ OK / ⚠️ WARNING / 🔴 ISSUE.
4. Present the audit report with: problems detected, refined version, suggested acceptance criteria in Given/When/Then format.
5. Ask: Ask for confirmation before modifying the PRD — do not proceed without explicit approval.

### Audit Mode Rules
- Only audit user stories. Do not touch requirements, architecture, risks, or other PRD sections.
- Do not add new stories or delete existing ones.
- Do not change intent — refined version must preserve original goal.
- Present report before applying changes. Never auto-modify the PRD.
- All acceptance criteria in Given/When/Then format.

## Mode 2 — Standalone Refinement (`refine`)

1. Receive the story from the user.
2. Analyze against INVEST.
3. Ask up to 3 clarification questions if ambiguous.
4. Deliver refined story with: canonical format, 3+ acceptance criteria in Given/When/Then, brief explanation of changes made.

### Standalone Mode Rules
- Max 3 clarification questions. If still ambiguous, refine best-effort and mark open areas.
- Do not judge the idea — refinement improves expression, not value.
- Always deliver Given/When/Then acceptance criteria. No code, no architecture, no stack references.

## Anti-Patterns

- **Technical tasks disguised as stories**: "Refactor caching layer" → task, not story.
- **Epics disguised as stories**: too large, 10+ acceptance criteria.
- **Vague acceptance criteria**: "handles errors gracefully", "works correctly".
- **Implementation prescribed**: "Use Redis with TTL of 300s".
- **Circular dependencies**: Story A needs B which needs A.

## Quality Checklist

- [ ] Story traces to a PRD requirement
- [ ] Each INVEST criterion passes
- [ ] Acceptance criteria use Given/When/Then
- [ ] No vague words in criteria ("properly", "correctly", "fast")
- [ ] Dependencies are one-way only
- [ ] 3+ acceptance criteria per story
- [ ] No code, architecture, or stack references
- [ ] Audit mode: user confirmed before PRD modification
- [ ] Standalone mode: ≤3 clarification questions asked
