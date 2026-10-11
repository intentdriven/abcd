---
id: itd-98
slug: solo-vs-duo-is-measured-not-debated
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-97]
severity: major
---

# Solo vs. Duo Is Measured, Not Debated

## Press Release

> _Facilitator-seeded draft (2026-07-25 role-ownership review) — the product
> thinker owns the final press-release framing._

> **Whether an automated facilitator matches a human one is answered with
> receipts, not opinions.** Point abcd at a matched backlog and it runs the
> same work twice — one arm solo, one arm duo — and returns a scored
> comparison: what shipped, what the gates refused, how often a human had to
> step in and whether those escalations were the right ones, and how the
> fidelity verdicts compare. Every number in the report traces to a receipt —
> a PR, a gate log, a verdict file — and every blank is loud.
>
> "We stopped arguing about whether solo was ready," said Carol, a product
> thinker evaluating abcd for their team. "The report showed solo landed nine of
> eleven items clean and escalated exactly the two it should have. That's not
> a vibe — it's a table I can take to Bob."

## Why This Matters

Solo vs. duo is a key axis on which abcd itself will be tested (maintainer
direction, 2026-07-25): the framework is designed while being built, and the
claim "the facilitator can be a mode" (itd-97) is only credible with a
measured comparison behind it. The measurement substrate has recorded homes —
the CLI eval harness (itd-75), effectiveness tracking (itd-17), and
benchmark-driven configuration (itd-64) — and the judging discipline already
exists: itd-81 (judge calibration) encodes the correlated-errors and
effective-votes findings any outcome panel must respect. What is missing is
the experiment itself: a paired-run protocol, a metric set, and a comparison
report with receipts. abcd-cli's own backlog is the first corpus — pure
dogfooding.

## What's In Scope

- **A paired-run protocol**: the same (or twinned) backlog executed once in
  solo mode and once in duo mode, under the same run contract, gates, and
  budget accounting.
- **A metric set with receipts**: items landed; gate refusals; escalation
  count *and* escalation quality (were the human handoffs the ones the role
  boundary demands?); fidelity-review verdicts; wall-clock and token cost.
- **An outcome judge panel under the itd-81 discipline** — independent
  votes, correlated-error check, effective-agreement reported.
- **The comparison report artefact** — one document, per-metric values, each
  citing its receipt; divergent items itemised (what solo did vs. what duo
  did), never averaged away.
- **First corpus: abcd-cli** — the experiment runs against this repo's own
  queue before any managed repo.

## What's Out of Scope

- **Shipping the mode itself** — itd-97.
- **Statistical machinery beyond honest counts** in v1 — no significance
  claims from n=1 paired runs; the report says what happened, not what
  generalises.
- **Host/model benchmarking** — itd-17 and itd-64 own model-level
  effectiveness; this intent compares *modes*, holding the rest fixed.

## Acceptance Criteria

- **Given** a matched backlog and both modes configured, **when** the paired
  run completes, **then** a comparison report lands with per-metric values
  each citing its receipts — no uncited number appears.
- **Given** the two arms diverge on an item, **when** the report renders,
  **then** the divergence is itemised per item rather than folded into an
  aggregate.
- **Given** the outcome panel scores quality, **when** the judges run,
  **then** the itd-81 calibration discipline applies and the report records
  effective agreement, not just raw vote counts.
- **Given** a metric cannot be measured for an arm, **when** the report
  renders, **then** the blank is named and loud — never silently zero.
- **Given** the first paired run on abcd-cli's own backlog completes,
  **when** the report is read, **then** it supports a recorded
  solo-readiness decision by the product thinker (adopt, hold, or rerun),
  filed as a dated decision.

## Open Questions

- **Matched-backlog construction** — the same items run twice risk
  contamination (the second arm inherits knowledge via the record); twinned
  item pairs risk unequal difficulty. Which bias is cheaper to control?
- **Who plays duo's human in the experiment** — the maintainer's time is the
  scarce input; does a scoped "human budget" per arm make the comparison
  honest?
- **The pass bar** — what does "solo is ready" mean (parity on landed items?
  zero missed escalations?) and who sets it (product thinker decision by
  design)?

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._

## References

- itd-97 (the mode this measures), itd-99 (plural thinkers — a later
  experiment axis).
- itd-75 (CLI eval harness), itd-17 (model-effectiveness tracking), itd-64
  (benchmark-driven config) — the measurement substrate.
- `.abcd/development/intents/disciplines/itd-81-judge-calibration.md` — the
  panel discipline the outcome judges must follow.
