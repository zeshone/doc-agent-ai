---
name: doc-tech
description: |
  Create a technical specification through interactive planning and expert review. Use when starting a new feature or project that needs documented architecture, implementation approach, and design decisions. Invoke with product requirements (PRD, brief, or similar).
author: Zesh-One
license: MIT
---

# Create Technical Specification

## Workflow

### Gather Inputs

Ask the user to provide:

- **Product requirements**: PRD, product brief, or similar. Accept local files (`@path/to/file.md`), URLs (fetch via appropriate tools), or pasted content.
- **Project tracking context**: A project, epic, or initiative in their task management system (Linear, GitHub, Jira, etc.). Use available MCP tools to fetch details.
- **Output path**: Where to write the spec (e.g., `tmp/feature-tech-spec.md`).

Be flexible about input sources. Don't assume specific tools or formats.

### Positioning of `tech`

`tech` is the most technically precise phase in the flow. It is more specific than `prd` and is the phase where the architecture, technical approach, interfaces, constraints, rollout, and validation plan become explicit.

That does **NOT** mean writing in cryptic jargon. The writing must stay legible for small/medium teams, freelancers, and less experienced documentation readers while still being rigorous enough for senior engineers and tech leads.

Rules:

- Use precise technical language, but explain what decisions mean.
- Never hide assumptions. If something is missing, ask.
- If the answer is still unknown, mark it clearly as `TBD` or `Open Decision`.
- Avoid empty claims like "robust", "scalable", or "secure" unless you explain the concrete mechanism, limit, or tradeoff.

### Modules

If working on a module (`tech <sistema>/<modulo>`): first ask *"Does this module inherit the parent system architecture (delta spec, ~95% of cases) or diverge (full spec, ~5%)?"* If delta → reference the parent tech spec and document only deviations. If full → treat as standalone but align with parent Non-Goals.

### Explore Codebases

Always ask first which repositories need exploration before starting.

- Suggest likely candidates based on the project scope
- Ask the user to confirm or identify additional repos
- If a relevant repo is unknown, stop and clarify instead of inferring one silently

Scan for:
- Relevant existing patterns and conventions
- Database schemas and models
- API structure and routing
- Frontend component architecture
- Infrastructure and deployment configuration

### Planning Interview

Run a deeper technical planning interview before drafting. Guide the interview toward:

- Requirements clarification and scope boundaries
- Bounded context and domain ownership
- Repositories and codebases involved
- Integration points and external dependencies
- Interfaces and contracts (API, events, jobs, schemas, files)
- Data model, storage, migrations, and retention constraints
- Deployment topology and environment boundaries
- Security, privacy, compliance, and access control
- Observability, alerting, and operational support expectations
- Performance expectations, bottlenecks, and scaling assumptions
- Rollout, fallback, and migration strategy
- Validation strategy: technical acceptance, testing, monitoring, and post-release verification

If any of these are materially relevant and still unclear, ask follow-up questions. Do not invent answers.

### Write Initial Draft

Write incrementally to the output file. Use the template structure from [references/template.md](references/template.md).

For architecture diagrams, always use Mermaid. Never use ASCII art.

Write sections incrementally, not in one shot.

The draft must make these explicit with technical precision:

- Architecture and component boundaries
- Data flows and control flows
- Interfaces and contracts
- Key design decisions and tradeoffs
- Constraints and non-goals
- Risks and mitigations
- Rollout, migration, and fallback approach
- Technical validation plan

When information is missing:

- Ask if the missing detail blocks a sound decision
- Otherwise write `TBD` / `Open Decision` visibly in the relevant section

### Expert Review

Self-review the draft against the iteration validation rules below (stale references, contradictions, orphaned details, milestone alignment).

### Iterate

Continue refining until the user is satisfied. Long sessions with incremental changes create drift—validate consistency before concluding:

- **Stale references**: Scope changes in one section may leave dangling references elsewhere
- **Contradictions**: Later decisions may conflict with earlier sections
- **Orphaned details**: Implementation details that no longer match the architecture
- **Milestone alignment**: Ensure milestones still reflect the current scope

Also address:
- Expanding under-specified sections
- Adding missing edge cases or error handling
- Refining milestones for incremental delivery
- Replacing vague adjectives with concrete mechanisms, limits, or decisions
- Surfacing `TBD`s and open technical decisions instead of burying uncertainty

## Design Decisions Table

Captures architecture trade-offs:

| Decision | Choice | Alternatives | Rationale | Notes |
|----------|--------|--------------|-----------|-------|
| Brief description | What we chose | What we didn't | Why this choice | Caveats, follow-ups |

Populate this table throughout the process as decisions emerge. Focus on:
- Choices that required deliberation
- Trade-offs between valid alternatives
- Decisions that future readers would question

Capture decision highlights, not interview dialogue.

## Style

- This is the highest-precision phase in the flow: be more specific than `prd`
- Describe architecture and approach precisely; include implementation-level contracts when they matter
- Use clear language even when the topic is highly technical
- Explain decisions and tradeoffs, not just component lists
- Never assume missing data; ask or mark `TBD` / `Open Decision`
- Do not use vague phrases without concrete meaning
- Use Mermaid for architecture diagrams
- Reference related code, docs, and issues inline
