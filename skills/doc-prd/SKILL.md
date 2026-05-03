---
name: doc-prd
description: 'Generate high-quality Product Requirements Documents (PRDs) for software systems and AI-powered features. Includes executive summaries, user flows, user stories, technical constraints, integrations, and risk analysis.'
author: Zesh-One
license: MIT
---

# Product Requirements Document (PRD)

## Positioning

The PRD sits **between** requirements elicitation (`rec`) and the technical specification (`tech`).

- It is **more technical than elicitation**: it must define flows, stories, acceptance criteria, dependencies, integrations, constraints, measurable success criteria, and rollout/risk considerations.
- It is **less implementation-specific than a tech spec**: it should guide PMs, tech leads, senior developers, and documentation novices without collapsing into code-level design.
- It must stay **clear and pedagogical**. Use precise language, not empty jargon.

## Overview

Design comprehensive, production-grade Product Requirements Documents (PRDs) that bridge business intent and technical execution. A strong PRD makes scope, behavior, constraints, risks, and open decisions explicit.

This skill is for mixed audiences: small/medium teams, freelancers with low documentation maturity, PMs, and expert engineering leads. Write so that a novice can follow the reasoning and an expert can trust the precision.

## When to Use

Use this skill when:

- Starting a new product or feature development cycle
- Translating validated requirements into product scope and delivery direction
- Defining user flows, stories, acceptance criteria, and measurable outcomes
- Capturing integrations, restrictions, dependencies, and rollout concerns before the tech spec
- Stakeholders need a single PRD that works for both product and engineering conversations

---

## Operational Workflow

### Phase 1: Discovery (Clarify Before Drafting)

Before writing a single line of the PRD, you **MUST** ask clarifying questions to close critical gaps. **Never assume missing context**, even if it seems obvious.

**Minimum rule:** ask at least **2 clarifying questions** before generating.

**Ask about missing or subjective areas such as:**

- **Core problem**: Why are we building this now?
- **Success metrics**: How do we know this worked?
- **Primary flows**: What must happen step by step?
- **Scope boundaries**: What is explicitly in scope vs. out of scope?
- **Dependencies / integrations**: Which systems, APIs, teams, or vendors are involved?
- **Constraints**: Stack, budget, timeline, rollout, migration, SLA/SLO, performance, security, privacy, compliance, regulatory requirements

**Unknown handling:**

- If the answer is required to produce a reliable PRD, **ask**.
- If the decision remains legitimately open after clarification, mark it as **`TBD`** in the relevant section with a clear note.
- Never invent stack, SLA, metrics, security posture, compliance obligations, rollout strategy, or integration assumptions.

### Phase 2: Analysis & Scoping

Synthesize the validated inputs and turn them into product scope that engineering can actually execute.

- Map **primary and exception user flows**
- Convert needs into **user stories with measurable acceptance criteria**
- Identify **dependencies, restrictions, assumptions, and open decisions**
- Define **Non-Goals** to protect timeline and scope
- If working on a module, align with the parent system PRD so the module does not contradict system-level Non-Goals

### Phase 3: Technical Drafting

Generate the document using the **Strict PRD Schema** below.

The draft must be:

- **More technical than `rec`**
- **Clearer than a raw engineering note**
- **Precise without sounding cryptic**

If you use technical language, explain it through behavior, constraints, or impact — not buzzwords.

---

## PRD Quality Standards

### 1. Technical Depth With Clarity

The PRD must describe:

- user flows
- user stories
- acceptance criteria
- dependencies and integrations
- technical and operational restrictions
- risks and rollout considerations
- measurable success criteria

But it must do so with **clear, teachable language**. Avoid needless acronyms, vague adjectives, and architecture theater.

### 2. Requirements Quality

Use concrete, measurable criteria. Avoid words like "fast", "easy", or "intuitive" unless they are immediately quantified.

```diff
# Vague (BAD)
- The search should be fast and return relevant results.
- The UI must be easy to use.

# Concrete (GOOD)
+ The search must return first-page results within 200ms for a dataset of 10k records.
+ The search experience must achieve >= 85% Precision@10 in benchmark evaluations.
+ New users must complete the primary flow in <= 3 minutes without facilitator assistance.
```

