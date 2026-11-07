---
id: itd-2611041410013155
slug: when-the-author-of-a-promise-has-said
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
refines: [itd-170, itd-175]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A promise's own test of itself comes back when it falls due

## Press Release

> When the author of a promise has said what would show it wrong and when that could be known, abcd brings that statement back at that moment. Whatever has come due is brought back together, not one prompt per promise, and the product thinker's dated judgement stays on the promise. The moment can be a date or an event. A promise with no such statement says so, nothing requires one, and abcd reports judgements and counts, never a score of which promises paid off.

## Why This Matters

Typed links: `refines`
[itd-170](itd-170-what-the-product-thinker-reports-after.md) (a report from use
attaches to the promise that predicted it) and
[itd-175](itd-175-the-product-thinker-writes-down-how-this.md) (the product thinker
writes down how a promise could be wrong). Those two carry the reading and the
naming; this draft adds the moment that joins them. The product thinker chose a
draft of its own over adding this to either, so planning should treat the three as
one lifecycle and add no fourth place to state the same thing.

Shipping is not the end of a promise, and abcd treats it as if it were. An intent
can already say what would show it wrong
([adr-51](../../decisions/adrs/0051-intents-declare-mechanism-and-scope.md)), but
nothing brings that statement back. [itd-53](../shipped/itd-53-review-queue-auto-drain-fidelity-gate.md)
predicted it would be shown wrong if its owed count had not moved a month after
shipping; a search of its audit notes on 2026-11-04 found no such reading. The need
also appears by hand: [itd-2611011325421304](../planned/itd-2611011325421304-a-landing-runs-the-tests-its-change-can.md)
carries a criterion read over ten consecutive landings against a baseline.

Most ideas miss what they were meant to move: [Kohavi and colleagues](https://exp-platform.com/Documents/2012-09%20ACMRecSysNR.pdf),
from controlled experiments at Microsoft, put the share at 60 to 90 percent. Those
were large populations, so the figure does not carry over directly to a prototype
with few users, and that is why the reading here is a person's dated judgement and
not a measurement.

## What's In Scope

- **A due moment**: a date or an event the author has stated, read from the
  statement the promise already carries.
- **Bringing it back in a batch**: whatever is due arrives together. The record's
  history warns against one prompt per promise: its surface page calls a
  retrospective per intent a chore nobody finishes.
- **A dated judgement on the promise**: the product thinker's, kept after any later
  fix, using the states itd-170 defines. "Not yet known" is a valid answer.
- **A named absence**: a promise with no such statement says so.
- **Counts on request**: judgements and counts with their sample size.

## What's Out of Scope

- A required section. adr-51 kept these sections nullable because required ones
  turned into boilerplate, and iss-2609091009111294 turned similar refusals into
  advisory rows. Disciplines and plumbing have no moment of use at all.
- A score or league table of which promises paid off. One person usually names the
  test and judges it, and [DORA warns](https://dora.dev/guides/dora-metrics-four-keys/)
  against metrics as goals.
- abcd measuring use itself. A product's own figures shown at the due moment were
  discussed as a later option and are not part of this draft.

## Mechanism

- We expect statements to get read once they arrive in a batch at the moment they
  can be answered, because the shipped list of owed reviews
  (itd-2609150819445595) bets that a visible debt gets paid, and because only one
  retrospective is on file (v0.12.0) despite a nudge at each cut, so a per-release
  nudge alone does not do it. Shown wrong if due statements pile up unjudged, or if judgements
  come back as a reflex "worked" with nothing behind them.

## Scope Conditions

- Promises whose author stated a test and a moment. Everything else only says it
  has none.
- A judgement is one person's dated opinion, on a sample that may be one, and makes
  no causal claim.
- Someone is still using abcd in that repository when the moment arrives, so the
  statement has somewhere to come back to.

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

- **How abcd reads a due moment without a new field.** The moment lives in prose
  today, and a machine has to know when it arrives. Whether a light convention in
  the existing statement can do that, or the no-new-field aim has to give way, is
  the first question for planning.
- **Where it comes back.** The status board, the release's retrospective seed, or
  both.
- **The write path.** A dated verdict on a shipped intent, close to the existing
  condition dispositions (itd-2609020625405251) but occasioned by use.
- **"Not yet known".** When it is asked again, without nagging, and whether it can
  ever close by itself.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
