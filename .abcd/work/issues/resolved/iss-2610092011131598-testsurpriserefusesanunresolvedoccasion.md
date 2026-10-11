---
schema_version: 1
id: "iss-2610092011131598"
slug: "testsurpriserefusesanunresolvedoccasion"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "2026-10-09 merge queue run for PR #880"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/surprise_test.go"
remedy: "Have the id allocator draw again (a bounded number of times) when a freshly minted id collides with one already in the ledger within the same process, keeping the refusal only for a collision with a committed record a peer made; test it with a forced collision on the first draw, watched fail first."
resolution: "Surprise, admission, disposition and reframe ids now mint through mintRecordID, which draws again (at most eight times) when the drawn id lands on a record this process already wrote; a clash with a record the process did not write is still refused at the write."
impact: fix
resolved_by:
  commit: "8c4e821d2955a8b48c47d8dee4e8635d7727007f"
---

TestSurpriseRefusesAnUnresolvedOccasion failed in the merge queue on 2026-10-09 (PR #880, macOS race lane, run 37980092117) with 'duplicate issue id: srp-2610092000060599 already exists in this ledger; the mint collided with a record already committed': the test mints three surprise records in quick succession, two drew the same timestamp-plus-four-digit id within one second, and the allocator refuses a collision instead of drawing again, so a records-only pull request was dropped from the queue.

## Grounds

- pursued: several reading-ledger records written in one second by one process no longer fail on a same-second id clash, while a peer's record is never overwritten; TestSurpriseRedrawsAnIDThisProcessAlreadyWrote failing, or TestSurpriseRefusesACollisionWithAPeersRecord accepting the write, would show it wrong
