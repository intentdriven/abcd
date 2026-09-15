---
schema_version: 1
id: "iss-2608291814575169"
slug: "decision-append-gates-live-in-shell-not-core"
severity: "minor"
category: "architectural-insight"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: "Makefile"
---

ultra-v0.6.8 altitude 4: lint-decisions in the Makefile adds a fourth bash gate (DA001-DA004) on top of the shell stopgap the Makefile itself calls temporary, and ci.yml carries the same base-ref-resolution shell three times (record-lint, issue-resolution, decisions-append) with a hand-copied zero-sha/absent-commit guard per step; preflight checks origin/main..HEAD while CI checks BASE_SHA..HEAD. Deeper fix: DA001-DA004 belong in internal/core/lint as a record-lint verb that derives the base ref once.
