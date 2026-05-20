---
name: doc-scope
description: 'Classify the scope of a legacy feature via bounded discovery. Trigger: scope classifier, legacy feature scope, invoked by doc-feat only.'
author: Zesh-One
license: MIT
---

# Scope Classifier

Invoked ONLY by `doc-feat`. Not user-invokable directly.

## Activation Contract

Inputs received from `doc-feat`:
- `<ruta-legacy>` — absolute path, already verified as an existing directory
- `<descripcion>` — raw description string

Triggered only when no `--scope` flag was given. Returns `(mode, anchor_or_pattern)` to `doc-feat`.

## Bounded Reads (HARD CAPS — do NOT exceed)

1. Glob top-level of `<ruta-legacy>`: list directories and manifest files only. NO recursive glob.
2. Read at most **5** surface files in this priority order (stop when you reach 5):
   - `package.json`, `go.mod`, `pom.xml`, `Cargo.toml`, `pyproject.toml`
   - `README.md`, `README.*`, `CHANGELOG.md`
3. STOP. No AST analysis. No directory recursion.

## Decision Tree

Apply in order. First match wins.

**Case 1 — Local anchor**
If `<descripcion>` mentions a token `T` that matches a top-level directory name (case-insensitive, substring or kebab/camel/snake variant) → propose `mode=local`, `anchor=<that dir absolute path>`.

**Case 2 — Cross-cutting**
Else if `<descripcion>` contains any of:
`logging`, `log`, `auth`, `authn`, `authz`, `audit`, `tracing`, `telemetry`, `metrics`, `observability`, `i18n`, `l10n`, `locale`, `permissions`, `rbac`, `error handling`, `errors`, `retry`, `validation`, `sanitization`, `security`
→ propose `mode=cross`, `pattern=<keyword-derived grep, e.g. "*Logger*" or "auth.*">`.

**Case 3 — None**
Else → propose `mode=none`, `anchor_or_pattern=null`.

## Confirmation Protocol (MANDATORY)

Emit to user:
> "I propose mode `<mode>` with `<anchor_or_pattern>`. Confirm, correct, or force another mode?"

Accept these responses:

| User input | Action |
|------------|--------|
| `ok` / `yes` / `sí` / `confirm` | Return `(mode, anchor_or_pattern)` as proposed |
| `local <path>` | Override mode=local, anchor=path. Return. |
| `cross <pattern>` | Override mode=cross, pattern=pattern. Return. |
| `none` | Override mode=none. Return. |
| Free-text correction | Adopt user's wording verbatim. Return. |

## Return Contract

Return the tuple `(mode, anchor_or_pattern)` to `doc-feat`.

- NEVER write files.
- NEVER recurse beyond the top-level directory listing.
- NEVER re-run classification after the user has confirmed or overridden.
