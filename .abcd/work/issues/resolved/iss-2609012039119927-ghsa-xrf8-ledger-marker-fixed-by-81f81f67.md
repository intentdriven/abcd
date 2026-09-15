---
schema_version: 1
id: "iss-2609012039119927"
slug: "ghsa-xrf8-ledger-marker-fixed-by-81f81f67"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/apply.go"
resolution: "Fixed before this run by 81f81f67 (fix(ahoy): contain install writes against a committed .abcd ancestor symlink); recorded so the advisory has a ledger marker. The fix itself is described by iss-2608270735428527."
impact: internal
resolved_by:
  commit: "81f81f67df7132067cc04fedda9609224fd052a0"
---

GHSA-xrf8-4432-gw2f ledger marker: the advisory (ahoy install writes escaping the working tree through a committed .abcd ancestor symlink) was fixed before this run by 81f81f67df7132067cc04fedda9609224fd052a0 "fix(ahoy): contain install writes against a committed .abcd ancestor symlink" (first tag v0.6.7). On main, `internal/core/ahoy/apply.go:Install` refuses up front via `abcdDirHazard` (a symlink or non-directory .abcd is `Status: "refused"`), `store.go:writeRepoJSON` and `writeConfig` resolve through `os.OpenRoot(cwd)` plus `fsutil.WriteFileAtomicPreserveModeInRoot`, and `internal/core/ahoy/symlink_escape_test.go:TestInstallRefusesAbcdDirSymlinkEscape` passes. The existing resolved record iss-2608270735428527 describes the fix but names no GHSA id and carries no resolved_by stamp, so the advisory was not findable from the ledger by id; this record binds the two and is not a second defect.

## Grounds

- pursued: the defect is closed on main; the record exists to bind the advisory to its fixing commit
