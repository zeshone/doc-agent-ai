---
name: doc-to-sdd
description: 'Compact documentation into LLM-optimized SDD context files. Trigger: /to-sdd, sdd context, compact docs.'
author: Zesh-One
license: MIT
---

# SDD Context Compactor

## Trigger / Positioning

**Triggers:** `/doc-to-sdd`, `doc-to-sdd`, `sdd context`, `compact docs`, `to-sdd <system>`

**Position:** Standalone command — NOT part of the `/doc-arch` flow sequence. Can run after any combination of `idea`, `rec`, `prd`, `tech`, `ddd` phases have completed.

**Direct routing:** `/doc-to-sdd` invokes the `doc-to-sdd` sub-agent directly. No orchestrator intermediary.

## Activation Contract

| Input | Required | Source |
|-------|----------|--------|
| `<system>` | Yes | User invocation: `/doc-to-sdd <system>` |
| `<system>_idea-brief.md` | No (fallback: `<system>.md` first paragraph) | `doc-idea` output |
| `<system>_requirements.md` | No (minimum: one of rec/prd for business layer) | `doc-rec` output |
| `<system>_prd.md` | No (minimum: one of rec/prd for business layer) | `doc-prd` output |
| `<system>_tech-spec.md` | No (minimum: one of tech/ddd for technical layer) | `doc-tech` output |
| `<system>_db-design.md` | No (minimum: one of tech/ddd for technical layer) | `doc-ddd` output |
| `BASE_PATH` | Yes | Platform environment |

**Source artifact suffixes:**

| Suffix | Layer | Producer |
|--------|-------|----------|
| `_idea-brief.md` | Business | `doc-idea` |
| `_requirements.md` | Business | `doc-rec` |
| `_prd.md` | Business | `doc-prd` |
| `_tech-spec.md` | Technical | `doc-tech` |
| `_db-design.md` | Technical | `doc-ddd` |

**Discovery method:** Direct path construction using `<system>` argument + known suffixes. No glob needed — file naming is deterministic. Check existence with file-read; if not found, mark as missing and continue.

**Idea extraction heuristic:** Read `_idea-brief.md` first. If absent, extract the first paragraph from `<system>.md` after the H1 title and before the first `##` header. If no `##` header found, take the first 3 non-empty lines after the title.

## Exit Contract

**Output files** (written to `agent_sdd_context_project/` under the resolved docs root — `<BASE_PATH><system>/` in vault mode, `docs/doc-agent/` in in-project mode):

| File | Layer | Sources |
|------|-------|---------|
| SDD context | Business | idea + rec + prd |
| SDD tech context | Technical | tech + ddd |

**Naming is mode-aware:**

| Mode | Scope | Business file | Technical file |
|------|-------|----------------|-----------------|
| Vault | System-level (`to-sdd <system>`) | `<system>_sdd-context.md` | `<system>_sdd-tech-context.md` |
| Vault | Feature/module-level (`<module>` = doc-arch node short name) | `<system>_<module>_sdd-context.md` | `<system>_<module>_sdd-tech-context.md` |
| In-project | Any | `_sdd-context.md` (unchanged, bare) | `_sdd-tech-context.md` (unchanged, bare) |

Feature/module-level naming is a forward-compatible convention — the rule applies regardless of whether module-scoped invocation is currently exercised.

**Index update:** Add or replace `## SDD Context` section in `<BASE_PATH><system>/<system>.md` (vault) or the project index (in-project) with the mode-appropriate block, using the same filenames from the naming table above:

Vault, system-level:

```markdown
## SDD Context

LLM-optimized context files for agentic SDD programming:
- `agent_sdd_context_project/<system>_sdd-context.md` — Business layer (problem, requirements, stories, decisions)
- `agent_sdd_context_project/<system>_sdd-tech-context.md` — Technical layer (architecture, stack, contracts, data model)
```

Vault, feature/module-level:

```markdown
## SDD Context

LLM-optimized context files for agentic SDD programming:
- `agent_sdd_context_project/<system>_<module>_sdd-context.md` — Business layer (problem, requirements, stories, decisions)
- `agent_sdd_context_project/<system>_<module>_sdd-tech-context.md` — Technical layer (architecture, stack, contracts, data model)
```

In-project (bare, unchanged):

```markdown
## SDD Context

LLM-optimized context files for agentic SDD programming:
- `agent_sdd_context_project/_sdd-context.md` — Business layer (problem, requirements, stories, decisions)
- `agent_sdd_context_project/_sdd-tech-context.md` — Technical layer (architecture, stack, contracts, data model)
```

If a `## SDD Context` section already exists, replace it entirely.

**All output MUST be in English** regardless of source artifact language.

## Compaction Procedure

**NON-NEGOTIABLE — MAXIMUM TOKEN EFFICIENCY:** Every sentence must carry essential information. Eliminate redundancy, filler words, preamble, and narrative framing. Write as if every token costs money. If a sentence does not contribute to understanding the WHAT, WHY, or HOW, delete it.

**NON-NEGOTIABLE — CLARITY > TOKEN DENSITY:** If a concept requires multiple sentences to be precise and unambiguous, those sentences MUST be kept. Precision and correctness are never sacrificed. In a conflict between saving tokens and being understood, clarity wins every time.

### Pipeline

