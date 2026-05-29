Execute the Database Design Documentation step.

The user invoked: `/ddd $ARGUMENTS`

Delegate to the `doc-ddd` sub-agent with the argument: `$ARGUMENTS`.

**This is an optional step.** It is triggered when:
1. User explicitly invokes `/ddd`
2. Project contains persistence artifacts (`.sql`, `migrations/`, `schema.prisma`, `models/`)
3. The orchestrator asks "Include database design documentation?" and the user confirms

**Prerequisites:** `doc-tech` phase should be completed first. If the tech spec is missing, provide schema files, migration history, or entity descriptions manually.

**Output:** `<BASE_PATH><sistema>/<sistema>_db-design.md`

The argument may be:
- `<sistema>` → system DB design document
- `<sistema>/<modulo>` → module DB design document
- `<sistema>/<modulo>/<submodulo>` → sub-module DB design document
