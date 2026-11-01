---
id: itd-170
slug: what-the-product-thinker-reports-after
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
---

# What the product thinker reports after using the product finds its way back to the promise that predicted it

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

The product thinker's real test is use. They see whether the product matches what they expected, and that judgement arrives long after the verdict that predicted it.

Nothing carries it back. An audit verdict of met with concerns is a prediction: it says the promise was kept and names what might yet bite. Whether the concern materialised is answered only by the person using the thing, and today their answer has nowhere to go and nothing to attach to. The prediction and its outcome never meet, so the record cannot say whether its own concerns were worth recording.

Closing that edge is what makes the audit worth more than ceremony over time. It also gives the facilitator something to trace: a report that the product does not match expectation should lead back to the promise, the criterion, the verdict, and the work that produced it.

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [verification-assurance SOTA note](../../research/notes/2026-10-23-verification-assurance-sota.md).

- **Record the prediction with its confidence, then compare, because memory rewrites it.** The decision-journal template records the expected outcome, the reasoning and the probability assigned, and later what happened: "It's hard to remember what we knew at the time"; the journal "helps combat this by recording what you knew and what you thought at the time" (Farnam Street, Decision journal). An audit's "met with concerns" is that prediction; this intent is its review.
- **Signals are named before the test and read against that naming afterwards** ("We will know we have succeeded when ...", stated "before conducting the test to reduce biased interpretations", ThoughtWorks on hypothesis-driven development).
- **New information is routed back to the decision it bears on, not filed as a new item.** The opportunity solution tree exists so that when "a failed experiment" arrives, "teams can use their tree to quickly revisit those decisions" (Product Talk glossary). This is the ruling's "not a bare new item".
- **A link that carries a fingerprint of its target knows when the target moved**; the [traceability note](../../research/notes/2026-10-04-brief-to-record-traceability-sota.md) already recorded Doorstop's link fingerprint and the "suspect link" of requirements tools, with write-back as part of the act that changes the target.
- **"Re-open on recurrence" is the bot convention, and the ruling's opposite stance is the better record.** A WebKit review bot "will re-open if the flake re-occurs"; reopening erases the outcome this intent exists to learn. SARIF's `baselineState` (`unchanged`, `updated`, `absent`) is a vocabulary for a state that stays without reopening.
- **Locally**, a `MET_WITH_CONCERNS` concern has a rationale and evidence pointers but no id of its own; scope conditions have persistent ids and itd-175's defeaters would too. A report from use has nothing to attach to until a concern is addressable.

**Path 2: native, shaped to the decision journal and the pre-stated signal.** Each concern gets an id at ingest (the versioned fingerprint convention of itd-165); a report from use is a record naming the intent, the criterion and the concern id, carrying a state the product thinker sets (came true, did not, not yet known) that stays set; a fix is recorded against it with an edge and the concern keeps its state, as ruled. One report shape serves an auditor's concern, a defeater of itd-175's, and a scope condition's disposition, which is the seam. A calibration view (how many recorded concerns came true) waits for a corpus, as itd-165 defers its ratchet. No dependency. The independent fit-challenge runs at plan time.

## Open Questions

**Decided 2026-08-29 (product thinker).** A report from use attaches to the promise it came from, so a recorded concern can be marked as having come true. Not a bare new item, and not a reopening of the original promise.

- (SOTA 2026-10-23) Where does a report from use get written on the product thinker's machine (a record in the working tier, or the local tier promoted on acceptance), given that it is theirs and is never posted on their behalf?
- (SOTA 2026-10-23) Several reports on one concern: counted on the record as recurrence, or one record per report linked to the concern?

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

**Decided 2026-08-29 (product thinker).** A concern reported as having come true stays marked that way, and the fix is recorded against it rather than replacing it. Nothing disappears on being fixed: the point of recording a concern is to learn later whether recording it was worth doing, and a record that closes cleanly cannot answer that.
