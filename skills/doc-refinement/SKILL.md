---
name: doc-refinement
description: >
  Audits and professionally refines user stories. In automatic mode (triggered by the arch flow),
  it audits the user stories in the previously generated PRD against INVEST criteria. In standalone
  mode, it takes a user story written by the user and refines it into a professional, verifiable story.
author: Zesh-One
license: MIT
---

# User Story Refinement — Audit and Sanitization

## Positioning

`doc-refinement` runs **immediately after** `doc-prd` and **before** `doc-tech`.

It serves two distinct purposes:

1. **Audit mode (automatic)**: When launched as part of the `/arch` flow, it reads the PRD file generated in the previous phase and audits exclusively its user stories against quality criteria.
2. **Standalone mode (manual)**: When the user invokes the command directly, they provide a user story and the skill refines it into a professional version.

It does not modify requirements, add new functionality, or touch technical aspects of the PRD. Its sole focus is **user story quality**.

## Overview

A poorly written user story is the root cause of 80% of implementation problems. This skill applies systematic criteria to detect ambiguous, incomplete, unverifiable, or poorly formatted stories, and corrects them without altering the original intent.

It is based on:

- **INVEST criteria**: Independent, Negotiable, Valuable, Estimable, Small, Testable.
- **Canonical format**: "As a [user], I want [action] so that [benefit]."
- **Product best practices**: measurable acceptance criteria in Given/When/Then format, user-centric language, value focus.

## When to Use

Use this skill when:

- A PRD was just generated and its user stories need auditing (automatic mode).
- The user writes a user story and wants it refined (standalone mode).
- PRD stories are detected to be vague, technical, or lacking clear acceptance criteria.

Do **not** use when:

- No PRD exists (in automatic mode).
- The user is asking for scope changes or new features (that is `doc-prd` or `doc-rec` work).
- The input is not a user story (technical requirements, architecture, etc.).

---

## INVEST Criteria — Full Reference

### I — Independent

```
✅ Can be developed without waiting for other stories
✅ No circular dependencies
❌ "This needs Story X which needs this"
```

The story should deliver value on its own. If it cannot be built without another story, they should be merged or the dependency must be explicit.

### N — Negotiable

```
✅ HOW is flexible (implementation not prescribed)
✅ WHAT is clear (outcome defined)
❌ "Must use React Query with exact caching config"
```

The story describes the desired outcome, not the implementation path. The team negotiates the "how" during development.

### V — Valuable

```
✅ Delivers value to USER or BUSINESS
✅ Value stated explicitly in the "so that" clause
❌ "Refactor database layer" (no user-facing value)
✅ "User sees search results faster because we optimized queries"
```

Every story must trace to a user or business outcome. Technical stories without user value are tasks, not stories.

### E — Estimable

```
✅ Team can estimate complexity (S/M/L)
✅ No major unknowns blocking estimation
❌ "Integrate with external API" (which API? what operations?)
```

The story has enough context for the team to gauge effort. Major unknowns block estimation and should be resolved first.

### S — Small

```
✅ Completable in one iteration
✅ Can be code reviewed in one sitting
❌ 10+ acceptance criteria, spanning multiple components
```

If a story is too large, it should be split. A good heuristic: 3–5 acceptance criteria max.

### T — Testable

```
✅ ALL acceptance criteria are independently verifiable
✅ Given/When/Then format used
❌ "System should handle errors gracefully"
✅ "Given invalid input, then error message X displays"
```

Every criterion must be binary — it passes or it fails. Vague words like "properly", "correctly", "fast", or "intuitive" have no place in acceptance criteria.

---

## Mode 1 — Automatic Audit (arch flow)

This mode activates when `doc-refinement` is called by the `doc-arch` orchestrator as part of the full workflow. In this mode, the skill works **only** with the PRD file generated in the previous phase.

### Audit Protocol

1. **Read the PRD** from the previous phase (`<node>_prd.md`).

2. **Extract all user stories** from the "User Stories" section.

3. **Audit each story** against the INVEST matrix:

| Criterion | What to check | Guiding question |
|-----------|---------------|------------------|
| **Format** | Uses "As a / I want / so that"? | Does the story follow the canonical format? |
| **Independent** | Deliverable without other stories? | Can value be delivered with this story alone? |
| **Valuable** | Clear user benefit in "so that"? | Does "so that" describe an outcome, not a task? |
| **Small** | Small enough for one iteration? | Can it be completed in a sprint? |
| **Testable** | Measurable acceptance criteria? | How do I know it is done? |
| **Language** | Free of unnecessary technical jargon? | Would a non-technical stakeholder understand it? |
| **Unambiguous** | All terms are concrete? | Are "fast", "easy", "intuitive" quantified? |

