---
schema_version: 1
id: "iss-2611252133341605"
slug: "status-json-prints-a-zero-lapsed-at"
severity: "nitpick"
category: "bug"
source: "agent-finding"
found_during: "planning of 25 November"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/release.go"
remedy: "Use omitzero, or a pointer as the sibling record in claim does, and test that an ended release with no lapse prints no lapsed_at."
resolution: "abcd peers --json no longer prints a lapsed_at of 0001-01-01T00:00:00Z in its release_withdrawals rows for a release that has not lapsed (the abcd status --json document carries no release withdrawals, so it never printed one): ReleaseWithdrawals.LapsedAt takes omitzero, and so does ArmedMerges.LapsedAt for consistency, though that struct is read in memory by launch's merge-hold row and never JSON-encoded in production. The same defect is fixed wherever a value-typed time was tagged omitempty: abcd history staged --all-repos --json no longer prints a zero oldest_staged_at for a repository holding only quarantined transcripts, and the quarantined entries and the stop check's session state, owed set and owned work drop unset times too. TestAnUnlapsedReleasePrintsNoLapsedAt and the TestAZeroTimePrintsNoKey tests in history and autonomy hold this, and TestAnOlderSessionStateStillLoads holds that a session state an older abcd wrote with zero times still loads (dcn-2611271510101433, which corrects dcn-2611271404412782)."
impact: fix
---

abcd status --json prints a lapsed_at of 0001-01-01T00:00:00Z for a release withdrawal that has not lapsed: the field is a time with omitempty, which never drops a zero time. A reader of the JSON takes it for a lapse at that instant. Run on 2026-11-25.
