---
schema_version: 1
id: "iss-2608301212423009"
slug: "the-comment-authorising-the-promoted-grounds-refusal-misstat"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "itd-179-round-2-ruthless"
found_at: "internal/core/intent/ready.go"
resolution: "the comment at internal/core/intent/ready.go and spc-57's Staging paragraph now state the measured flip — 10 of 66 planned records carry an entry, 56 fail the check, 36 were READY before and are NOT READY after"
impact: internal
resolved_by:
  intent: "itd-179"
---

the comment authorising the promoted grounds refusal misstates the corpus state it rests on

Found by the round-2 adversarial ruthless review of build/itd-179.

`internal/core/intent/ready.go:285-289` reads: "The staging that put recording
before refusing is spent: the `planned/` bucket carries entries, so the gate no
longer arrives as a wall of pre-existing failures."

Measured against the branch tip by the reviewer: 10 of 66 planned records carry
an entry, 56 fail the grounds check, and 36 of those were READY before this
branch and are NOT READY after. The security reviewer's independent corpus run
agrees on the shape (126 records across all families fail the check, all 126
remediable in one command). So a maintainer reading this justification
concludes the flip is near-free, when it moves a third of the planned bucket.

spc-57's own Staging paragraph states the same precondition — "the refusal is
promoted in a second commit once the `planned/` bucket carries entries" — and
that precondition is not met.

The forward-only decision itself is RULED (orchestrator, logged for the
facilitator to reverse) and is not what this record is about. This is about its
description: the code and the spec should state the measured number, so the
decision is visible as the deliberate one it is.

Remedy: state in both places that 36 previously-ready planned intents flip to
NOT READY, each recording its grounds when next picked up.

## Grounds

- pursued: we expect a justification that describes the corpus wrongly to be read as licence for a near-free change, and a maintainer reading spent staging where a third of the planned bucket flips is what would show it
