---
id: itd-175
slug: the-product-thinker-writes-down-how-this
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
---

# The product thinker writes down how this could be wrong, and what would show it

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

The product thinker writes down how this could be wrong, and what would show it

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [verification-assurance SOTA note](../../research/notes/2026-10-23-verification-assurance-sota.md).

- **Assurance cases record doubts as first-class elements and record the decision to accept what cannot be resolved.** Assurance 2.0 assesses confidence from the positive side (evidence and argument), the negative side (doubts "typically expressed as defeaters", which developers "should vigorously explore" and "should record ... and their resolution" while guarding against confirmation bias) and residual doubts, where "It is crucial however that these judgments are conscious ones and that they are recorded in the assurance case" (Bloomfield and Rushby, Assessing Confidence with Assurance 2.0). The ruling that an absent list is named at acceptance is this stance.
- **A defeater is a recorded doubt that can grow into a sub-case confirming or refuting it**, kept "for the benefit of subsequent developers and assessors" (Bloomfield, Netkachova and Rushby, Defeaters and Eliminative Argumentation in Assurance 2.0).
- **The premortem is the product-practice form**: imagine the work "has already failed" and list the reasons; it works by "making it safe for dissenters who are knowledgeable about the undertaking" (Klein, Performing a Project Premortem, summary). Klein's later account reports a 2010 study where the premortem group "showed the greatest reduction in (over)confidence"; the study itself could not be retrieved.
- **The "30 per cent more reasons" figure is not in the 1989 abstract** it is attributed to, which reports that temporal perspective had "little influence" and that outcome certainty changed the kind of explanation given. It should not appear in a record.
- **The signal is written before the test.** Hypothesis-driven development's "We will know we have succeeded when <we see a measurable signal>" is to be "stated before conducting the test to reduce biased interpretations of the results" (ThoughtWorks); continuous discovery says "define success upfront" and "draw a line in the sand", with assumptions in five kinds: desirability, feasibility, usability, viability, ethical (Product Talk, Assumption testing).
- **Locally**, the intent record already holds a facilitator-written Mechanism ("we expect X because Y; shown wrong if") and Grounds ("wrong if"), and scope conditions carry persistent ids the fidelity verdict dispositions. Missing is the product thinker's own list, in their words, with an id per entry and a named absence.

**Path 2: a native defeater list on the intent, shaped to Assurance 2.0's defeater and hypothesis-driven development's pre-stated signal.** Each entry: how this could be wrong and what would show it, stamped with a persistent id like a scope condition, so the verdict can disposition it and a report from use (itd-170) can mark it as having come true; the acceptance says "no list written" when declined, as ruled. Formal notation (GSN, SACM) is not worth adopting for a one-page record. No dependency. The independent fit-challenge runs at plan time.

## Open Questions

**Decided 2026-08-29 (product thinker).** A promise can be accepted with no list written, and the acceptance says so. Requiring one would block an acceptance on a day when the product thinker simply wants to say yes; hiding its absence would leave the acceptance resting on nothing while looking identical to one that does not. Naming the absence makes skipping it a stated choice.

- (SOTA 2026-10-23) How does the product thinker's list differ from the facilitator's Mechanism and Grounds, which already say "shown wrong if"? The obvious answer is voice and author (the record says who wrote each), but the plan should state it so the two do not overlap.
- (SOTA 2026-10-23) Does each entry get a persistent id at acceptance, so the fidelity verdict and a later report from use can disposition it the way scope conditions are?

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
