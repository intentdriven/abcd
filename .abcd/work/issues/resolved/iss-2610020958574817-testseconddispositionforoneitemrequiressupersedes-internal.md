---
schema_version: 1
id: "iss-2610020958574817"
slug: "testseconddispositionforoneitemrequiressupersedes-internal"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/ingest_reading_test.go"
remedy: "Install setSeqMinter (internal/core/capture/mint_test.go) at the start of the test, so each mint takes its own second and two dispositions can never share an id; the test asserts only that the ids differ and that the refusal names the first, which a sequential clock keeps. Shown wrong if the test then fails on an id collision. Other capture tests that mint more than one reading-ledger record (a disposition, an admission, an ingested reading item) into one ledger without a pinned minter carry the same one-in-ten-thousand same-second residue; each is pinned only after reading whether it tests the production mint itself."
resolution: "setSeqMinter pins the second-disposition test's mint to one second per record, so the two dispositions cannot share an id; the wider class of unpinned multi-mint reading-ledger tests is named in the record and left unpinned"
impact: internal
resolved_by:
  commit: "a1abdd6a8"
---

TestSecondDispositionForOneItemRequiresSupersedes (internal/core/capture/ingest_reading_test.go) mints two dispositions into one ledger through the production minter (real clock, crypto/rand suffix), so when both land in the same UTC second and draw the same four-digit suffix the second write is refused by refuseExistingRecord as a duplicate id and the test fails. It failed this way in the race lane of make preflight on fix/scanner-selector-closing-call at 6aeaea01b (dsp-2610020953467982 minted twice); the refusal is the ledger's designed behaviour, so the defect is the test's unpinned mint.

## Grounds

- pursued: TestSecondDispositionForOneItemRequiresSupersedes passes 20 runs in a row and the race lane, and with the clock and suffix pinned to collide it fails with the observed duplicate-id refusal; shown wrong if it fails again on an id collision
