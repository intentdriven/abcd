---
id: dcn-2609150000000002
slug: an-intent-owns-one-or-more-specs-and-it
date: 2026-09-15
---
2026-09-15 — An intent owns one or more specs, and it ships when its last spec closes (product thinker, ruling on iss-2609100508566552, where `spec close` shipped the intent unconditionally and a session meeting a half-delivered intent could only stop). An intent that has been thought through stands as written: a spec that delivers part of it is closed on its own terms, a new spec is minted for the remainder and attached to the same intent (spec closed X, spec open Y, intent still `planned/`), and the intent moves to `shipped/` on the close after which no open spec names it, with `--impact` demanded at that transition and no earlier. The rule is adr-2609151513118583 and invariant 17 of the brief; the build is owed and the issue stays open until it lands. Rejected: close still means ship and narrow the spec (falsifies a thought-through intent); a spec may close without shipping and the intent is shipped by hand (a step after the merge is the one that gets forgotten); split the intent at close.
