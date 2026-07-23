Execute the SDD Context Compaction step.

The user invoked: `/doc-to-sdd $ARGUMENTS`

Delegate to the `doc-to-sdd` sub-agent with the argument: `$ARGUMENTS`.

**This is a standalone command.** It is NOT part of the `/doc-arch` full-flow sequence. It can run independently after any combination of documentation phases.

**Prerequisites:** At least one business-layer artifact (`_requirements.md` or `_prd.md`) OR at least one technical-layer artifact (`_tech-spec.md` or `_db-design.md`) must exist under the docs root resolved per the preamble below.

**Output:** LLM-optimized SDD context files written to `agent_sdd_context_project/` under the resolved docs root. Naming is mode-aware:
- Vault mode: `<system>_sdd-context.md` / `<system>_sdd-tech-context.md` (system-level), or `<system>_<module>_sdd-context.md` / `<system>_<module>_sdd-tech-context.md` (feature/module-level)
- In-project mode: `_sdd-context.md` / `_sdd-tech-context.md` (bare, unchanged)

Both files carry the same content regardless of mode:
- Business layer file — problem, requirements, stories, decisions
- Technical layer file — architecture, stack, contracts, data model

All output is in English regardless of source artifact language.

---

{{PATH_RESOLUTION}}
