---
schema_version: 1
id: "iss-2608301620341037"
slug: "spc-57-s-tests-section-lists-none-of-the-grounds-floor-tests"
severity: "nitpick"
category: "observation"
source: "user-observation"
found_during: "itd-179-round-5-builder"
found_at: ".abcd/development/specs/open"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (errata convention owed (draft itd-2609150819439571): spc-57 is closed and its Tests section still omits the grounds-floor tests). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
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
