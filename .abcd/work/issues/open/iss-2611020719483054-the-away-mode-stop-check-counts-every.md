---
schema_version: 1
id: "iss-2611020719483054"
slug: "the-away-mode-stop-check-counts-every"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "parallel sessions in sibling worktrees, 2026-11-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/autonomy/owedread.go"
remedy: "Have the stop check's owed-work read count only the runs and lanes the stopping session drives (by its session claim, or the worktree it stands in), and name another session's live run as theirs rather than owed here."
---

The away-mode stop check counts every live run's lanes as this checkout's own: with several sessions building in sibling worktrees of one repository, a session's turn end was refused more than fifty times on 2026-11-01 for lanes other sessions owned on their own branches, because the owed-work read takes every run the checkout's local tier lists rather than the runs this session drives.
