---
schema_version: 1
id: "iss-2608301455384564"
slug: "the-grounds-floor-s-stated-limits-mark-free-text-in-an-unnam"
severity: "nitpick"
category: "observation"
source: "user-observation"
found_during: "itd-179-round-4-recheck"
found_at: "internal/core/grounds/grounds.go"
deferred_after: "v0.11.1"
deferral_reason: "ruling F owed to the product thinker: internal/core/grounds/grounds.go states both limits in its comments (a scriptio-continua script it does not name still reads as one word, and MinTextLetters demands twenty letters whatever the script). Accept that as the stated limit, or re-weight the floor. (re-checked at e792a2314 by lane drainDQ3, run A, 2026-09-29)"
---

the grounds floor's stated limits: mark-free text in an unnamed scriptio-continua script is refused, and the letter floor is far heavier for CJK than for English

Amended 2026-08-30, after a46f37e3 moved what the second clause names. The
floor it described counted every rune against `MinTextLen`; the floor now
counts letters against `MinTextLetters`, so the constant and the refusal the
original wording pointed at no longer exist under those names.

The asymmetry the record exists to state is unchanged and slightly sharper.
Twenty ideographs carry a sentence where twenty Latin letters carry three or
four words, and a script's own punctuation now counts toward the floor in
neither case. The first clause is untouched and still true.

Kept as one observation rather than reclosed and refiled: the limit being
recorded is the same limit, restated in the unit the floor now measures.
