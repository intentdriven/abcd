---
id: itd-53
slug: review-queue-auto-drain-fidelity-gate
spec_id: spc-2609211930059886
kind: standalone
suggested_kind: standalone
reclassification_history: []
related_adrs: [adr-16]
routed_from: ["spc-33:I-D2"]
prd_path: null
severity: major
impact: additive
---

# A Shipped Intent No Longer Drifts Out Of Audit Just Because Nobody Ran The Review

> **Re-scoped on 2026-09-21** by the product thinker: one bounded command that pays the backlog, `abcd intent audit --owed`, run once by the autonomous run's first batch. The standing list is itd-2609150819445595 and the inline audit of every new lane is `abcd build`'s (itd-2609201916151817, itd-50); the boundary-hooked autodrain and the blocking gate this record first described are dropped. The press release and scope below are read through this paragraph and the Decisions section.


## Press Release

> **abcd closes the audit back-edge: when a specced block of work ships, its owed fidelity review actually gets run at a safe moment, and a standing gate surfaces any shipped intent whose review is missing or unmet — without ever blocking the autonomous loop.** Today abcd does the honest half: closing a spec moves its intent to shipped and enqueues a fidelity-review entry. But the review itself only runs when someone manually invokes it, so the queue can quietly accumulate owed reviews that nobody drains, and a shipped intent can sit with its acceptance never machine-checked. This intent adds an opt-in drainer that runs queued reviews at a safe boundary (after a loop, at a session edge, in a pre-commit or CI step — never inside the pure close hook), leaving entries deferred rather than failed when no review backend is reachable, plus a consistency gate that lists shipped intents whose latest review is absent or not-met. Enforcement, not just bookkeeping — and loop purity preserved.

> "We shipped a dozen specs and only later found half their intents had never actually been audited against what we built," said Iris, a product manager leaning on abcd's intent discipline. "The queue was doing its job recording that a review was owed. Nothing was paying the queue down. I want the owed reviews to drain themselves when a backend is around, and I want a single list of 'shipped but not really verified' so nothing slips."

## Why This Matters

abcd's intent discipline promises that shipped work is reconciled against the intention that justified it. The on-close lifecycle hook (deliberately a pure data function — no subprocess, no oracle dispatch, because it runs inside the autonomous loop) correctly only *enqueues* the review. That decoupling is right: running a blocking oracle call on the loop's critical path would add latency and, in headless mode, frequently hit an unreachable backend and stall. But the consequence is that "a review is owed" and "a review was run" have drifted apart, with nothing closing the gap.

The fix is not to make the close hook run the review — that would break loop purity and re-introduce the stall. The fix is a separate **drainer** that runs at a safe boundary where a backend is reachable, plus a **gate** that reports the truth of record. This keeps the clean seam (enqueue is pure; drain is a deliberate, backend-aware step) while making "shipped" mean "shipped and actually audited." The queue already carries the hard correctness machinery — a two-lock protocol guarding the audit-notes writeback, claim/drain/crash-stale recovery — so the drainer rides existing safety rather than inventing it.

## What's In Scope

- An opt-in `review.autodrain` configuration (default off) that, at a safe boundary (after an autonomous turn, at a session edge, or in a pre-commit/CI step — never in the pure close hook), drains pending fidelity-review queue entries by running them when a backend is reachable.
- No-backend behavior: entries stay `deferred` (not failed), so a headless run never blocks on an unreachable backend.
- A consistency gate / report that lists shipped intents whose latest fidelity review is missing or not-met — turning the back-edge from trigger-and-record into a surfaced gate.
- **Coverage vocabulary.** The gate reports each intent's consumption state in five words: **uncovered** (scheduled, no spec claims it), **covered-shallow** (a spec claims it but that spec's own obligations — implementation, tests, validation — are not all green), **covered-deep** (claimed and transitively green; the only state that may combine with the intent's own MET criteria into "done"), **orphaned** (a spec claims an intent that does not exist), and **unwanted** (a spec claims an intent that never asked for coverage: a draft, a superseded intent, or a discipline — disciplines are complied with, never consumed).
- Preservation of the close hook's purity: the hook still only enqueues; the drainer and gate are separate surfaces.

## What's Out of Scope

- **Making the on-close hook run the review inline.** Explicitly rejected — it breaks loop purity and re-introduces the headless stall. The whole point is to keep enqueue and run separate.
- **The headless backend-reachability fix.** The drainer needs a reachable backend to make progress; the in-process-oracle gap is itd-47's concern, upstream of this intent.
- **The facilitator-elected loop-toward-acceptance policy** (re-open work on not-met, terminal unachievable→replan). That rides on this drainer as substrate but is its own intent (itd-50).

## Scope Conditions

None stated.

## Mechanism

We expect the backlog of owed audits to clear once paying it is one bounded command, because the debt accrued only while each audit was a hand-run request and ingest; shown wrong if the owed count is unchanged a month after it ships.

## Acceptance Criteria

- **Given** `abcd intent audit --owed [--max <n>]`, **when** it runs, **then** it lists the owed audits oldest first and runs each as the host pass a single-intent audit uses, ingesting each verdict; the cap stops it and the summary names how many remain.
- **Given** no reviewer is reachable, **when** it runs, **then** every entry is left owed, not failed, and the summary says why nothing ran.
- **Given** a verdict, **when** it is ingested, **then** it lands exactly as a single audit's does (audit notes, receipt, scope-condition dispositions), and a not-met verdict on a long-shipped intent is captured, not fixed.
- **Given** a spec closes, **when** the close hook runs, **then** it still only enqueues; nothing starts a reviewer from it.

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that gave this intent its spec:

1. **One bounded command for the backlog**, run once by the autonomous run's batch 0; no boundary-hooked autodrain.
2. **The gate reports and never blocks**; the report is itd-2609150819445595's listing.
3. **Cost is bounded by `--max`** and by the host pass running one audit at a time.

## Open Questions

_None open; decisions 1 to 3 settle the three this record carried._

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._

## References

- Substrate for: **itd-50** (facilitator-elected loop-toward-acceptance +
  unachievable→replan) — that policy layer rides this drainer.
- Depends on / coordinates with: itd-47 (headless backend reachability — the
  drainer needs a reachable backend to make progress).
- Source: design discussion on the audit-loop enforcement design
  (a dated working-log entry, 2026-06-02, "should spec-close auto-run review" — resolved
  NO; add a drainer instead).
- Touches: the pure on-close lifecycle hook (spc-28) and the review-queue
  drain/claim machinery; the fidelity reviewer (spc-12) is the run target.

## Grounds

- pursued: the run's first batch audits every shipped-but-open intent and has no verb to run the audits with; we expect the owed count to fall to zero in that batch and stay near it once build audits inline; shown wrong if the owed count is unchanged a month after it ships