### 3. Unknowns and Open Decisions

If information is incomplete, subjective, or still under discussion:

- **Ask first** if the answer is needed now
- Otherwise document it as **`TBD`**
- Add a short note explaining **what is unknown** and **why it remains open**

Example:

```markdown
**SLA / Availability:** TBD — no uptime target was provided yet by the stakeholder team.
```

### 4. Audience Fit

Write for both:

- teams that need stronger documentation structure
- experienced PMs / tech leads who expect precision

That means:

- explain critical terms when they matter
- prefer direct sentences over dense prose
- be technically accurate without becoming unreadable

---

## Strict PRD Schema

You **MUST** follow this exact top-level structure for the output:

### 1. Executive Summary

- **Problem Statement**: 1-2 sentences on the pain point.
- **Proposed Solution**: 1-2 sentences on the fix.
- **Success Criteria**: 3-5 measurable KPIs.

### 2. User Experience & Functionality

- **User Personas**: Who is this for?
- **Primary User Flows**: Main and exception paths that matter for delivery.
- **User Stories**: `As a [user], I want to [action] so that [benefit].`
- **Acceptance Criteria**: Bulleted list of "Done" definitions for each story.
- **Non-Goals**: What are we NOT building?

### 3. AI System Requirements (If Applicable)

- **Tool Requirements**: What tools and APIs are needed?
- **Evaluation Strategy**: How to measure output quality and accuracy.

### 4. Technical Specifications

- **Architecture Overview**: Data flow and component interaction.
- **Dependencies & Integration Points**: APIs, DBs, Auth, third-party services, upstream/downstream systems.
- **Technical Constraints**: Stack restrictions, performance targets, rollout limits, or operational boundaries.
- **Security & Privacy**: Data handling, access control, auditability, and compliance notes.

### 5. Risks & Roadmap

- **Phased Rollout**: MVP -> v1.1 -> v2.0.
- **Technical Risks**: Latency, cost, dependency failures, adoption risks, data quality issues, etc.
- **Open Decisions / TBDs**: Explicit unresolved items that must not be silently assumed.

---

## Implementation Guidelines

### DO (Always)

- **Ask before assuming**: if a critical fact is missing, clarify it.
- **Use refined questions**: more precise than elicitation, but still easy to understand.
- **Define measurable criteria**: success, acceptance, and operational expectations should be concrete.
- **Name open decisions**: unresolved items must be visible as `TBD`, not buried.
- **Iterate**: present a draft and ask for feedback on specific sections.

### DON'T (Avoid)

- **Skip discovery**: never write a PRD without asking at least 2 clarifying questions first.
- **Hallucinate constraints**: if stack, SLA, metrics, compliance, security model, or rollout are unknown, ask or mark `TBD`.
- **Hide ambiguity behind jargon**: if a sentence sounds impressive but does not explain behavior or constraints, rewrite it.
- **Contradict parent context**: when documenting a module, do not conflict with the parent PRD's scope or Non-Goals.

---

## Example: Intelligent Search System

### 1. Executive Summary

**Problem**: Users struggle to find specific documentation snippets in large repositories without guessing the right keywords.

**Solution**: An intelligent search system that answers natural language queries with source-linked results.

**Success Criteria**:

- Reduce median search time by 50%.
- Maintain citation accuracy >= 95% in benchmark evaluations.
- Keep first response latency <= 2 seconds for benchmark queries.

### 2. User Experience & Functionality

- **Primary Flow**: User asks a question, reviews ranked cited answers, and opens the referenced source.
- **Story**: As a developer, I want to ask natural language questions so I don't have to guess keywords.
- **AC**:
  - Supports clarification when the question is ambiguous.
  - Returns citations for every generated answer.

### 4. Technical Specifications

- **Dependencies & Integration Points**: Repository indexer, authentication layer, and source citation service.
- **Technical Constraints**: Repository indexing cadence is TBD pending infrastructure decision.

### 5. Risks & Roadmap

- **Open Decision / TBD**: Hosting model for embeddings remains TBD until cost envelope is approved.