1. **Discover** source artifacts using resolution order per layer (see Activation Contract).
2. **Parse** each artifact into sections at H1/H2/H3 boundaries.
3. **Classify** each section against KEEP/DROP criteria.
4. **Extract** and restructure KEEP sections into target output schema.
5. **Write** output files to `agent_sdd_context_project/`.
6. **Update** project index (`<system>.md`) with SDD Context reference block.

### KEEP Criteria

Retain: core problem statements, key decisions + rationale, architecture choices, interface/contract definitions, constraints, user stories with acceptance criteria, TBDs and open questions, data models, schema definitions, non-functional requirements.

### DROP Criteria

Discard: narrative introductions, repeated context, verbose examples (keep minimal essential only), formatting flair, table-of-contents, changelog sections, "generated by" metadata, redundant restatements.

### TBD Preservation Rule

**TBDs and open questions MUST be preserved verbatim — never invent resolutions.** If a source artifact marks a decision as TBD or lists an open question, copy it exactly as written. Do not substitute assumptions, guesses, or placeholder answers. The SDD agent must know what is genuinely unresolved.

### Merge Rule

When the same topic appears in multiple source artifacts (e.g., decisions in both PRD and tech spec), merge into a single section preserving all unique entries. Flag contradictions with `⚠️ CONFLICT:` prefix.

## Output Format Specification

**Density benchmark:** ~50-150 lines per file. This is a GUIDELINE — clarity is the only hard boundary. Do NOT exceed 200 lines per file without justification.

### `_sdd-context.md` (Business Layer)

```markdown
# {system} — SDD Context (Business)

## Problem Statement
{Core problem from idea + rec, 2-4 sentences max}

## Requirements Summary
| ID | Requirement | Priority | Source |
|----|-------------|----------|--------|

## User Stories
### US-{n}: {title}
- **As a** {role}, **I want** {goal}, **so that** {value}
- **Acceptance criteria:** {Given/When/Then from PRD}

## Constraints and Non-Goals
| Type | Description |
|------|-------------|

## Decisions Log
| Decision | Choice | Rationale |
|----------|--------|-----------|

## Open Questions / TBDs
- [ ] {item}
```

### `_sdd-tech-context.md` (Technical Layer)

```markdown
# {system} — SDD Context (Technical)

## Architecture Overview
{Architecture style, key components, 3-5 sentences max}

## Technology Stack
| Layer | Technology | Rationale |
|-------|-----------|-----------|

## Interface Contracts
### {Interface Name}
- **Type:** {REST/GraphQL/gRPC/event}
- **Endpoints/Operations:** {list}
- **Key contracts:** {request/response shapes or references}

## Data Model
{ERD reference or inline summary, key entities + relationships}

## Architecture Decisions
| Decision | Choice | Alternatives | Rationale |
|----------|--------|-------------|-----------|

## Constraints and NFRs
| Category | Constraint | Mechanism |
|----------|-----------|-----------|

## Open Decisions / TBDs
- [ ] {item}
```

## Partial Availability Rules

| Available Sources | Output | Missing Warning |
|-------------------|--------|-----------------|
| idea + rec + prd + tech + ddd | Both files | None |
| rec + prd (no idea) | `_sdd-context.md` only | "No idea artifact found; business context sourced from rec+prd only." |
| idea + rec (no prd) | `_sdd-context.md` with reduced stories | "No PRD found; user stories sourced from requirements only. Acceptance criteria may be incomplete." |
| tech + ddd | `_sdd-tech-context.md` only | "No business-layer artifacts found; technical context generated independently." |
| tech only (no ddd) | `_sdd-tech-context.md` without data model section | "No DB design artifact found; data model section omitted." |
| ddd only (no tech) | `_sdd-tech-context.md` with data model only | "No tech spec found; architecture and interfaces sections omitted." |
| Only idea | STOP | "Insufficient source artifacts. Minimum: rec OR prd for business context, OR tech OR ddd for technical context." |
| None | STOP | "No source artifacts found. Run the documentation workflow first." |

**Minimum viable set:** At least one of {rec, prd} for business file, OR at least one of {tech, ddd} for technical file.

## Error Table

| Condition | Action |
|-----------|--------|
| System directory does not exist | STOP: "System `<system>` does not exist. Start with `/doc-rec <system>` first." |
| Source artifact file is empty (0 bytes) | Treat as missing; emit warning: "`<artifact>` is empty — skipped." |
| Source artifact unreadable (permissions) | STOP: "Cannot read `<artifact>`. Check file permissions." |
| Source artifact has no recognizable markdown headers | Warn: "`<artifact>` has no parseable structure — extracting raw content." Attempt best-effort extraction. |
| Output directory cannot be created | STOP: "Cannot create `agent_sdd_context_project/`. Check disk space and permissions." |
| All source artifacts missing for a layer | Do not create that layer's output file. Report which artifacts are missing. |
| Both layers have no sources | STOP: "No source artifacts found. Run the documentation workflow first." |

## Anti-Patterns

| Forbidden | Why |
|-----------|-----|
| Inventing missing information | Fabricated context misleads SDD agents; flag gaps instead |
| Summarizing decisions — preserve verbatim rationale | Summaries lose nuance; SDD agents need exact reasoning |
| Merging contradictory statements without flagging | Contradictions signal unresolved design tension; mark with `⚠️ CONFLICT:` |
| Exceeding 200 lines per file without justification | Density is the goal; bloat defeats the purpose of compaction |
