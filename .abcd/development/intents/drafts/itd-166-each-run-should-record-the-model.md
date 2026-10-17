---
id: itd-166
slug: each-run-should-record-the-model
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2608290811463906]
---

# A run records what it was actually run with

## Press Release

> _Seeded by promotion from iss-2608290811463906. Expand into the full press-release narrative before planning._

## Why This Matters

A model identifier does not determine behaviour, so a record that names only the identifier does not make a run interpretable or reproducible. Whatever the model exposes belongs alongside it: reasoning or thinking depth, sampling parameters, a seed where one exists, the context-window variant, and any speed or quality mode.

Two runs stamped with the same identifier can differ on all of these, so a record naming only the identifier invites a false comparison between them. An audit verdict, a benchmark, or a regression blamed on a code change may in fact be a settings difference. That matters most where a verdict is treated as evidence.

This is facilitator-tier diagnostic data and it never surfaces to the product thinker: its use is the trace backwards when a report arrives that the product does not match what was expected. That also settles where it lives, since a product-thinker-facing record carries none of it.

The hard constraint is that a model cannot reliably introspect the settings it runs under, so the dispatching layer has to supply them. Any design that asks the agent to state its own settings inherits the truthfulness problem the attribution gate already has.

Widened by the decomposition of itd-2610161733316518 (2026-10-16, routing confirmed by the product thinker): the record has two sides. The **asked** side is what the dispatching layer requested: model and snapshot, provider and any provider pin, sampling, seed, effort, context, and the prompt and tool definitions by hash. The **served** side is what actually answered: the model and upstream provider the response names, any fallback taken, token counts including reasoning and cached, and cost where the provider reports it. A local model adds the weight file's fingerprint, its quantisation, the runtime and its version, and the served context. Whatever a route cannot expose is written down as unpinned or self-reported rather than left out (see the principle that a model run is re-runnable, not reproducible).

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

1. **Who reads it.** This record keeps the settings at facilitator tier, never surfaced to the product thinker; the model-matrix run's report shows the product thinker a comparison whose trust rests on them. Settle whether the report shows a drift flag only, with the record itself staying facilitator-tier.

## Typed Links

- **blocks `itd-2610161733316518`** (one feature built across a model matrix): the run's manifest is this record, and nothing it compares is interpretable without it.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
