---
schema_version: 1
id: "iss-2608301620343236"
slug: "the-len-rs-1-guard-in-scriptiocontinuaonly-is-unreachable-as"
severity: "nitpick"
category: "tech-debt"
source: "user-observation"
found_during: "itd-179-round-5-builder"
found_at: "internal/core/grounds/grounds.go"
resolution: "The guard is kept and pinned by a direct helper test rather than deleted, because it is half of the predicate's definition rather than defensive scaffolding."
impact: internal
resolved_by:
  intent: "itd-179"
---

the len(rs) != 1 guard in scriptioContinuaOnly is unreachable as called so no fixture can enter it

Reported by the round-5 builder and left in place with the reasoning stated in
`6330b8b2`'s body rather than claimed as tested, which was the right call at
the time.

`textUnits` emits a scriptio-continua letter as its own single-rune unit, so a
multi-rune unit never begins with one, and the sibling test
`!isScriptioContinuaLetter(rs[0])` already rejects every shape `len(rs) != 1`
would. Deleting it leaves the whole suite green, and the builder found no text
reaching `ValidateText` that distinguishes the two.

The first reading of this record routed it to deletion, on the sibling
precedent: itd-189's round 3 deleted two stand-downs under the finding that a
branch no fixture can enter is how a guard comes to look tested, and
iss-2608301519254240 was raised against the twin it left behind.

Inspection reversed that, and the difference is worth keeping. Those stand-downs
were defensive scaffolding against states their callers could not produce. This
one is half of what the predicate MEANS. "Every unit is a single
scriptio-continua letter" and "every unit begins with one" are different claims
which agree only because of how `textUnits` splits today, and deleting the
length test would leave the weaker claim in the code, agreeing silently for as
long as that holds and no longer.

So it is pinned rather than removed, at the helper, which is the only level a
fixture for it exists at. Unreachable through `ValidateText` is not the same as
unreachable, and the test that proves it is load-bearing is the one that kills
the mutant `textUnits` cannot currently feed it.

## Grounds

- pursued: We expect a direct helper test to be the right close rather than deletion, because the guard states what a unit must be and not merely what the caller happens to send, so removing it would leave a weaker claim agreeing silently with the stronger one until textUnits changed.
