---
name: doc-rec-lite
description: 'Gather requirements for a single legacy feature using Volere-lite. Trigger: lite requirements, single-feature requirements, legacy feature rec, invoked by doc-feat.'
author: Zesh-One
license: MIT
---

# Requirements Elicitation — Feature Scope (Lite)

Invoked ONLY by `doc-feat`. Not user-invokable directly.

## Activation Contract

Inputs from `doc-feat`:
- `<sistema>` — confirmed system name
- `<slug>` — confirmed feature slug
- `<descripcion>` — raw feature description
- `(mode, anchor_or_pattern)` — resolved scope from `doc-scope` or `--scope` flag

## Hard Rules

| Rule | Detail |
|------|--------|
| Single-feature scope | Capture requirements ONLY for `<descripcion>`. Never for the host system as a whole. |
| No BABOK multi-session | No stakeholder planning, no workshop scheduling, no iterative rounds. One focused session. |
| No host-system requirements | Do not produce requirements for parent modules, adjacent services, or the overall product. |
| Bounded discovery only | Use `(mode, anchor_or_pattern)` to limit codebase reads. Do not scan beyond what `doc-scope` permitted. |
| Progressive depth | Start with the business need. Deepen to technical detail only after context is established. |

## Elicitation Approach

Use the three progressive layers from `doc-rec` (the parent skill), but scoped to this feature only:

- **Layer 1 — Business need**: Why does this feature matter? What problem does it solve for users?
- **Layer 2 — Current behavior**: How is this handled today (even if manually or incompletely)?
- **Layer 3 — Solution detail**: What data, rules, edge cases, and constraints apply to THIS feature?

Do not open with technical questions before business context is established.

## Volere-Lite Schema

Capture each requirement using the compact Volere Snow Card format from `doc-rec`:

```
REQ-<n>: <short title>
Type: [Functional | Non-Functional | Constraint | Transition]
Description: <what must be true>
Rationale: <why this is needed>
Source: <who said it or where it came from>
Fit criterion: <how you will know this requirement is met>
```

Reference the full elicitation technique guidance in `skills/doc-rec/references/elicitation-techniques.md` if deeper technique selection is needed.

## Output Contract

Write `<BASE_PATH><sistema>-features/<slug>/<slug>_requirements.md`.

The file MUST contain:
- A brief scope statement (1-2 sentences confirming this is feature-scoped)
- All captured requirements in Volere-lite format
- Any assumptions, constraints, and open decisions noted separately
- Source attribution on every requirement
