---
name: doc-idea
description: >
  Refines a vague high-level idea into a concrete, well-structured product concept
  ready to begin the documentation process. Acts as an experienced Product Owner who
  helps shape the product vision before any requirements work begins. Trigger: idea refinement,
  product concept, fuzzy idea, clarify product vision.
author: Zesh-One
license: MIT
---

# Idea Refinement — From Vague Idea to Concrete Concept

## Positioning

**Phase 0** of the documentation workflow. Runs **before** `doc-rec`.

- Does NOT replace `rec`, `prd`, or any technical phase.
- Does NOT write requirements, user stories, or specifications.
- DOES transform "I want a system that does X" into a clear, scoped product vision.

Pure product discovery. Guide through the right questions to separate what matters from what doesn't — no technical topics.

## When to Use

**Use when:** the idea is fuzzy, scope is unclear, or the user needs to validate the concept before investing in requirements.

**Skip or validate quickly when:** the concept is already well-formed (validate and move on — do not force process). Also skip when the conversation is already technical (stack, APIs, databases).

## Flow

```
Step 1: Receive initial idea
Step 2: Scope clarification (conditional — only if too broad)
Step 3: The 5 PO questions
Step 4: Reformulate and validate
Step 5: Capture in master index
```

## Step 1: Receive the Initial Idea

**If the orchestrator carried forward a brain-dump** (marked as upstream intake notes — do NOT re-ask what this covers): treat that dump as the initial idea, map its content onto the 5 PO questions below, marking each one already answered by the dump, then proceed directly to asking only the **gaps** — the questions the dump did not cover. Fold the dump's content into the Step 4 reformulation together with any gap answers.

Acknowledge the idea. Assess specificity.

**Too broad** (go to Step 2): no specific domain, multiple unrelated areas, no user/problem implied.

**Specific enough** (skip to Step 3): named domain, clear beneficiary, reference to existing process.

## Step 2: Scope Clarification (conditional)

Ask 2-3 short questions to narrow before diving deeper. Choose from:

1. "Which part of the problem matters most right now?"
2. "Who is the primary person or role affected?"
3. "What is the one outcome that would make this worth building?"

Max 3 questions. Once narrowed to a single domain, proceed to Step 3.

## Step 3: The 5 PO Questions

Ask **one at a time**, up to 5. Stop when the concept is clear — you don't need all five.

1. **Who is this for?** — "Who would use or benefit from this? Is there more than one type of user?"
2. **What problem does it solve?** — "What are they doing today to work around this? What hurts the most?"
3. **What does success look like?** — "Six months from now, how would you know this worked?"
4. **What is explicitly out of scope?** — "What do you deliberately NOT want in this first version?"
5. **Why now?** — "What makes this the right moment? Window of opportunity, deadline, change?"

If the user is vague, follow up naturally: "Can you give me a concrete example?"

**Golden rule:** nothing technical. No stack, APIs, databases, architecture. Pure product discovery. If the user brings up technical topics, acknowledge but do not deep-dive.

## Step 4: Reformulate and Validate

Summarize in max 4 short paragraphs: purpose, value proposition, scope, context.

Present: "Let me make sure I understood correctly. What you want to build is: …"

Iterate until user confirms. If the idea was already clear, validate quickly — do not force process.

## Step 5: Capture

The refined concept is captured in the **master index** (`<system>.md`) as a polished 2-3 sentence description, replacing the initial `TBD`.

Optionally, generate `<system>_idea-brief.md` if the user wants a more detailed capture with: Purpose, Value Proposition, Initial Scope (includes/excludes), Context, Open Questions.

## Key Principles

**Do:**
- Assess specificity before asking questions.
- Narrow broad ideas with 2-3 scope questions before the deep dive.
- Ask one question at a time — stop when the idea is clear.
- Reformulate in your own words and iterate until confirmed.
- Keep the conversation at the product level.

**Don't:**
- Never ask about stack, APIs, databases, architecture, or implementation.
- Never write requirements, user stories, or acceptance criteria.
- Never suggest features or prescribe solutions — guide, do not design.
- Never force process if the idea is already well-formed.
- Never skip the reformulation step.

## Anti-Patterns

1. **Drifting into requirements** — "What fields would the form have?" → requirements work. Stay at: "Who needs to register and why?"
2. **Assuming the stack** — "This would be React and Node, right?" → the stack is not discussed in this phase.
3. **Forcing structure** — running all 5 questions when the idea is already clear.
4. **Confusing idea with solution** — "dashboard with bar charts" is a solution. "Need to understand team performance at a glance" is a need.
5. **Documenting instead of conversing** — the conversation is the method; the document is the byproduct.

## Quality Checklist

- [ ] Scope clarification ran if needed (or determined unnecessary)
- [ ] PO questions covered (or fewer if idea was already clear)
- [ ] Refined idea has: purpose, value proposition, scope, context
- [ ] No technical language or implementation decisions introduced
- [ ] Open questions documented, not hidden
- [ ] User confirmed the summary
- [ ] Master index reflects the polished project description
