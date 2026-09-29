---
schema_version: 1
id: "iss-2608301620341037"
slug: "spc-57-s-tests-section-lists-none-of-the-grounds-floor-tests"
severity: "nitpick"
category: "observation"
source: "user-observation"
found_during: "itd-179-round-5-builder"
found_at: ".abcd/development/specs/open"
remedy: "Waits on the itd-2609150819439571 planning interview (M25, the errata convention): record in spc-57, in whatever correction form the convention settles (an appended dated Errata section is the candidate), the grounds floor tests added in rounds three to five, the TestGroundsFloor tests in `internal/core/grounds/grounds_test.go`, without rewriting the closed Tests section, and resolve this record with that change, proven by record-lint green."
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed A): Under the errata convention (M25), where does a correction to a closed spec's Tests section go?"
---

spc-57's Tests section lists none of the grounds floor tests added in rounds three to five

Reported by the round-5 builder. Pre-existing rather than branch-introduced by
round 5: round 4 did not list its own additions either.

`spc-57`'s Tests section names none of the grounds floor tests added across
rounds 3, 4 and 5, so the spec's account of what holds the floor is behind the
floor by three rounds. Record currency, not behaviour: nothing is untested, the
record just does not say what tests it.

Recorded rather than fixed in the same change because the spec is the itd-179
ship's own artefact and its Tests section is written at the ship commit, which
is where this belongs.

## Remedy grounds (2026-09-29)

- spc-57 is closed, so a correction to its Tests section is exactly the case the errata convention exists to place; the test names are found at the base, so the erratum's content is ready once its form is.
- No outside-practice check: the form is the convention's to settle, and a sibling record carries the SOTA check for it.
- Rejected: editing the closed spec in place, which the convention's planning is there to rule on.
