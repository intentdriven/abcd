---
id: itd-171
slug: a-decision-recorded-in-an-intent-an-adr-or-the-decision-log
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
promoted_from: iss-2608290819228175
---

# A decision points back to the conversation where it was reached

## Press Release

> _Seeded by promotion from iss-2608290819228175. Expand into the full press-release narrative before planning._

## Why This Matters

A decision recorded in an intent, an architecture decision record, or the decision log should point at the passage of the redacted transcript where it was actually reached, so the reasoning behind a record is recoverable rather than lost with the session that produced it. The record is the floor of recoverable theory, not the theory itself.

This is the trace the facilitator needs when the product thinker reports that the product does not match what they expected. Because the facilitator is a machine by default, the trace has to be machine-followable rather than merely readable by a person who was there.

The store is closer to this than it looks. A transcript record already carries a session identifier, a content hash, and a capture timestamp, so the session is already addressable. What is missing is granularity inside one transcript: there is no per-turn anchor, so a link can name the session but not the passage.

Two constraints rule out the obvious approach. A line number is the wrong anchor, and this repository already refuses that shape elsewhere. And redaction rewrites spans in place, so any offset computed against the raw transcript is invalid against the stored redacted one: an anchor must be derived from the redacted artefact or be content-addressed rather than positional.

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
