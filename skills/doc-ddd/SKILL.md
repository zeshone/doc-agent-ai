# Database Design Documentation

## Trigger / Positioning

**Triggers:** `/ddd`, `doc-ddd`, `ddd <sistema>`, `database design`, `schema documentation`, `ERD`, `persist*`

**Position in workflow:** After `tech` — once architecture, storage, and data flows are known.

**Hard triggers (launch without asking):**

- User mentions explicit intent: "document database", " diseño de base de datos", "db design"
- Project contains persistence layer artifacts: `.sql`, `migrations/`, `schema.prisma`, `models/`
- User references an explicit DB technology: PostgreSQL, MySQL, SQLite, MSSQL, MongoDB, etc.

**Soft triggers (ask first):**

- Architecture has data/storage sections
- `tech` mentions entities, tables, relationships, migrations
- User describes data persistence needs without naming DB

**Dismiss triggers:** Explicit exclusion: "no necesitamos base de datos", "in-memory only", "no persistence layer"

## Activation Contract

| Input | Required | Source |
|-------|----------|--------|
| `<sistema>` | Yes | From orchestrator (`arch`, `mod`) or user |
| `<sistema>_tech-spec.md` | Yes (fallback: user-provided notes) | From `doc-tech` or manual upload |
| `tech` phase completed | Yes | Self-reported |
| Language | Yes | Inherited from system (ask if ambiguous) |
| `BASE_PATH` | Yes | Platform environment |

**Fallback when `tech` is missing:** Ask the user for: existing schema files, migration history, or entity descriptions before drafting.

## Exit Contract

Output: `<BASE_PATH><sistema>/<sistema>_db-design.md`

This file MUST exist and be non-empty before concluding.

## Workflow

### Phase 1: Gather Source Knowledge

1. **Read `doc-tech` output** (if present). Extract:
   - Data model elements (entities, tables, collections)
   - Relationships and cardinalities
   - Storage constraints (retention, backup, migration strategy)
   - DBMS technology mentioned (PostgreSQL, MySQL, MongoDB, SQLite, etc.)

2. **Scan project for schema artifacts.** Look in order:
   - SQL files: `*.sql`, `migrations/`, `schema.sql`
   - ORM/framework files: `models/`, `schema.prisma`, `entities/`, `entities.ts`
   - Config files: `application.yml` (Spring), `settings.py` (Django)
   - NoSQL collections mapped in tech spec

3. **Unknown handling:** If schema artifacts exist but relationships are unclear → ask: "Did you want to document relationships between entities, or just the current structure?" If no artifacts → mark as `TBD` and draft from `tech` interpretation only.

### Phase 2: Classify DBMS (if not already stated)

Ask only if the tech spec is silent on DBMS and schema files don't reveal it.

Common options by context:
- `relational` (PostgreSQL, MySQL, MariaDB, SQLite, MSSQL)
- `document` (MongoDB, Couchbase)
- `key-value` (Redis, DynamoDB)
- `column-family` (Cassandra)
- `embedded` (SQLite on mobile/desktop)

Default: `relational` unless evidence points otherwise.

### Phase 3: Analyze Design

Extract or infer:

| Element | Description |
|---------|-------------|
| **Entities / tables / collections** | Core domain objects |
| **Attributes / columns / fields** | Per-entity properties |
| **Primary keys** | Unique identifiers |
| **Foreign keys / relationships** | How entities connect |
| **Indexes** | Performance accelerators (if documented in `tech`) |
| **Constraints** | NOT NULL, UNIQUE, CHECK, domains |

For **relational**, apply normalization sanity check:
- 1NF: Atomic values per cell
- 2NF: No partial dependencies on composite keys
- 3NF: No transitive dependencies

If you detect clear normalization violations (redundancy, update anomalies) → surface as a **Design Issue** with severity tag.

### Phase 4: Draft Document

Follow the **Database Design Document Schema** below. If schema artifacts were scanned, include inline ERD (Mermaid `erDiagram`).

If you are guessing based on `tech` alone (no schema files found) → warn: "Generating from architecture description only — verify against actual schema before using as authoritative source."

### Phase 5: Design Decisions Table

For any decision that required trade-off, ask or infer choice → captures it.

## Database Design Document Schema

```markdown
# <sistema> — Database Design Document

> **Generated:** YYYY-MM-DD  
> **DBMS:** <technology>  
> **Source:** doc-tech / schema files / manual input  
> **Language:** <en|es|pt|...>

## 1. Overview

Brief description of the data layer: what is stored, why, and how it maps to the domain.

## 2. Entity-Relationship Model *(if relational or document)*

<!-- Mermaid ERD: one block per entity table -->

## 3. Schema Details

### 3.1 <entity-name> *(repeat per entity)*

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | UUID / INT / TEXT | PK, NOT NULL | |

**Relationships:**
- `FK → <target_entity>(id)` — description
- or **standalone** (no outbound FKs)

## 4. Indexes

| Table | Index | Type | Purpose |
|-------|-------|------|---------|
| users | idx_users_email | UNIQUE | Fast lookup by email |

*(If no indexes documented: write "No indexes defined in current scope.")*

## 5. Constraints and Rules

- Domain constraints (e.g., `status IN ('active','inactive')`)
- Business rules enforced at DB level
- Referential integrity rules

## 6. Design Rationale

| Decision | Choice | Rationale |
|----------|--------|-----------|

## 7. Migrations and Lifecycle

- Migration tool / approach
- Rollback strategy
- Data retention / purge policy

## 8. Security Model *(if documented)*

- Access control at DB level
- Encryption (at-rest, in-transit)
- Audit fields (created_at, updated_at, created_by)

## 9. Open Decisions / TBDs

- Items without confirmed decision
- Items where source artifacts were ambiguous
```

## When to Stop

Conclude ONLY when:

1. Output file is written and non-empty
2. User confirms correctness or provides corrections (one round of review is expected)
3. No schema artifacts found AND user does not want manual documentation

Conclude WITHOUT writing the file (silent dismissal):
- User explicitly dismisses: "no db doc needed" / "omit database step" / "skip ddd"
- Source artifacts contain no recognizable schema (empty scan result with empty tech spec)
- System is clearly in-memory or ephemeral (no persistence intent)

## Anti-Patterns

| Forbidden | Why |
|-----------|-----|
| Copy-paste raw SQL as "documentation" | Unreadable; doesn't explain design intent |
| Document every column with no context | Drown signal in noise |
| Use ASCII art for diagrams | Mermaid only |
| Invent schema that contradicts `tech` | Respect upstream artifact as source of truth |
| Document entities absent from `tech` without flagging | Mark uncorroborated entities with ⚠️ |
| Skip relationships section | Relationships are the design, not the table list |

## Integration Points

| Phase | Interaction |
|-------|-------------|
| `arch` / `mod` orchestrator | Receives option to trigger `doc-ddd`. Asks user: "Include database design documentation?" Only launches if user confirms or hard trigger fires. |
| `doc-tech` | Primary source. `doc-ddd` reads the tech spec to extract data layer elements. |
| Platform `BASE_PATH` | Output target is `<BASE_PATH><sistema>/<sistema>_db-design.md` |
| Project schema files | Scanned automatically if present; not required but enhances accuracy |
