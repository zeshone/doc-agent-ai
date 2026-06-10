Execute Step 4 (user story refinement) for the specified system or module.

The user invoked: `/doc-refine $ARGUMENTS`

Delegate to the doc-refinement sub-agent with the argument: $ARGUMENTS

The argument may be:
- `<system>` → audit all user stories in the system PRD against INVEST criteria
- `<system>/<module>` → audit module PRD user stories
- `<system>/<module>/<submodule>` → audit sub-module PRD user stories
- empty → standalone mode: the user provides a single story to refine

Follow the `refine` protocol defined in your skill for the corresponding level.
Prerequisite for non-standalone mode: `_prd.md` must exist. If it does not, instruct the user to run `/doc-prd` first.
Refine is a quality gate — never add, delete, or change story scope without explicit user confirmation.

---

{{PATH_RESOLUTION}}
