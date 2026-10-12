---
schema_version: 1
id: "iss-2610111820465940"
slug: "the-pre-commit-hook-ahoy-install"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "review of run-2610111657473488 lane-1, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/defaults/pre-commit"
remedy: "Give the scaffolded pre-commit header the same three-way core.hooksPath advice as commands/prepare-this-repo.md step 5 (unarmed: set core.hooksPath; foreign: have the global dispatcher call .githooks/; armed: nothing), and extend prepare_hookspath_test.go's pattern check to this file, watched fail first."
refines: [iss-2610101903385281]
resolution: "The scaffolded pre-commit header now gives the three-way core.hooksPath advice keyed on banlist.hooks_path (unarmed: set it; foreign: have the global dispatcher call .githooks/; armed: nothing), and TestHooksPathSiblingsArmOnlyWhenUnarmed judges the file through a comment-aware branch end."
impact: fix
resolved_by:
  commit: "2204fc1fa9f18faf3ba208cffe4dd1360032a36c"
---

The pre-commit hook ahoy install scaffolds into adopted repositories (internal/core/ahoy/defaults/pre-commit, header lines 8-10) still tells every clone to run 'git config core.hooksPath .githooks' unconditionally, which shadows a foreign global hooks dispatcher instead of having it call .githooks/. A third sibling of the advice iss-2610101903385281 made conditional in CONTRIBUTING.md and the commit-msg hook; found by that lane's review (run-2610111657473488, PR #939).

## Grounds

- pursued: an adopted repository's scaffolded pre-commit no longer tells a clone under a global hooks dispatcher to set a local core.hooksPath; a setsHooksPath match in internal/core/ahoy/defaults/pre-commit outside its unarmed comment bullet, or a missing armed/unarmed/foreign bullet, would show it wrong, and the sibling test would fail on it.
