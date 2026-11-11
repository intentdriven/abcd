---
schema_version: 1
id: "iss-2611101258315374"
slug: "the-command-line-updater-asks-the-person"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "person's own terminal, 10 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/update/update.go"
remedy: "Compare the installed binary with the resolved release before asking; when it is already current and verified, say so and exit 0 without a question. Ask only when a swap would happen."
---

The command-line updater asks the person to confirm before it has checked whether there is anything to do. Run with the installed binary already at the latest release, it resolves the latest release, asks whether to proceed, and only after a yes reports that the binary is already current. The question is wasted: nothing would have changed either way.
