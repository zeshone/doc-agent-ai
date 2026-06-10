Execute Step 6 (issue breakdown) for the specified system or module.

The user invoked: `/doc-pti $ARGUMENTS`

Delegate to the doc-pti sub-agent with the argument: $ARGUMENTS

The argument may be:
- `<system>` → system issues
- `<system>/<module>` → module issues
- `<system>/<module>/<submodule>` → sub-module issues

Follow the `pti` protocol defined in your skill for the corresponding level.
Prerequisite: `_prd.md` must exist. If it does not, instruct the user to run `/doc-prd` first.
Generate the issues as a local `.md` file — do not create them in GitHub unless the user explicitly requests it.

---

{{PATH_RESOLUTION}}
