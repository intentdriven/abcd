---
schema_version: 1
id: "iss-2608301206034359"
slug: "the-grounds-substance-floor-counts-runes-not-letters-so-twen"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "itd-179-round-2-security"
found_at: "internal/core/grounds/grounds.go"
resolution: "ValidateText now measures the floor in letter-runs as well as runes, so a text of zero-width spaces, dots or digits carries no words and is refused at the one place both the writer and the reader inherit it from"
impact: fix
resolved_by:
  intent: "itd-179"
---

the grounds substance floor counts runes not letters so twenty zero-width spaces satisfy the promoted refusal

Found by the round-2 adversarial security review of build/itd-179.

`abcd intent ready itd-N --grounds "pursued:<20 x U+200B>"` is accepted and
written. `Fold` leaves the zero-width runes (TrimSpace trims NBSP but not
ZWSP); `ValidateText` passes the rune-length check; `wordRe` finds zero
letter-runs, and `onlyDegenerate := len(words) > 0` at
`internal/core/grounds/grounds.go:163` short-circuits to false. The reader
`groundsCheck` applies the same floor, so the record renders as an entry with
nothing visible in it and the seventh readiness check reports OK. Twenty dots
and twenty digits pass identically.

This matters because the check is the one this intent PROMOTED from a report to
a refusal. A refusal answerable with twenty zero-width spaces is not a floor.
The type doc at lines 58-63 claims it "refuses the degenerate texts", and
`commands/intent.md` tells the host that `- pursued: yes` is not an entry,
which is true while twenty dots is.

Same shape as the defect c2dfd9ad already fixed (the check claimed a floor the
reader did not enforce); the floor now exists, and a zero-letter text walks
through it.

Remedy: `ValidateText` requires a minimum count of `wordRe` matches and refuses
`len(words) == 0`, rather than counting runes. Both the writer and
`ParseGrounds` inherit it from the one place.

## Grounds

- pursued: we expect a floor stated in characters to be answerable with characters that render as nothing, and a text of twenty zero-width spaces clearing the promoted refusal is what showed it
