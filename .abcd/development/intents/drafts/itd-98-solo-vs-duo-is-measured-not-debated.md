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

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in
the [models, prompts and measurement SOTA note](../../research/notes/2026-10-23-models-prompts-measurement-sota.md);
repeats, power, blind review and the trial record are in the
[model-matrix note](../../research/notes/2026-10-16-model-matrix-sota.md).

- **Pair versus solo: it depends, and effort flips the answer.** Across
  fifteen experiments, pairing moderately raised quality, moderately cut
  duration and moderately increased effort, depending on task complexity and
  the pair's expertise (Dybå, Arisholm, Sjøberg, Hannay and Shull, 2007; the
  2009 journal meta-analysis was not opened). Duo's human minutes are effort
  and must be counted, or the comparison omits half of one arm's cost.
- **The best human-plus-AI trial measured the opposite of what people
  felt**: 16 maintainers, 246 issues randomised per issue, AI-allowed issues
  19 % slower against a predicted 24 % and a felt 20 % speedup (METR, 2025;
  the blog post cited gives no interval).
- **The follow-up could not staff the no-AI arm**: 30–50 % of developers
  withheld tasks from it, pay and multi-agent timing confounded the rest, and
  neither cohort's interval excluded zero (−18 %, −38 % to +9 %, n = 10;
  −4 %, −15 % to +9 %, n = 47); METR lists fixed-task experiments and
  developer-level randomisation as alternatives (METR, February 2026). The
  "who plays duo's human" question is the binding constraint; a declared
  human budget per arm is the honest form.
- **The one vendor RCT is one task**: 55.8 % faster building an HTTP server
  (Peng et al., 2023).
- **Running the same item in both arms is a crossover, and carryover is the
  threat nobody analyses**: of 67 crossover experiments in software
  engineering, carryover was modelled in about 3 % (Frattini, Fucci and Vegas,
  2024). Randomise items to arms; twin only where an item cannot be split;
  name carryover wherever an item did run twice.
- **Escalation quality is a two-by-two**, not a rate: the reliance
  literature's constructs are fragmented across three views (Raees and
  Papangelis, 2026), and the field's common objective form, scoring
  over-reliance (accepting a wrong recommendation) and under-reliance
  (rejecting a right one) separately, is its convention rather than that
  paper's finding. Report both error cells, with a judge outside
  the run proposing the "should have escalated" column.
- **Power**: three-run comparisons rank the weaker arm first in 28–44 % of
  resamples (matrix note); the report states the smallest difference it could
  detect, as the matrix intent already requires.

**Path 2**: the paired run is a matrix run whose cell is the mode (the matrix
intent's decision 11 names this reuse), with its trial record, repeats
sizing, blind review and per-item divergence listing; two metrics join the
record for this use, human minutes per arm and tokens per arm (from
itd-2609292107351737's metering), and escalation is scored as the two-by-two.
Repeats sizing and a minimum detectable difference contradict the v1
out-of-scope line on statistical machinery; that line stands until the
product thinker revisits it, and a v1 that keeps honest counts still states
its n and that no difference below the observed spread can be read. The seam
is the trial record's mapping onto an Inspect sample and a Harbor trial. No
dependency. The independent fit-challenge runs at plan time.

## Open Questions

- (SOTA 2026-10-23) Whether items are randomised to arms (METR's unit) with
  twinned pairs only as the exception, which answers the matched-backlog
  question below in favour of controlling carryover over controlling
  difficulty.
- (SOTA 2026-10-23) Whether human minutes per arm are declared as a budget
  before the run and counted as effort in the report, and who judges the
  "should have escalated" column when the maintainer was duo's human.
- (SOTA 2026-10-23) Whether the v1 out-of-scope line on statistical machinery
  (honest counts, no significance claims from n=1) is revisited to admit
  repeats sizing and a stated minimum detectable difference, or v1 keeps
  honest counts and states only its n and that no difference below the
  observed spread can be read.
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
