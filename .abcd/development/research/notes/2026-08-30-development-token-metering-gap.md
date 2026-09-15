# Development token metering: a gap, parked

**Date:** 2026-08-30
**Status:** PARKED, not filed. Proposed for a future phase; see "Why this is not
an intent yet" below.

## The gap

Nothing in this repository records what a development act actually cost in
tokens. Two records come near it and neither closes it.

`itd-29` (planned, autonomous-run resilience) carries a pre-flight budget check
that ESTIMATES token cost — `tasks × ~80k tokens × iteration count` — against
remaining quota, and refuses to start when the arithmetic fails. It also
promises mid-run telemetry naming the share of a daily budget consumed.

`itd-98` (draft, solo vs duo) lists "wall-clock and token cost" in its metric
set, but only inside that one experiment's comparison report.

So the estimate exists and the measurement does not. `itd-29` predicts forward
from a hardcoded per-task constant and never learns whether it was right: there
is no feedback loop, no ground truth, and therefore no way for the constant to
be corrected. It can be wrong indefinitely without anything noticing, which is
the shape [enforcement-claims-are-facts](../../principles/enforcement-claims-are-facts.md)
objects to — a number stated as if it were established.

## Why it is worth closing

The evidence is this cycle. Answering a straightforward question — what has the
review discipline cost — required hand-tallying eight subagent figures out of
transient task notifications, and the result was good enough to change how the
remaining phase is run
([the cost calibration](2026-08-30-review-round-cost-calibration.md)).

That tally is not reproducible. The numbers lived in one session's
notifications, they are gone with it, and the session lost to a machine crash
the same day took its own figures with it. A signal valuable enough to redirect
a phase should not depend on someone happening to read it before the window
closes.

Two consumers are already waiting: `itd-29`'s estimator wants actuals to correct
itself against, and `itd-98`'s comparison report wants the metric it already
names.

## Proposed routing, hand-run but unconfirmed

- **Capability → intent.** One: a development act records its actual input and
  output tokens, so estimates are corrected by actuals and retrospectives have a
  substrate. Typed link `refines` itd-29, which supplies the ground truth its
  estimator lacks; itd-98 becomes a consumer rather than a duplicate.
- **Trust rule → ADR.** Probably none needed. Invariant 15 already enumerates
  `session-separation evidence (metadata only, never bodies)` as a sanctioned
  reader of the transcript store, and token counts are metadata of exactly that
  kind. Extending an existing carve-out beats minting a rule beside it.
- **Stance → principle.** None needed;
  [prefer-the-experiment-to-the-inference](../../principles/prefer-the-experiment-to-the-inference.md)
  already holds it.
- **Plumbing → brief.** Which store the counts land in. The history store is
  already keyed on the repo's root-commit SHA and is the obvious candidate.

Verdict if filed as it stands: FILE-AS-IS on one intent, with two
consolidations. That would be the third consecutive hand-run whose main result
is finding homes already occupied, which is itself worth noting when the
decomposition method is next assessed.

## Why this is not an intent yet

The maintainer's standing authorisation for this cycle covers the cold-reading
workstream's own filings and stops at anything touching records outside them.
This is a general abcd capability, not cold-reading work, so filing it under
that authorisation would be overreaching it. Parked deliberately, on the
maintainer's instruction, rather than filed and left for someone to question.

**Trigger:** the next phase's planning. The routing above is a proposal to
confirm or overturn, not a decision already taken.

## A sibling, parked the same day

[A size classification for specs and tasks](2026-08-30-spec-size-classification-gap.md)
is the other half of this. Metering records what a development act COST;
classification predicts what it WILL cost. Either alone is weak — an estimate
with no ground truth is `itd-29`'s hardcoded constant, and actuals with no
scheme to test are a log nobody queries. They should be designed together.

