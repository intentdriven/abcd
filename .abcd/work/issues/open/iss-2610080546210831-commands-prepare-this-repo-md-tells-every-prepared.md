---
schema_version: 1
id: "iss-2610080546210831"
slug: "commands-prepare-this-repo-md-tells-every-prepared"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "review of the hooks_path lane (run-2610080127501542), 2026-10-08"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/prepare-this-repo.md"
remedy: "Make the prepare-this-repo step that sets core.hooksPath conditional on the hooks_path state ahoy now reports: set it only when unarmed, and when a global hooks dispatcher is in force (foreign) tell the person to call the repo's .githooks from it instead of overriding it."
---

commands/prepare-this-repo.md tells every prepared repository to run git config core.hooksPath .githooks with no condition, which shadows a global hooks dispatcher: the exact advice ahoy's status now refuses to give when hooks_path is foreign.
