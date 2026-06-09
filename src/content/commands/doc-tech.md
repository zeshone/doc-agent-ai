Execute Step 5 (Tech Spec) for the specified system or module.

The user invoked: `/tech $ARGUMENTS`

Delegate to the doc-tech sub-agent with the argument: $ARGUMENTS

The argument may be:
- `<system>` → full system tech spec
- `<system>/<module>` → module tech spec (delta or full, depending on the architecture)
- `<system>/<module>/<submodule>` → sub-module tech spec

Follow the `tech` protocol defined in your skill for the corresponding level.
Prerequisite: `_prd.md` must exist. If it does not, instruct the user to run `/prd` first.
If it is a module: always ask whether it inherits the parent architecture or diverges from it.
