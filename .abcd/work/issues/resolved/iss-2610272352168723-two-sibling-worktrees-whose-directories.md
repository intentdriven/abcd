---
schema_version: 1
id: "iss-2610272352168723"
slug: "two-sibling-worktrees-whose-directories"
severity: "nitpick"
category: "bug"
source: "user-observation"
found_during: "review pair of fix/peers-cap-warning after its rebase, 2026-10-27"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/view.go"
remedy: "Disambiguate clashing sibling names by the shortest distinguishing path suffix rather than one parent up, and key the ceiling's per-sibling count by copy key, not by the shown name"
resolution: "a sibling reads by the fewest trailing folders that tell it apart, and the ceiling counts siblings by copy key"
impact: fix
resolved_by:
  commit: "1dd021024ab01b1af02bbee8daaab12f49cd9f95"
---

Two sibling worktrees whose directories share a name and a parent's name (x/a/lane-b and y/a/lane-b) both read as 'a/lane-b' in abcd peers, and the ceiling counts them as one sibling 'a/lane-b' with both holds: the total stays right, but which worktree holds what is lost
