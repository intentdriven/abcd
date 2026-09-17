---
id: itd-166
slug: each-run-should-record-the-model-settings-alongside-the-mode
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
promoted_from: iss-2608290811463906
---

# A run records what it was actually run with

## Press Release

> _Seeded by promotion from iss-2608290811463906. Expand into the full press-release narrative before planning._

## Why This Matters

A model identifier does not determine behaviour, so a record that names only the identifier does not make a run interpretable or reproducible. Whatever the model exposes belongs alongside it: reasoning or thinking depth, sampling parameters, a seed where one exists, the context-window variant, and any speed or quality mode.

Two runs stamped with the same identifier can differ on all of these, so a record naming only the identifier invites a false comparison between them. An audit verdict, a benchmark, or a regression blamed on a code change may in fact be a settings difference. That matters most where a verdict is treated as evidence.

This is facilitator-tier diagnostic data and it never surfaces to the product thinker: its use is the trace backwards when a report arrives that the product does not match what was expected. That also settles where it lives, since a product-thinker-facing record carries none of it.

The hard constraint is that a model cannot reliably introspect the settings it runs under, so the dispatching layer has to supply them. Any design that asks the agent to state its own settings inherits the truthfulness problem the attribution gate already has.

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
