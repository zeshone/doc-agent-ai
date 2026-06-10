Execute Step 3 (PRD) for the specified system or module.

The user invoked: `/doc-prd $ARGUMENTS`

Delegate to the doc-prd sub-agent with the argument: $ARGUMENTS

The argument may be:
- `<system>` → system-level PRD
- `<system>/<module>` → module-level PRD
- `<system>/<module>/<submodule>` → sub-module-level PRD

Follow the `prd` protocol defined in your skill for the corresponding level.
Prerequisite: `_requirements.md` must exist. If it does not, instruct the user to run `/doc-rec` first.

---

{{PATH_RESOLUTION}}
