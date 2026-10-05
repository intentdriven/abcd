---
schema_version: 1
id: "iss-2610050927169919"
slug: "the-findings-gate-reads-open-issues-at-head"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "review of the launch ship uncommitted-records fix, 2026-10-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/changelog/findings.go"
remedy: "Have GuardFindings refuse, or count, an open-folder record that differs from HEAD, through the same launch.DirtyTreeFiles reader the uncommitted-records refusal uses."
---

GuardFindings reads the open/ issue folder at HEAD, so a major or critical finding captured but not yet committed is invisible to the findings gate; with --allow-dirty past the ingest's dirty-tree gate, the cut proceeds past it. Same class as iss-2610050259118177 (records read at HEAD), on the open side.
