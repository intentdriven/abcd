---
schema_version: 1
id: "iss-2610080546210831"
slug: "commands-prepare-this-repo-md-tells"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "review of the hooks_path lane (run-2610080127501542), 2026-10-08"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/prepare-this-repo.md"
remedy: "Make the prepare-this-repo step that sets core.hooksPath conditional on the hooks_path state ahoy now reports: set it only when unarmed, and when a global hooks dispatcher is in force (foreign) tell the person to call the repo's .githooks from it instead of overriding it."
resolution: "The prepare-this-repo commit-gates step already sets core.hooksPath only when banlist.hooks_path reads unarmed, and for foreign tells the person to have the dispatcher call .githooks/ (reworded in afa6e7b16). This change adds the detector that holds it there: the arming command is refused outside the unarmed branch, every HooksPathState must have a branch, and the JSON path the page names is pinned to ahoy's tags."
impact: internal
resolved_by:
  commit: "d0b94320892abc817b5e0a5ccb1949f25497201a"
---

commands/prepare-this-repo.md tells every prepared repository to run git config core.hooksPath .githooks with no condition, which shadows a global hooks dispatcher: the exact advice ahoy's status now refuses to give when hooks_path is foreign.

## Grounds

- pursued: prepare-this-repo never advises a local core.hooksPath unless ahoy reports hooks_path unarmed; shown wrong if the arming command appears outside the unarmed branch, the foreign branch stops pointing at the dispatcher, or banlist.hooks_path stops being the field ahoy emits, any of which fails TestPrepareThisRepoArmsHooksOnlyWhenUnarmed or TestPrepareHooksPathFieldIsWhatAhoyEmits
