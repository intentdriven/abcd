---
schema_version: 1
id: "iss-2609012039114437"
slug: "ghsa-h2gm-ledger-marker-fixed-by-15a31ea7"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/gitignore.go"
resolution: "Fixed before this run by 15a31ea7 (fix(gitutil): pin core.fsmonitor off on check-ignore probes); recorded so the advisory has a ledger marker. The fix itself is described by iss-2608270735420161."
impact: internal
resolved_by:
  commit: "15a31ea7104a82b5af999ab7ee3a28dac9a0a4a5"
---

GHSA-h2gm-w3hm-8xpq ledger marker: the advisory (git check-ignore probes executing a repo-local core.fsmonitor) was fixed before this run by 15a31ea7104a82b5af999ab7ee3a28dac9a0a4a5 "fix(gitutil): pin core.fsmonitor off on check-ignore probes" (plus 742e2f44 allow-listing the test in the hermetic-git gate; first tag v0.6.7). On main, `internal/gitutil/gitignore.go:CheckIgnored` routes through `isolatedGit` (core.hooksPath=/dev/null, core.fsmonitor=false), `internal/core/launch/bundle.go:checkIgnoredStrict` passes the same two pins explicitly with `--no-index` and `gitutil.IsolatedEnv()`, and `internal/gitutil/fsmonitor_exec_test.go:TestCheckIgnoredDoesNotExecuteRepoLocalFsmonitor` passes. The existing resolved record iss-2608270735420161 describes the fix but names no GHSA id and carries no resolved_by stamp; this record binds the advisory to its fixing commit and is not a second defect.

## Grounds

- pursued: the defect is closed on main; the record exists to bind the advisory to its fixing commit
