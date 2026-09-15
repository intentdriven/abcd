---
id: itd-50
slug: loop-toward-acceptance
spec_id: null
kind: standalone
suggested_kind: standalone
reclassification_history: []
related_adrs: []
routed_from: ["spc-33:I-D3"]
glossary_terms_used: [core/intent, core/spec, core/phase, core/oracle, core/lifeboat, interview/session]
grilled_at: 2026-06-02
blocked_by: [itd-53]
builds_on: [itd-44, itd-43]
severity: major
---

# The Audit Loop Drives An Intent To Acceptance — Or Calls For A Replan

## Press Release

> **abcd's fidelity audit stops being a report card and becomes a loop the facilitator can drive toward acceptance.** Today, when a shipped intent's delivered reality fails a criterion, the `intent-fidelity-reviewer` records a `NOT_MET` verdict and the voyage moves on — the divergence is logged, but nothing closes it. With this change, the facilitator elects, per intent, whether that intent should *loop toward acceptance*: a `NOT_MET` verdict re-opens the work and iterates against the same acceptance criteria until they read `MET`, bounded by a budget so the loop can never grind forever. When an intent genuinely *cannot* be met as written, the loop doesn't thrash — it terminates with an explicit `UNACHIEVABLE` verdict that summons the product thinker and facilitator to sit together and replan the intent. And only once the machine-checkable criteria all read `MET` is the product thinker invited to manually verify the intention — so a human is never asked to hand-test something the audit already knows is broken.
>
> "I used to get a wall of `NOT_MET` verdicts at the end of a phase and no idea which ones were 'nearly there, push it' versus 'we were wrong about this, let's rethink it,'" said Carol, a tech lead. "Now I flag the intents that matter as loop-to-acceptance, and abcd drives them to `MET` for me. The ones that can't be met surface as a replan invitation, not a red mark I have to chase down. And I only get pulled in to manually verify the *intention* once the machine says the floor is solid — my time goes to judgment, not to re-testing broken work."

## Why This Matters

abcd's headline discipline is the audit loop: the dotted edge that compares delivered reality back against an intent's acceptance criteria. But in its current form the edge is **trigger-and-record, not enforcing**. When a spec closes, abcd moves the intent to `shipped/` and enqueues a review (spc-28); the `intent-fidelity-reviewer` (spc-12) emits per-criterion verdicts (`MET` / `MET_WITH_CONCERNS` / `NOT_MET` / `INCONCLUSIVE`) and writes them to the intent's `## Audit Notes`. A `NOT_MET` is faithfully recorded — and then nothing happens. The loop detects the miss but does not close it.

This leaves two real gaps:

