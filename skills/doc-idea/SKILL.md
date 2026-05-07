---
name: doc-idea
description: >
  Refines a vague high-level idea into a concrete, well-structured product concept
  ready to begin the documentation process. Acts as an experienced Product Owner who
  helps shape the product vision before any requirements work begins. Use when the user
  has a fuzzy initial idea and needs to clarify it.
author: Zesh-One
license: MIT
---

# Idea Refinement — From Vague Idea to Concrete Concept

## Positioning

`doc-idea` is **Phase 0** of the documentation workflow. It runs **before** requirements elicitation (`doc-rec`).

- It does **not** replace `rec`, `prd`, or any technical phase.
- It does **not** write requirements, user stories, or specifications.
- It **does** transform "I want a system that does X" into a clear, scoped, communicable product vision.

Think of it as the conversation an experienced Product Owner would have with a stakeholder who walks in with a napkin sketch — no technical jargon, no templates, just the right questions to separate what matters from what doesn't.

## When to Use

Use this skill when:

- The user describes their project with phrases like "I want to build something for…" or "I have an idea about…".
- The idea is fuzzy, contradictory, or the scope is unclear.
- The user needs to validate whether their concept makes sense before investing time in requirements.
- The full workflow (`/arch <system>`) is about to start and the idea is not yet mature.

Do **not** use when:

- The user already has a PRD, elicited requirements, or a clear specification.
- The conversation is already in technical territory (stack, APIs, databases).

---

## Flow

```
Step 1: Receive the initial idea
Step 2: Scope clarification (conditional — only if the idea is too broad)
Step 3: The 5 PO questions
Step 4: Reformulate and validate
Step 5: Capture the refined idea
```

---

## Step 1: Receive the Initial Idea

Acknowledge what the user described. Assess whether the idea is specific enough to work with meaningfully.

**Signs the idea is too broad** (any one → go to Step 2):
- No specific domain or problem area mentioned ("improve operations", "automate things").
- Multiple unrelated areas implied ("inventory, billing, and customer support").
- No user mentioned and no problem implied.

**Signs the idea is specific enough** (skip Step 2 → go directly to Step 3):
- A named domain or problem is mentioned ("a system to track warranty claims").
- A clear beneficiary is implied ("our support team can't keep up with tickets").
- The user is referencing an existing process ("we handle this in spreadsheets today").

---

## Step 2: Scope Clarification (conditional)

Ask **2–3 short questions**, all at once, to narrow the scope before diving deeper. Explain briefly why:

> "Your idea spans a few different areas — a couple of quick questions before I dig in."

Choose from:
1. **Which part of the problem matters most right now?** (What would you tackle first?)
2. **Who is the primary person or role affected by this?**
3. **What is the one outcome that would make this worth building?**

Do not ask more than 3 questions at this stage. Once answers narrow the scope to a single domain, proceed to Step 3.

---

## Step 3: The 5 PO Questions

Ask **one question at a time**, up to 5 total. Stop as soon as the concept is sufficiently clear — you do not need all five if earlier answers cover the ground.

Tailor wording to the conversation. Default questions:

1. **Who is this for?**
   > "Who are the people or roles that would use or benefit from this? Is there more than one type of user?"

2. **What problem does it solve?**
   > "What are they doing today to work around this? What hurts the most about the current process?"

3. **What does success look like?**
   > "Six months from now, how would you know this worked? What would change for the users or the business?"

4. **What is explicitly out of scope?**
   > "Is there anything you deliberately do NOT want to include in this first version? What is not part of this?"

5. **Why now?**
   > "What makes this the right moment to build this? Is there a window of opportunity, a deadline, or something that changed?"

Each question should feel conversational, not like a checklist. If the user is vague, follow up naturally:

- "Can you give me a concrete example of that?"
- "Who would be most affected if this doesn't happen?"

**Golden rule**: nothing technical. No stack, no APIs, no databases, no architecture. This is pure product discovery. If the user brings up technical topics, acknowledge them but do not deep-dive.

---

## Step 4: Reformulate and Validate

After the questions, reformulate the idea in your own words — **maximum 4 short paragraphs** covering:

1. **Purpose**: what problem it solves and for whom.
2. **Value proposition**: what changes for the users or the business.
3. **Scope**: what is included and what is explicitly excluded.
4. **Context**: why now and what visible constraints exist.

Present the summary to the user:

> "Let me make sure I understood correctly. What you want to build is: …"

If the user corrects or adjusts, iterate. If the user confirms, the refinement is complete. If the idea was already clear from the start, validate quickly and move on — do not force process.

---

## Step 5: Capture the Refined Idea

The refined concept is captured in the **master index** (`<system>.md`) as a polished 2–3 sentence project description, replacing the initial `TBD`.

Optionally, if the flow requires it or the user wants a more detailed capture, generate `<system>_idea-brief.md`:

```markdown
# Idea Brief — <System Name>

## Purpose
_(1–2 sentences: what problem it solves, for whom)_

## Value Proposition
_(1–2 sentences: what changes for users or the business)_

## Initial Scope
- **Includes**: …
- **Excludes**: …

## Context and Visible Constraints
_(why now, windows of opportunity, known limitations)_

## Open Questions
_(what is still unknown and will need to be resolved in later phases)_
```

---

## Key Principles

**Do:**
- Start with the initial idea and assess specificity before asking questions.
- Narrow broad ideas with 2–3 scope questions before the deep dive.
- Ask one question at a time during the PO phase — stop when the idea is clear.
- Reformulate in your own words and iterate until the user confirms.
- Keep the conversation at the product level — avoid all technical topics.

**Don't:**
- Never ask about stack, APIs, databases, architecture, or implementation.
- Never write requirements, user stories, or acceptance criteria — that is `rec` and `prd` work.
- Never suggest features or prescribe solutions — guide, do not design.
- Never force process if the idea is already well-formed — validate and move on.
- Never skip the reformulation step — the user must confirm the summary.

---

## Anti-Patterns

**1. Drifting into requirements**
Bad: "What fields would the registration form have?" or "How would users authenticate?"
Good: "Who would need to register and why?"

**2. Assuming the stack**
Bad: "This would be React and Node, right?"
Good: The stack is not discussed in this phase.

**3. Forcing structure when unnecessary**
Bad: Running all 5 PO questions when the user explained everything clearly.
Good: Validate quickly with a summary and move on.

**4. Confusing idea with solution**
Bad: "The user wants a dashboard with bar charts." (That is a solution.)
Good: "The user needs to understand team performance at a glance." (That is a need.)

**5. Documenting instead of conversing**
Bad: Filling in a template without having a real conversation.
Good: The conversation is the method; the document is the byproduct.

---

## Quality Checklist

- [ ] Scope clarification ran if the idea was too broad (or was determined unnecessary)
- [ ] The 5 PO questions were covered (or fewer if the idea was already clear)
- [ ] The refined idea has: purpose, value proposition, scope, and context
- [ ] No technical language or implementation decisions were introduced
- [ ] Open questions are documented, not hidden
- [ ] The user confirmed the summary before advancing
- [ ] The master index reflects the polished project description
