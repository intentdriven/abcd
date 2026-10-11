# A remedy is filed whole, built where it can be, measured where it cannot

**The rule.** Once a remedy for a larger problem is identified, every part of
it is written down in its proper home (issues, intents, decision records,
principles) before any of it is built. Then the parts that can be built now are
built, and each part that cannot yet be built is measured, so its record
carries the numbers a future run needs to build it or to drop it.

**Why.** A remedy found in the middle of other work is usually larger than the
part that gets done. The part that gets done ships; the rest lives in a
conversation and is gone when the conversation ends. A part filed with no
evidence is no better: whoever picks it up next starts by re-deriving what the
first session already saw, and a part whose value was never measured is either
built on faith or never built at all.

The 2026-10-11 investigation into CI load drew the shape. The research named
eight options to cut CI cycles. Some could be built that day (run the macOS
check once per change, in the merge queue; drop the redundant run on every
push to the default branch; one race-enabled test pass on macOS instead of
two). Others could not: turning off the up-to-date rule needed a ruling, and
raising the macOS concurrency limit needed a request to the forge. Measuring
before filing changed the second group. Six days of runs showed the up-to-date
re-run hit only 6 of 71 merged pull requests, so its option dropped from a
quick win to a small one, and the macOS job had grown from 13 minutes to a
median of 40, which made its two test passes the larger target.

**Bounds.**

- Filing comes first, but filing is not a substitute for building: a part that
  can be built in the same change is built, not filed and stepped over
  (`pre-existing-is-not-a-defence`).
- "Measure" means the cheapest reading that settles the record's open question
  (run counts, timings, an experiment), labelled with how it was taken and
  when; it is not a study. Where nothing cheap settles it, the record says
  what would.
- Each part goes to its own home, routed as `decompose-before-filing` routes a
  proposal; one omnibus issue holding every part is the conversation again.
- A part the person must decide is filed as a question for them, with the
  measurement beside it, never decided by the session that found it.

**Promotion.** No check ties a remedy's parts together, so a remedy filed in
half is invisible. A record of the remedy listing its parts and each part's
state (built, filed, measured, awaiting a ruling) would promote this. Related:
`decompose-before-filing`, `prefer-the-experiment-to-the-inference`,
`reality-is-filable`, `the-record-lands-with-the-act`.
