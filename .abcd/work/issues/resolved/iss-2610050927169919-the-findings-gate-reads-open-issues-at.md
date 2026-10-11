---
schema_version: 1
id: "iss-2610050927169919"
slug: "the-findings-gate-reads-open-issues-at"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "review of the launch ship uncommitted-records fix, 2026-10-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/changelog/findings.go"
remedy: "Have GuardFindings refuse, or count, an open-folder record that differs from HEAD, through the same launch.DirtyTreeFiles reader the uncommitted-records refusal uses."
resolution: "GuardFindings refuses a working tree whose open/ issue records differ from HEAD (FindingGuard.Uncommitted, every path named, no --allow-dirty waiver), read through launch.DirtyTreeFiles as the derivation's uncommitted-records refusal is; the release cut raises it through the guard's own verdict backstop"
impact: fix
resolved_by:
  commit: "f0d46e83ec45c14fc66b2cc1cc9c850c89d9bbbc"
---

GuardFindings reads the open/ issue folder at HEAD, so a major or critical finding captured but not yet committed is invisible to the findings gate; with --allow-dirty past the ingest's dirty-tree gate, the cut proceeds past it. Same class as iss-2610050259118177 (records read at HEAD), on the open side.

## Grounds

- pursued: launch ship over an uncommitted major/critical capture in open/ is refused naming the path, with or without --allow-dirty; shown wrong if Emit returns a ready cut while any open/ record differs from HEAD, or if dirt outside open/ (a README, a non-record file) refuses a clean cut
