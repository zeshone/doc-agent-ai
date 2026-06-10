Execute the Database Design Documentation step.

The user invoked: `/doc-ddd $ARGUMENTS`

Delegate to the `doc-ddd` sub-agent with the argument: `$ARGUMENTS`.

**This is an optional step.** It is triggered when:
1. User explicitly invokes `/doc-ddd`
2. Project contains persistence artifacts (`.sql`, `migrations/`, `schema.prisma`, `models/`)
3. The orchestrator asks "Include database design documentation?" and the user confirms

**Prerequisites:** `doc-tech` phase should be completed first. If the tech spec is missing, provide schema files, migration history, or entity descriptions manually.

**Output:** `<sistema>_db-design.md` written under the docs root resolved per the preamble below.

The argument may be:
- `<sistema>` → system DB design document
- `<sistema>/<modulo>` → module DB design document
- `<sistema>/<modulo>/<submodulo>` → sub-module DB design document

---

{{PATH_RESOLUTION}}
