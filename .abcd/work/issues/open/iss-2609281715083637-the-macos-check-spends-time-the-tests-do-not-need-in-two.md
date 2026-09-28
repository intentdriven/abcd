---
schema_version: 1
id: "iss-2609281715083637"
slug: "the-macos-check-spends-time-the-tests-do-not-need-in-two"
severity: "minor"
category: "tech-debt"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: ruling AR"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/fsutil/fsutil.go"
---

The macOS check spends time the tests do not need in two places. Every atomic write the tests make flushes to stable storage (fsutil's WriteFileAtomic family and the history-index bootstrap call File.Sync, which Go runs as F_FULLFSYNC on macOS, about 8.8 ms per synced write on this disk), though no test asserts anything a flush makes true; and the race step names ./internal/... in import-path order, so go test starts internal/surface/cli, the slowest package under -race, among the last on the three-core runner and the step waits on it as a tail.
