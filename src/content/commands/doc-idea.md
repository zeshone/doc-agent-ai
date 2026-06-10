Execute Step 1 (idea refinement) for the specified system or module.

The user invoked: `/idea $ARGUMENTS`

Delegate to the doc-idea sub-agent with the argument: $ARGUMENTS

The argument may be:
- `<system>` → system-level idea refinement
- `<system>/<module>` → module-level idea refinement
- `<system>/<module>/<submodule>` → sub-module-level idea refinement

Follow the `idea` protocol defined in your skill for the corresponding level.
Idea is pure product discovery — no stack, no APIs, no databases.
Output: master index description (and optionally `_idea-brief.md`).

---

{{PATH_RESOLUTION}}