1. **No way to drive a divergence to resolution.** At the *implementation* grain, abcd already has a fix-loop — the autonomous harness iterates on a `NEEDS_WORK` review until it ships. But that loop stops at the spec boundary; it has no equivalent at the *intent* grain. A facilitator who wants "keep working this intent until its acceptance criteria are met" has to drive it by hand.
2. **No terminal "we were wrong about this" state.** The verdict enum has `NOT_MET` (diverged — implicitly retry-able) and `INCONCLUSIVE` (couldn't determine), but nothing that says *"we tried, and the intent as specified cannot be achieved."* Without that state, a loop-toward-acceptance would have no safe exit — it would either grind forever or silently give up. The honest outcome when an intent can't be met is not a retry and not a shrug; it's a **replan**, and replanning an intent is a human act that needs both the product thinker (who owns the *why*) and the facilitator (who owns the *how*).

There is also a sequencing waste today: manual verification, when it happens, isn't gated on machine-acceptance. A product thinker can be asked to test an intention whose machine-checkable criteria are already failing — spending human judgment on work the audit already knows is broken. Manual verification should be the *last* gate, opened only once the loop has reached `MET`.

This intent is **project-agnostic**: every abcd project ships intents whose delivered reality can diverge from their acceptance criteria, every one of those projects benefits from being able to drive a divergence to resolution or convert it into a replan, and every one benefits from reserving the product thinker's manual verification for intentions whose machine floor is already solid.

## What's In Scope

### Three audit-loop modes, facilitator-elected per intent

- **`record-only`** (the current behaviour, and the default) — `NOT_MET` is written to `## Audit Notes`; no re-work is triggered.
- **`loop-to-acceptance`** — a `NOT_MET` verdict re-opens the linked work and iterates against the same acceptance criteria until they read `MET`, bounded by a max-iteration budget. Mirrors the implementation-grain `SHIP`/`NEEDS_WORK` fix-loop, lifted to the intent grain. The facilitator sets this mode per intent (the intents that genuinely warrant the iteration cost).
- The mode lives in the **policy / drainer layer** (riding on the review-queue drainer), never in the pure on-close lifecycle hook — loop purity (no subprocess, no oracle dispatch in the hook) is preserved.

### A terminal `UNACHIEVABLE` outcome + replan invitation

- A new terminal audit outcome distinct from `NOT_MET`: *the intent as written cannot be met*. It bounds the loop (a loop-to-acceptance intent that exhausts its budget or is judged unachievable terminates here rather than thrashing).
- `UNACHIEVABLE` **summons a replan**: it surfaces an explicit invitation for the product thinker and facilitator to revisit the intent together (re-open to `drafts/`, or a dedicated replan surface — decided at plan). It is never an automatic rollback and never a silent give-up.
- Generalises spc-31's one-off `HOLD` outcome (flag-for-follow-up, never auto-rollback) into a first-class loop-exit state.

### Manual verification gated on machine-acceptance

- The product thinker's manual verification of an intention runs **only after every machine-checkable criterion reads `MET`**. Verification is the last gate, not the first line of defence.
- Manual sign-off is recorded as a **verification receipt** distinct from the machine verdict of record — so "the machine says MET" and "the product thinker confirms the intention is satisfied" are separately auditable.

## What's Out Of Scope

- **Building the review-queue auto-drainer itself.** This intent is the *policy layer* (which mode, when to replan, when to invite manual verification); the drainer that actually runs queued reviews at a safe boundary is its substrate, scoped separately (the opt-in auto-drain spec candidate).
- **Removing the `record-only` default.** Existing intents keep today's behaviour unless the facilitator opts them into a loop.
- **Automatic rollback / demotion of a shipped intent.** `UNACHIEVABLE` invites a replan; it never silently un-ships or rewrites delivered work.
- **Headless oracle reachability.** `loop-to-acceptance` needs a reachable oracle backend to iterate; the headless-reachability fix is itd-47's job, a dependency rather than scope here.

## Scope Conditions

None stated.

## Acceptance Criteria

- *Given* an intent carries `audit_mode: loop-to-acceptance` in its frontmatter (set at plan time; portable with the intent), *when* a fidelity review returns `NOT_MET` for a criterion, *then* the linked work is re-opened and re-reviewed against the same criteria, and the cycle repeats until all criteria read `MET` or the iteration budget is exhausted.
- *Given* an intent in `loop-to-acceptance` whose iteration budget is exhausted (or whose criteria are judged unmeetable as written), *when* the loop terminates, *then* the intent-level Family-2 rollup becomes `UNACHIEVABLE`, the loop **stops** (never auto-continues), the intent is **marked with a written explanation of why it is unachievable**, and a replan invitation is recorded naming both the product thinker and facilitator — with no automatic rollback of delivered reality and no machine-authored replan.
- *Given* an intent whose machine-checkable criteria all read `MET`, *when* the product thinker is invited to verify, *then* a manual-verification step is offered and its sign-off is recorded as a verification receipt distinct from the machine verdict of record.
- *Given* an intent whose machine-checkable criteria all read `MET` **but** the product thinker judges the criteria themselves were wrong (the why is not delivered despite every criterion passing), *when* manual verification is rejected, *then* the intent routes to the **replan** path (revise the intent's criteria), **not** a synthetic `NOT_MET` that would re-loop the implementation against criteria that already pass.
- *Given* an intent whose machine-checkable criteria do **not** all read `MET`, *when* the workflow reaches the manual-verification point, *then* the product thinker is **not** asked to verify — the loop (or the replan invitation) runs first.
- *Given* a fidelity review returns `INCONCLUSIVE` (the fail-closed result of a malformed or unreachable reviewer), *when* the loop processes it, *then* it is recorded as today — `INCONCLUSIVE` does **not** summon the product thinker and does **not** itself trigger replan (it is a "could not run the audit" signal, not a "the intent is impossible" signal).
- *Given* an intent left at the default `audit_mode: record-only`, *when* a fidelity review returns `NOT_MET`, *then* behaviour is unchanged from today — the verdict is recorded to `## Audit Notes` and no re-work is triggered.
- *Given* an intent reaches `UNACHIEVABLE` (loop exit) or its manual verification is rejected (wrong-criteria replan), *when* the product thinker takes it up, *then* they use the `/abcd:intent grill` skill to think the replan through, and the recorded `why-unachievable` / rejection justification seeds that grill session.
- *Given* the on-close lifecycle hook (`intent_lifecycle`), *when* any of these modes is active, *then* the hook remains a pure data function (no subprocess, no oracle dispatch) — the mode logic lives in the drainer/policy layer.

## Resolved (grill 2026-06-02)

The 2026-06-02 grill (5 questions across Dialectic / Definition / Counterfactual / Maieutics / Elenchus) settled five design points; they are reflected in the acceptance criteria above:

- **Mode storage → intent frontmatter** (`audit_mode: record-only | loop-to-acceptance`), set at plan time. Portable with the intent (survives the lifeboat); `record-only` is the explicit default value.
- **`UNACHIEVABLE` → a Family-2 intent-level rollup value** (`CLEAN | BROKEN | UNACHIEVABLE`); per-criterion verdicts unchanged. It is *not* a third enum and *not* reused from Family-1 `MAJOR_RETHINK` (that scores a change, not a promise — the families stay disjoint).
- **Wrong-criteria rejection → replan, not re-loop.** When the machine says `MET` but the product thinker judges the criteria themselves were wrong, the defect is the *criteria*, not the code — route to replan (rewrite criteria), never a synthetic `NOT_MET` (which would loop forever against criteria that already pass). The replan path therefore has **two entry points**: loop-exhausted/unmeetable, and human-rejects-satisfied-criteria.
- **`UNACHIEVABLE` always stops and summons the product thinker**, with a written `why-unachievable` explanation; no machine auto-replan (the product thinker owns the why). The product thinker uses `/abcd:intent grill` to think the replan through, seeded by that explanation.
- **`INCONCLUSIVE` stays fail-closed only — no summons, no replan.** It is the result of a malformed/unreachable reviewer (a backend signal), not a "the evidence is contradictory" signal, so it cannot be trusted as a human-summons trigger.

## Open Questions

- **The loop budget for `loop-to-acceptance`.** What consumes an iteration (a full re-review? a re-open + re-implement + re-review cycle?), and what the default budget is. Mirror `MAX_REVIEW_ITERATIONS` or set an intent-grain equivalent.
- **How the replan invitation surfaces.** Re-open the intent to `drafts/` with a `replan_reason`? A dedicated replan queue/surface the facilitator drains? What state the original delivered reality is left in. (Both replan entry points — `UNACHIEVABLE` and wrong-criteria rejection — share this surface.)
- **How manual-verification sign-off is recorded.** A verification receipt schema, distinct from the machine verdict of record, with an explicit `rejected` state that carries the wrong-criteria justification into the seeded grill.
- **Iteration autonomy bound.** `loop-to-acceptance` iterates unattended up to budget; the precise rule for when a budget-exhausted loop flips to `UNACHIEVABLE` vs. is judged unmeetable earlier.

## Related

- **spc-28** (intent lifecycle hook + review queue) — ships the on-close move + review *enqueue*; this intent is the policy layer that decides what happens to a queued/recorded verdict.
- **spc-12** (intent-fidelity-reviewer) — ships the Role 1 per-criterion verdicts (`MET`/`NOT_MET`/…) this loop consumes; this intent adds the terminal `UNACHIEVABLE` outcome the current enum lacks.
- **itd-47** (Codex leg in `_build_cli_oracle`) — `loop-to-acceptance` needs a reachable oracle to iterate headlessly; dependency, not scope.
- **spc-31** `HOLD` outcome (`desync_report.run_fn31_gates`) — the one-off "flag-for-follow-up, never rollback" precedent this intent generalises into a first-class loop-exit state.
- **itd-44** (fourth intent kind / decision) and **itd-43** (terminology) — the product-thinker / facilitator two-vocabulary split this intent leans on.
- **a dated working-log entry (2026-06-02)** — the `[Design/Spec-candidate]` entry and the upstream `[Design/Decision-needed]` auto-drain recommendation that produced this intent; the auto-drainer is this intent's substrate.

## Prior Art

- `spc-52-audit-loop-to-acceptance-modes` — the predecessor implementation delivered this intent (tasks .1–.3); its AC reconciliation below is carried as design input per the brief's delivery-state provenance note. In this repo itd-50 is undelivered (nothing in the Go tree implements the audit loop) and ships only when this intent reaches `shipped/` with its own audit notes.

### Predecessor AC reconciliation (spc-52)

In the predecessor implementation each acceptance criterion above is satisfied and the open questions are resolved at its plan + build time; the table attributes which spc-52 task owns which behaviour.

| itd-50 Acceptance Criterion | Status | Where delivered |
|---|---|---|
| `loop-to-acceptance` re-opens + re-reviews a `NOT_MET` until `MET` or budget exhausted | Satisfied | spc-52.2 — `audit_loop_policy.decide_loop_action` + the queue-layer re-enqueue in `review_queue._apply_loop_policy` (reopen de-risk gate: no clean native spec-store reopen surface, so the loop re-enqueues at the queue layer) |
| budget-exhausted / unmeetable → `UNACHIEVABLE` rollup, loop stops, written explanation + replan invitation naming both roles, no rollback, no machine-authored replan | Satisfied | spc-52.2 — `UNACHIEVABLE` is a rollup-layer terminal; `write_replan_invitation` writes the `why-unachievable` block; intent stays `shipped/` |
| machine criteria all `MET` → manual-verification offered, recorded as a receipt distinct from the machine verdict | Satisfied | spc-52.3 — `verification_receipt.write_receipt` (`offered` receipt under `.abcd/logbook/audit/verify-<ts>/`, never merged into `## Audit Notes`) |
| `MET` but product thinker rejects (wrong criteria) → replan path, NOT a synthetic `NOT_MET` | Satisfied | spc-52.3 — `verification_receipt.record_rejection_replan` re-enters the SHARED replan surface (one writer, two entry points); test-pinned that no `NOT_MET` is written |
| machine criteria NOT all `MET` → product thinker not asked (loop / replan runs first) | Satisfied | spc-52.3 — `is_acceptance_eligible` gate; the gate API refuses an ineligible rollup (premature-suppression test) |
| `INCONCLUSIVE` → recorded as today, no summons, no replan | Satisfied | spc-52.2 — `decide_loop_action` fail-closed branch; never flips to `UNACHIEVABLE` |
| `record-only` (default) → behaviour unchanged from today | Satisfied | spc-52.1 — absent mode resolves to `record-only`; queue-entry shape regression-pinned |
| `UNACHIEVABLE` / rejection seeds `/abcd:intent grill` | Satisfied | spc-52.2 / .3 — both replan blocks name the grill seed |
| on-close hook stays a pure data function (no subprocess / oracle) | Satisfied | The mode logic rides the spc-43 drainer / policy layer; `intent_lifecycle` is untouched by the loop |

**Open questions (predecessor answers, to re-adjudicate at spec time):** loop budget = one re-open+re-review cycle per iteration, default `3` (spc-52.1 § Decision context); replan surface = no `drafts/` move, a `why-unachievable` + replan block in `## Audit Notes` with the intent kept in `shipped/` (spc-52.2 R4); manual-verification sign-off = the receipt schema `{intent_id, machine_rollup, state, justification?, recorded_by_role, ts}` with the `rejected_wrong_criteria` state carrying the justification to the shared replan surface (spc-52.3 R5).