4. **Classify findings**:

- ✅ **OK**: The story meets the criterion.
- ⚠️ **WARNING**: Minor issues that can be corrected.
- 🔴 **ISSUE**: Structural problems requiring reformulation.

5. **Generate the audit report** and present it to the user:

```markdown
## Audit Report — User Stories

**PRD audited**: `<node>_prd.md`
**Stories found**: N
**Stories OK**: N
**Stories with warnings**: N
**Stories with issues**: N

### Story #1: <original title>

**Status**: ⚠️ WARNING / 🔴 ISSUE

**Problems detected**:
- [Criterion]: description of the problem

**Refined version**:
> As a [user], I want [action] so that [benefit].

**Suggested acceptance criteria**:
- [ ] Given [context], when [action], then [result]
- [ ] Given [context], when [action], then [result]
```

6. **Ask the user** whether to apply the corrections to the PRD. If the user accepts, update the PRD file with the refined stories. If the user declines, leave the report as a reference without modifying the PRD.

### Automatic Mode Rules

- **Only audit user stories from the PRD.** Do not review requirements, architecture, risks, or other sections.
- **Do not add new stories.** Only refine existing ones.
- **Do not delete stories.** If a story is redundant, flag it but do not remove it without confirmation.
- **Do not change intent.** The refined version must preserve the original goal.
- **Present the report before applying changes.** Never modify the PRD without explicit approval.
- **All acceptance criteria must use Given/When/Then format.** Replace vague language with verifiable statements.

---

## Mode 2 — Standalone Refinement (manual)

This mode activates when the user invokes the refinement command directly (outside the arch flow). The user provides a user story and the skill refines it.

### Standalone Protocol

1. **Receive the story** from the user. It can be a simple sentence, a paragraph, or structured text.

2. **Analyze the story** against INVEST criteria:
   - Does it follow the canonical format?
   - Is the user persona clear?
   - Is the benefit real (not technical)?
   - Can it be verified?

3. **Ask clarification questions** if the story is ambiguous or incomplete (maximum 3):

   - "Who is the user that needs this?"
   - "What concrete outcome do they expect?"
   - "How would you know this is correctly implemented?"

4. **Deliver the refined story** in professional format:

```
**As a** [user persona],
**I want** [action],
**so that** [benefit].

**Acceptance Criteria**:
- [ ] Given [context], when [action], then [result]
- [ ] Given [context], when [action], then [result]
- [ ] Given [context], when [action], then [result]

**Notes**: _(additional context if applicable)_
```

5. **Briefly explain the changes**: what was improved and why.

### Standalone Mode Rules

- **Maximum 3 clarification questions.** If the story is still ambiguous after 3 questions, refine it as best you can and mark remaining open areas.
- **Do not judge the idea.** Refinement improves expression, it does not question the feature's value.
- **Always deliver acceptance criteria in Given/When/Then format.** A story without verifiable AC is not refined.
- **At least 3 acceptance criteria per story.**
- **No code snippets, no architecture decisions, no tech stack references.**

---

## Key Principles

**Do:**
- Audit every user story against all six INVEST criteria.
- Use Given/When/Then format for all acceptance criteria.
- Flag problems with concrete examples and suggested fixes.
- Ask before modifying any file — never auto-apply changes.
- Stop asking questions when the story is clear enough to refine.

**Don't:**
- Never add new stories or delete existing ones.
- Never change the scope or intent of a story.
- Never touch sections of the PRD outside User Stories (audit mode).
- Never use vague language in acceptance criteria.
- Never create a new file for revisions — update the existing PRD in place.

---

## Anti-Patterns

- **Technical stories without user value** — "Refactor the caching layer" is a task, not a story.
- **Epics disguised as stories** — too large, too many acceptance criteria.
- **Vague acceptance criteria** — "properly handles", "works correctly", "is fast".
- **Implementation prescribed in the story** — "Use Redis with TTL of 300s".
- **Circular dependencies between stories** — Story A needs Story B which needs Story A.

---

## Verification Checklist

- [ ] Story traces to a PRD requirement
- [ ] Each INVEST criterion passes
- [ ] Acceptance criteria use Given/When/Then format
- [ ] No vague words in acceptance criteria ("properly", "correctly", "fast")
- [ ] Dependencies are one-way only (no circular dependencies)
- [ ] At least 3 acceptance criteria per story
- [ ] No code snippets, architecture decisions, or tech stack references
- [ ] In audit mode: only the User Stories section of the PRD was touched
- [ ] In audit mode: user confirmation was obtained before modifying the PRD
- [ ] In standalone mode: no more than 3 clarification questions were asked
