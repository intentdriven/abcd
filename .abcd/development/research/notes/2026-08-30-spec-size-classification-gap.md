# A size classification for specs and tasks: parked

**Date:** 2026-08-30
**Status:** PARKED, not filed. For a future session to design and file.
**Siblings:** [the token metering gap](2026-08-30-development-token-metering-gap.md)
(actuals) and [the review-round cost calibration](2026-08-30-review-round-cost-calibration.md)
(where the evidence came from).

## What is missing

There is no size class for a spec or a task. Nothing lets an intent be called
S, M, L, XL or XXL before it is built, so nothing can be scheduled, budgeted, or
compared against what it turned out to be.

`itd-29` is the nearest thing and it assumes the gap away: its pre-flight budget
check estimates `tasks × ~80k tokens × iteration count`, which is a FLAT
per-task constant. This phase's evidence says that constant cannot exist.

## The evidence, from cycle 1

Thirteen capability intents, nine of them built and measured:

| intent | commits | review rounds | diff |
|---|---|---|---|
| itd-179 | 50 | 29 | +6,289 / 111 files |
| itd-183 | 47 | 15 | +7,275 / 64 files |
| itd-189 | 51 | 11 | +5,293 / 57 files |
| itd-180 | 29 | 6 | +5,469 / 48 files |
| itd-177 | 22 | 3 | +3,143 / 97 files |
| itd-188 | 15 | 4 | +1,089 / 16 files |
| itd-182 | 14 | 0 | +686 / 29 files |
| itd-178 | 11 | 1 | +1,948 / 46 files |
| itd-181 | 5 | 0 | +1,091 / 10 files |

A flat constant would have to serve both itd-181 at five commits and itd-179 at
fifty. It cannot.

## The finding that should shape the scheme

**Depth does not track volume.** itd-177 touched 97 files and converged in three
rounds. itd-188 touched 16 files and took four. itd-181 and itd-182 took ZERO
rounds — right the first time — while carrying 1,091 and 686 lines respectively.

Rounds track difficulty; lines do not. So a size class keyed on expected diff
size would mis-sort this corpus, and keying it on files touched would mis-sort
it worse.

What the deep ones have in common is not size but **the number of ways they can
be quietly wrong**: itd-179 and itd-189 are both validation surfaces where a
predicate can accept something that means nothing, and itd-183 is a floor that
must recognise a construct however it is spelled. The shallow ones write
something once and are done.

That is the same variable the review-cost calibration arrived at from the other
direction, where review spend turned out to be driven by how much the reviewer
must PROBE rather than by how much it must read.

## Constraints any scheme must respect

- **No time estimates.** The roadmap is intent-driven and the press-release
  format carries priority rather than dates; a scale denominated in days or
  hours contradicts that. The classes have to be in some other unit — expected
  review rounds is the candidate this evidence supports, and surface count or
  "distinct ways to be quietly wrong" are the others worth trying.
- **It must be checkable against actuals**, or it becomes another unfalsifiable
  constant like `itd-29`'s. That requires the token metering parked in the
  sibling note, and the two should probably be designed together: one predicts,
  the other measures, and neither is worth much alone.
- **It must survive being wrong.** The point is a distribution, not a promise —
  a scheme that is embarrassing when an L turns out XXL will simply stop being
  used.

## What a future session should do

Design the classes, state what each is denominated in, back-fit them against the
nine measured intents above, and say plainly which ones the scheme mis-sorts.
Then file it, or record why not.

This is deliberately NOT filed as an intent. It is a general abcd capability
rather than cold-reading work, and this cycle's standing authorisation stops at
records outside the workstream's own filings.
