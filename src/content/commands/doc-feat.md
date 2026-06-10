Execute the legacy feature documentation mini-flow.

The user invoked: `/doc-feat $ARGUMENTS`

Follow the `feat <ruta-legacy> <descripcion> [--scope ...]` protocol defined in
the `doc-feat` skill at `{{BASE_PATH}}`:
1. Parse $ARGUMENTS into positional + flag tokens
2. Derive <sistema> and <slug> from the legacy path and description
3. Run scope classifier (or honor --scope flag if provided)
4. Mini-flow: rec-lite → prd-lite → (optional tech, risk-gated) → pti
5. Output to `{{BASE_PATH}}<sistema>-features/<slug>/` (vault) or `docs/doc-agent/features/<slug>/` (in-project), per the resolution rules below

---

{{PATH_RESOLUTION}}
