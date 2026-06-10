Initialize a new module and run its complete documentation workflow (idea → rec → prd → refine → tech → [ddd] → pti, ddd optional).

The user invoked: `/mod $ARGUMENTS`

The argument has two forms:
- `<system> <module>` → create `modules/<module>/` inside the system
- `<system>/<module> <submodule>` → create `modules/<submodule>/` inside the module

Follow the `mod` protocol defined in your skill:
1. Verify that the parent system exists and uses an evolutionary archetype
2. Create the module directory at the correct path
3. Create the module index with a bidirectional link to the parent
4. Add the module to the system master index
5. Run idea → rec → prd → refine → tech → [ddd] → pti in sequence, pausing between each step
6. Ask about `ddd` between `tech` and `pti` unless excluded or hard trigger fires
7. `ddd` is optional — only run if user confirms

---

{{PATH_RESOLUTION}}
