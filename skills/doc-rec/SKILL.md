---
name: doc-rec
description: >
  Facilitate structured requirements gathering. Use when the user says "gather requirements",
  "elicitation session", "stakeholder interview", "requirements workshop", "what do the users need",
  "discover requirements", "trawl for requirements", "understand the business need",
  "conduct interviews", or needs to systematically extract requirements from stakeholders -
  even if they don't explicitly say "elicitation".
author: Zesh-One
license: MIT
---

## Reference Files

- `references/elicitation-techniques.md` - Lookup table of 15+ elicitation techniques with when to use, participants, output format, and time needed. Read this in Step 2 when selecting techniques and in Step 3 when planning sessions.

## Overview

Method: BABOK Elicitation & Collaboration. Start from business events and stakeholders, not a blank template. Elicitation is iterative — each round reveals gaps for the next. Keep methodological rigor internal; keep the stakeholder-facing conversation in business language, progressing to technical depth only when context is established.

**Conversational rule:** keep the methodological rigor internally, but make the interface progressively technical. Start in executive/business language, then deepen only when context is strong enough. Do **not** open with BABOK jargon, formal classification labels, solution design, implementation details, integrations, or technical architecture questions unless the stakeholder already speaks naturally at that level.

## Workflow

### Step 1: Identify stakeholders and business events

If this is a module: first read the parent system's `_requirements.md`. Module requirements must complement, not duplicate, the parent baseline.

List every person, role, and adjacent system that touches the problem space. Use BABOK stakeholder categories: customer, end user, sponsor, domain expert, regulator, implementation team. Then list the business events that trigger the process or system under analysis.

### Step 2: Select elicitation techniques
Match techniques to stakeholder type and information need (see `references/elicitation-techniques.md`). Use a mix:
- **Interviews** for deep individual knowledge
- **Workshops** for cross-functional alignment and conflict surfacing
- **Observation/Apprenticing** (Robertson) for tacit knowledge users cannot articulate
- **Document Analysis** for existing rules, policies, and constraints
- **Prototyping** for validating understanding with users

### Step 3: Plan and schedule sessions
For each session, define:
```
SESSION: [type - interview / workshop / observation]
STAKEHOLDER(S): [names and roles]
OBJECTIVE: [what you need to learn]
TECHNIQUE: [from reference table]
DURATION: [estimated time]
PREPARATION: [materials, context docs, pre-reads]
```

When planning, also define the **depth level** expected for the session:
- **Initial / executive** - problem, goals, actors, pain, impact, visible constraints
- **Intermediate / operational** - current process, events, rules, exceptions, dependencies
- **Advanced / solution detail** - data, integrations, NFRs, transition, implementation constraints

### Step 4: Conduct elicitation
Conduct the conversation in **three progressive layers**:

#### Layer 1 - Executive / high-level start
Use plain business language. Focus on the problem space before structure, taxonomy, or solution shape.

Goal:
- Understand why this matters now
- Clarify business outcomes, impacted actors, and pain points
- Surface success criteria and business-visible constraints

Ask questions like:
- "What problem are you trying to solve, or what opportunity are you trying to capture?"
- "What is happening today that makes it necessary to move on this now?"
- "Who is most affected by this problem, and in what way?"
- "How is this handled today, even if the process is manual or incomplete?"
- "If this goes well, what would change for the business or for users?"
- "How would you know in 3 or 6 months that this was successful?"
- "Are there any visible constraints around time, budget, regulation, or risk that we should already take into account?"

Do **not** start this layer by asking for:
- BABOK categories or formal requirement classes
- APIs, integrations, schemas, entities, or architecture
- implementation details, stack, or delivery mechanics

#### Layer 2 - Operational / process understanding
Once the problem and outcomes are clear, move into how the business works today.

Focus on:
- Current process and workflow steps
- Business events and triggers
- Business rules and decision criteria
- Exceptions, failure paths, and dependencies between areas

Example probes:
- "What event triggers this process?"
- "What needs to happen first for this to move forward?"
- "Which business rules cannot be broken?"
- "What edge cases or exceptions create the most friction for you?"

#### Layer 3 - Advanced / solution detail
Only after Layers 1 and 2 are reasonably understood, move to solution-shaping questions.

Focus on:
- Solution boundaries and expectations
- Integrations and external dependencies
- Data inputs/outputs and ownership
- Non-functional requirements
- Transition requirements, rollout, migration, enablement

Example probes:
- "What systems or teams would need to interact with this solution?"
- "What data comes in, who maintains it, and what output do you expect to get?"
- "Are there requirements for security, auditability, availability, or performance?"
- "What would need to happen to move from the current state to the new one without disrupting operations?"

Throughout all layers:
- Start with open-ended context questions and then move to sharper probes
- Capture requirements using Robertson's Volere Snow Card format: requirement number, type, description, rationale, source, fit criterion
- Note assumptions, constraints, and conflicts separately
- Record who said what - source traceability starts here
- Keep the conversation de-technicalized at the start even if your internal notes remain rigorous

### Step 5: Confirm and validate
After each session:
- Write up findings within 24 hours
- Send back to stakeholders for confirmation (BABOK's "confirm elicitation results")
- Flag conflicts between stakeholders - do not resolve silently
- Identify gaps that need follow-up sessions

### Step 6: Organize and classify requirements
Classify using BABOK's hierarchy:
- **Business requirements** - why the organization needs this
- **Stakeholder requirements** - what users need to do
- **Solution requirements** - functional + non-functional
- **Transition requirements** - what's needed to move from current to future state

## Anti-Patterns

**1. Template-first elicitation**
Bad: Sending a requirements template to stakeholders and asking them to fill it in.
Good: Conduct interviews and workshops first, then organize findings into a structured format.

**2. Single-technique reliance**
Bad: Only conducting interviews, missing tacit knowledge that observation would reveal.
Good: Use at least 3 techniques per project - interviews for depth, workshops for alignment, observation for reality.

**3. Eliciting solutions instead of needs**
Bad: "The user wants a dropdown menu." (That's a solution.)
Good: "The user needs to select from a constrained set of valid values." (That's a requirement.)

**4. No source traceability**
Bad: Requirements listed without noting who requested them or why.
Good: Every requirement links back to a stakeholder, business event, or document source.

**5. One-pass elicitation**
Bad: Running a single round of interviews and declaring requirements "complete."
Good: Iterative trawling - each round reveals gaps that drive the next round.

**6. Technical cold open**
Bad: Opening with requirement taxonomies, system integrations, NFR checklists, or implementation questions before the stakeholder has framed the business problem.
Good: Start with executive language, establish outcomes and pain points, then increase technical depth progressively.

## Quality Checklist

- [ ] All stakeholder categories identified (users, sponsors, domain experts, regulators)
- [ ] Business events listed as elicitation drivers
- [ ] At least 3 elicitation techniques selected
- [ ] Conversation started in business language before technical depth increased
- [ ] Each session has a defined objective and preparation
- [ ] Requirements captured with source attribution
- [ ] Findings confirmed with stakeholders
- [ ] Conflicts between stakeholders flagged (not silently resolved)
- [ ] Requirements classified by BABOK hierarchy
- [ ] Gaps identified with follow-up sessions planned
