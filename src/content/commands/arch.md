Execute the complete documentation workflow for the specified system.

The user invoked: `/arch $ARGUMENTS`

Follow the `arch <system>` protocol defined in your skill:
1. Verify or create the `{{BASE_PATH}}$ARGUMENTS/` directory
2. Create the master index if it does not exist
3. Run idea → rec → prd → refine → tech → [ddd] → pti in sequence
4. Pause between each step for user confirmation
5. After `tech`: ask "¿Querés documentar el diseño de la base de datos?" unless user explicitly excluded or hard trigger applies
6. Update the index checkboxes as each phase is completed
7. `ddd` is optional — only run if user confirms or hard trigger fires
