---
schema_version: 1
id: "iss-2610112148115897"
slug: "the-prepare-commit-msg-hook-ahoy-install"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "lane for iss-2610111820465940, run-2610112109503709, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/defaults/prepare-commit-msg"
remedy: "Give the scaffolded prepare-commit-msg header the same three-way core.hooksPath advice as commands/prepare-this-repo.md step 5 (unarmed: set core.hooksPath; foreign: have the global dispatcher call .githooks/; armed: nothing), and add the file to TestHooksPathSiblingsArmOnlyWhenUnarmed's sibling list with the shellCommentBranchEnd judge, watched fail first."
duplicates: [iss-2610111820465940]
---

The prepare-commit-msg hook ahoy install scaffolds into adopted repositories (internal/core/ahoy/defaults/prepare-commit-msg, header lines 20-22) still tells every clone to run 'git config core.hooksPath .githooks' unconditionally, shadowing a foreign global hooks dispatcher. The fourth sibling of the advice iss-2610101903385281 made conditional, found by the lane for iss-2610111820465940 (the scaffolded pre-commit). A sweep of every core.hooksPath mention on main at 2026-10-11 found no other unconditional instruction.
