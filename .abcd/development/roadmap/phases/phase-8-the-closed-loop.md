# Phase 8 — The closed loop

## Expectation

By the end of this phase, the person a product is for can tell the system it is
wrong, and the system knows what they mean.

Today the record runs one way. A product thinker says what should exist, agents
build it, an audit judges the criteria, and the trail stops. What the product
thinker learns by *using* the thing, which is the only test that matters to
them, has nowhere to go and nothing to attach to. A concern recorded against a
delivered promise is a prediction nobody ever scores.

This phase closes that circuit. A product thinker who is not an engineer, who
does not read the code and does not sit at a terminal, reports that something
does not match what they expected. That report attaches to the promise it came
from, so a recorded concern can be marked as having come true. The facilitator,
which is a machine by default, traces it back through the verdict, the work, and
the conversation where the decision was reached. And before any of that, the
product thinker has written down how this could be wrong and what would show it,
so the acceptance they gave was an act rather than a signature.

## Milestone

**A report from use lands back on the promise that predicted it.**

## Phase Acceptance

- A stop names its addressee and its question as data, carries the options that
  would answer it, outlives the session that raised it, and is answerable later
  by whatever picks it up.
- A product thinker's report from use attaches to the promise it came from, and
  a concern recorded against that promise can be marked as having come true.
- A decision in a record points at the passage of the redacted transcript where
  it was reached, and the pointer survives redaction rather than being computed
  against text nobody stored.
- A promise carries a written list of how it could be wrong and what would show
  it, authored by the product thinker, before it is accepted.
- An acceptance names who gave it and the verification rung it rested on, and an
  unaccepted promise says so rather than staying silent.

## Scope

In:

- The stop-and-verdict protocol and its addressee data (itd-169).
- The report from use, and its attachment to the promise that predicted it
  (itd-170).
- The pointer from a decision back to the conversation that reached it
  (itd-171).
- The defeater list a product thinker authors, and the acceptance that rests on
  it (itd-175).
- The product thinker's own surface (itd-167), which is what makes the milestone
  true for a product thinker who is **not** also the facilitator. Under
  [adr-2609151528057131](../../decisions/adrs/2609151528057131-abcd-owns-the-product-thinker-s-surface.md)
  abcd owns that surface. The milestone is demonstrable without it through the
  mediated path, where a facilitator relays; it is not demonstrable for the
  target user without it.

Out:

- How the system words itself to a person, which is Phase 7's milestone rather
  than this one (itd-168, itd-176).
- How strong the checking is before an acceptance, which is a separate
  capability (itd-173) that this phase records the result of rather than builds.
- Who answers for delivered work, which is open in
  [rfc-3](../rfcs/rfc-3-the-facilitators-role-and-responsibility-for-generated-code.md)
  and is settled by decision rather than by code.

## Dependency rationale

The internal order is the circuit's own: a stop must exist as a durable
addressed artefact before it can be answered elsewhere, a decision must point at
its conversation before a report can be traced back through it, and a defeater
list must exist before an acceptance can rest on anything.

Phase 7 comes first. Its milestone is that every word abcd says to a person
earns its place, and this phase sends far more words to a person who is not an
engineer. Closing the circuit before the words are right would deliver a report
channel nobody can read. **Phase 7 is drafted and not yet in the record; landing
it is a prerequisite for this phase and is tracked separately.**

## Open questions

- Whether the product thinker's surface belongs in this phase or its own. It is
  the largest single piece of work here and the only one that is not a record
  change, and the milestone is demonstrable without it through the mediated
  path.
- What closes a concern. A report that a concern came true is clear; a promise
  that was never complained about is not obviously the same as one whose concern
  proved unfounded.
- Whether a report from use is one kind of thing or several. A mismatch with an
  expectation, a new wish, and a defect all arrive through the same channel and
  may not want the same treatment.
