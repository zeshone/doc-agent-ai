---
name: doc-prd
description: 'Generate high-quality Product Requirements Documents (PRDs) for software systems and AI-powered features. Includes executive summaries, user flows, user stories, technical constraints, integrations, and risk analysis. Trigger: PRD, product requirements, product spec.'
author: Zesh-One
license: MIT
---

# Product Requirements Document (PRD)

## Positioning

Sits **between** `rec` (requirements elicitation) and `tech` (technical specification).

- More technical than `rec`: defines flows, stories, acceptance criteria, dependencies, constraints, measurable success KPIs, rollout/risk.
- Less implementation-specific than `tech`: guides PMs, tech leads, developers without collapsing into code-level design.
- Clear and precise. Explain through behavior, constraints, and impact — not buzzwords.

## When to Use

- Starting a new product or feature cycle.
- Translating validated requirements into product scope.
- Defining flows, stories, criteria, and measurable outcomes.
- Capturing integrations, restrictions, dependencies, and rollout before the tech spec.

## Operational Workflow

### Phase 1: Discovery — Ask Before Drafting

**Minimum: 2 clarification questions** before writing anything.

If this is a module: before asking discovery questions, read the parent system's `_prd.md`. Align scope, Non-Goals, and constraints with the parent.

Ask about missing areas: core problem, success metrics, primary flows, scope boundaries, dependencies/integrations, constraints (stack, budget, timeline, SLA/SLO, security, compliance, regulatory).

**Unknown handling:** if the answer is required for a reliable PRD → ask. If legitimately open after clarification → mark as `TBD` with a note. Never invent stack, SLA, metrics, security posture, or integration assumptions.

### Phase 2: Analysis & Scoping

- Map primary and exception user flows.
- Convert needs into user stories with measurable acceptance criteria.
- Identify dependencies, restrictions, assumptions, and open decisions.
- Define **Non-Goals** to protect scope.
- For modules: align with parent system PRD so the module does not contradict system Non-Goals.

### Phase 3: Drafting

Generate using the **Strict PRD Schema** below. The draft must be:
- More technical than `rec`, clearer than a raw engineering note.
- Precise without cryptic jargon. Explain through behavior, constraints, or impact.

## Quality Standards

### Measurable Criteria

Avoid "fast", "easy", "intuitive" unless immediately quantified.

| Vague (BAD) | Concrete (GOOD) |
|-------------|-----------------|
| "Search should be fast" | "Search returns first-page results within 200ms for 10k records" |
| "UI must be easy to use" | "New users complete primary flow in ≤3 min without assistance" |

### TBDs and Open Decisions

If information is incomplete: ask first, or document as `TBD` with a note explaining what is unknown and why. Example: `**SLA:** TBD — no uptime target provided by stakeholder team.`

### Audience

Write for mixed audiences: teams needing stronger docs + experienced PMs/tech leads who expect precision. Explain critical terms, prefer direct sentences, be technically accurate without being unreadable.

## Strict PRD Schema

### 1. Executive Summary
- **Problem Statement** (1-2 sentences)
- **Proposed Solution** (1-2 sentences)
- **Success Criteria** (3-5 measurable KPIs)

### 2. User Experience & Functionality
- **User Personas**
- **Primary User Flows** (main + exception paths)
- **User Stories** (`As a [user], I want [action] so that [benefit]`)
- **Acceptance Criteria** (bulleted "Done" definitions per story)
- **Non-Goals** (explicitly excluded)

### 3. AI System Requirements *(if applicable)*
- **Tool Requirements**
- **Evaluation Strategy**

### 4. Technical Specifications
- **Architecture Overview**
- **Dependencies & Integration Points**
- **Technical Constraints**
- **Security & Privacy**

### 5. Risks & Roadmap
- **Phased Rollout** (MVP → v1.1 → v2.0)
- **Technical Risks**
- **Open Decisions / TBDs** (must remain visible, not buried)

## DO / DON'T

**DO:**
- Ask before assuming. If a critical fact is missing, clarify it.
- Define measurable criteria: success, acceptance, operational.
- Name open decisions: visible as `TBD`, not buried.
- Iterate: present draft, ask for feedback.

**DON'T:**
- Skip discovery. Never write a PRD without asking ≥2 clarification questions.
- Hallucinate constraints. Unknown → ask or mark `TBD`.
- Hide ambiguity behind jargon. If it sounds impressive but doesn't explain behavior → rewrite.
- Contradict parent context. Module PRDs must not conflict with parent scope or Non-Goals.
