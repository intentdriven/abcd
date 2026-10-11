---
schema_version: 1
id: "iss-2609281715083637"
slug: "the-macos-check-spends-time-the-tests-do"
severity: "minor"
category: "tech-debt"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: ruling AR"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/fsutil/fsutil.go"
resolution: "Both levers built under ruling AR. 7bbf7527a routes every flush through fsutil.Flush, which skips File.Sync only in a test binary that set ABCD_TEST_SKIP_FLUSH=1 (the Makefile's test and preflight targets and ci.yml's test steps set it); a cli plain run made 2,889 in-process flushes costing 18.6s locally. 94d311fcd names the nine slowest race packages ahead of ./internal/..., the same 78 packages by go list; at -p 3 on a scratch snapshot the race lane took 845s and 811s in import-path order and 557s slowest-first."
impact: internal
resolved_by:
  commit: "94d311fcd"
---

The macOS check spends time the tests do not need in two places. Every atomic write the tests make flushes to stable storage (fsutil's WriteFileAtomic family and the history-index bootstrap call File.Sync, which Go runs as F_FULLFSYNC on macOS, about 8.8 ms per synced write on this disk), though no test asserts anything a flush makes true; and the race step names ./internal/... in import-path order, so go test starts internal/surface/cli, the slowest package under -race, among the last on the three-core runner and the step waits on it as a tail.

## Grounds

- pursued: the macOS check job shortens by the flush time of every test write and by the race step's single-package tail; a macOS check on the merged tip no shorter than the last green run (23.7 min, race step 888s) would show it wrong
