---
schema_version: 1
id: "iss-2608261533290815"
slug: "privacy-hygiene-silently-skips"
severity: "nitpick"
category: "bug"
source: "agent-observation"
found_during: "bughunt-a round 9"
found_at: "internal/core/repolint/rule_privacy.go"
resolution: "Open failures are classified: EACCES/EPERM/EIO warn not-scanned, absent paths and symlink-shaped refusals stay silent by design; polarity table pinned"
impact: fix
resolved_by:
  commit: "e6f90055"
---

repolint privacy-hygiene silently skips a tracked file it cannot open: readTrackedFile returns not-ok with no oversize marker on an open failure, and the caller emits a finding only for the oversize case — so an EACCES/ENOTDIR tracked file produces no finding and the rule reports the repository clean, against the engine contract that a check that cannot run must not be silently reported as passing. The oversize arm of the same helper got exactly this fix; the open arm beside it stayed silent. Constraint on the fix: TrackedFiles lists index entries, so ENOENT (deleted-in-worktree, sparse checkout) must stay silent and only genuine unreadability warns. Acceptance: a non-ENOENT open failure yields a not-scanned warn finding, watched-fail via an ENOTDIR fixture.