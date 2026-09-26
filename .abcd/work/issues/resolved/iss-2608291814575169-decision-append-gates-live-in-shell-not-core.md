---
schema_version: 1
id: "iss-2608291814575169"
slug: "decision-append-gates-live-in-shell-not-core"
severity: "minor"
category: "architectural-insight"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: "Makefile"
resolution: "DA001-DA004 run in internal/core/lint (decisionsappend.go) behind 'record-lint decisions-append <base> <head>'; the base ref is derived once in Go (gitutil.ResolveRangeBase), and the decisions step in ci.yml passes BASE_SHA straight through. The shell gate and its case suite are retired: every case is a TestDecisionsAppend* case, and a whole-history replay (3,874 commits) gives the same 70 findings from both gates, byte-identical. The record-lint -agent-diff and issue-resolution range steps keep their own guards; they call gates this change does not port."
impact: internal
resolved_by:
  commit: "5b804989"
---

ultra-v0.6.8 altitude 4: lint-decisions in the Makefile adds a fourth bash gate (DA001-DA004) on top of the shell stopgap the Makefile itself calls temporary, and ci.yml carries the same base-ref-resolution shell three times (record-lint, issue-resolution, decisions-append) with a hand-copied zero-sha/absent-commit guard per step; preflight checks origin/main..HEAD while CI checks BASE_SHA..HEAD. Deeper fix: DA001-DA004 belong in internal/core/lint as a record-lint verb that derives the base ref once.

## Grounds

- pursued: the Go gate refuses every fixture the shell suite refused and passes every one it passed, and agrees with the shell gate on the whole real history; a fixture or a historical commit on which the two verdicts differ would show it wrong
