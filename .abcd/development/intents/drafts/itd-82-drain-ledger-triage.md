---
id: itd-82
slug: drain-ledger-triage
kind: standalone
suggested_kind: standalone
bundle: null
spec_id: null
reclassification_history: []
builds_on: [itd-4, itd-29, itd-46]
related_adrs: [adr-25, adr-27]
severity: major
---

# One Verb Sorts The Open Ledger Into Work That Ships Itself And Work That Needs A Human To Think

## Press Release

> **abcd turns "drain the backlog" from a nightly babysitting job into a single unattended verb.** `abcd drain` reads the open issue ledger, triages every ready issue, and routes each to the machine that already knows how to handle it: a self-contained defect is fixed autonomously — detector-first, gated by `make preflight`, opened as one PR per issue and never self-merged — while an issue that turns out to need a design decision is **promoted to an intent draft** (the `capture promote` path) seeded from the issue body, so the judgement lands in the intent→spec pipeline where a human owns it, not in a risky autonomous guess. Blocked, ops-only, and already-decided issues are skipped with a logged reason. When the run stops, one handoff shows every issue's disposition — this PR, that intent, or skipped-because. The maintainer's night collapses to two queues in the morning: PRs to review, and drafted intents to design.
>
> "I used to point the loop at the ledger and then hover, because the moment it hit something design-shaped it would either stall or — worse — try to design it," said Alice, a maintainer. "Now `drain` fixes the mechanical stuff while I sleep and *files the hard ones as intents for me to grill*. I wake up to a merge queue and a think queue, not a mess."

## Why This Matters

The C-phase `/abcd:run` dogfood proved the autonomous fix-loop works against `backlog: ledger`: detector-first fixes, dual-review catching real trust-boundary BLOCKs, one PR per issue. But it also proved the **triage** is the load-bearing, human-shaped part. Across the run, a person (the orchestrator) had to decide, per issue: is this a self-contained fix, a design-STOP that must not be attempted autonomously, or an ops/setup task that is not code at all? Get that wrong and the loop either stalls on a design decision or — the real hazard — makes one silently.

That classifier is exactly what a maintainer should not hand-run every night, and exactly what abcd already has the surfaces to automate *around*: the run engine fixes (itd-29 / the run protocol's `backlog: ledger` mode), and `capture promote` (itd-46 / spc-30) elevates an issue to an intent draft with a transactional four-field back-link. What is missing is the **sorting hat in front of them** — the verb that reads the ledger, classifies each issue, and dispatches to fix / promote / skip.

This matters beyond one nightly drain: it is the honest bridge between abcd's two substrates. Raw captures (iss-N, itd-4) accumulate faster than a human can triage; the designed value pipeline is intent→spec→run (itd-29). `abcd drain` is the conveyor that moves mechanical work straight to a PR and design work up into the intent pipeline, so neither queue silts up. It is project-agnostic: every abcd repo captures issues, every one benefits from a machine that drains the fixable and elevates the rest.

## What's In Scope

- **`abcd drain` — the ledger-triage front door.** Reads `abcd capture list --open --json`, takes ready issues (`blocked_by_open` empty), and per issue classifies + routes:
  - **Fixable / self-contained defect** → the autonomous run path (itd-29 / run protocol, `backlog: ledger`): detector-first fix, `make preflight` gate, correctness + (trust-boundary) security review, one PR per issue, ledger `resolve` folded into the branch, **never self-merged**.
  - **Design-STOP / feature-shaped / premise-change** → `capture promote <iss-N>` (itd-46): draft an intent seeded from the issue body with the four-field back-link (`source_issue`/`promoted_to` + `related_issues` ↔ `related_intents`). The issue stays open; **no autonomous design or adoption decision is made** — the draft waits for a human to grill and sign off.
  - **Ops / blocked / already-decided** → skip, with a logged reason (never a silent drop).
- **The classifier.** A deterministic pre-filter on ledger fields (drop blocked; honour an explicit design-STOP marker; route category=`future-work-seed`/feature-shaped to promote) plus a host-delegated LLM judgement (adr-25) for the ambiguous "is this genuinely self-contained?" call. **Fail-safe default: when confidence is low, treat as design-STOP (promote/defer), never as auto-fixable** — a wrong "fix it" is far costlier than a wrong "a human should look."
- **Auditable disposition handoff.** On stop/pause, a machine-readable record of every issue's route (fixed → PR, promoted → itd-N, skipped → reason) plus the classifier's rationale, so the triage is reviewable and re-runnable.
- **Cut A (this intent's shipping slice): fix-path first.** `abcd drain` ships with the classifier + the fix route + the skip route working against the existing run engine. Because `capture promote` (itd-46) is not yet built, the promote route in cut A **flags** each design-STOP issue as "needs promotion → design" with its reason and does **not** attempt an unbuilt verb — degrading loudly, never silently. The full promote route lands in a second cut once itd-46 ships.

## What's Out of Scope

- Building `capture promote` itself — that is **itd-46 / spc-30**; this intent *consumes* it (and, in cut A, stubs its slot).
- The autonomous fix engine and its resilience surface — **itd-29** and the run protocol; `drain` orchestrates them, it does not reimplement them.
- Merging PRs and designing promoted intents — both are **human gates by design**, not automation targets.
- Draining a `milestones`/spec backlog — `drain` is ledger-specific; spec execution is `abcd spec start` (itd-29).
- Any change to the ledger schema or the capture verbs beyond consuming `list`/`resolve`/`promote`.

## Acceptance Criteria

> _BDD format, per the [itd-1 discipline](../disciplines/itd-1-acceptance-gates.md)._

- **Given** an open ledger holding a self-contained defect, a `future-work-seed`/feature-shaped issue, and a blocked issue, **when** `abcd drain` runs, **then** each is routed distinctly — the defect to the fix path, the feature-shaped one to the promote path, the blocked one to skip — and every ready issue receives exactly one logged disposition (no silent drop).
- **Given** a fixable issue on the fix path, **when** drain works it, **then** a detector is armed and watched to fail, the fix lands behind it, `make preflight` is green, one PR is opened (not merged) with the ledger `resolve` folded into the branch, and the PR is one-issue-scoped.
- **Given** a design-STOP / feature-shaped issue and a built `capture promote` (post-cut-A), **when** drain routes it, **then** an intent draft is created seeded from the issue body with the four-field back-link, the issue remains open, and **no** adoption/design verdict is recorded autonomously.
- **Given** cut A (itd-46 not yet built), **when** drain hits a design-STOP issue, **then** it emits a "needs promotion" flag with the classifier's reason and does **not** invoke an unbuilt verb — the degradation is explicit in the handoff, never a silent skip.
- **Given** an issue whose self-contained-vs-design classification is low-confidence, **when** drain classifies it, **then** it routes to the design side (promote/defer), never to an autonomous fix.
- **Given** a completed or paused drain, **when** it writes its handoff, **then** every issue's route is recorded (fixed → PR number, promoted → itd-N, skipped → reason) and is machine-readable and re-runnable.
- **Given** a crafted `iss-N` argument containing path-traversal or unexpected characters, **when** drain resolves it, **then** it is validated against `^iss-[0-9]+$` and rejected otherwise — no file outside the ledger directories is read, written, or moved.

## Prior Art

- **[itd-29](../superseded/itd-29-autonomous-run-resilience.md)** (autonomous-run resilience) — the run/execution surface `drain` dispatches the fix path onto; `drain` is the *planning/triage* half itd-29's run half assumes exists.
- **[itd-46](../shipped/itd-46-abcd-intent-quoted-text-create-symmetric.md)** / **spc-30** (symmetric intent/capture create) — owns `capture promote <iss-N>`, the issue→intent elevator `drain` reuses. Cut A stubs it.
- **[itd-4](../planned/itd-4-issue-capture.md)** (issue capture) — the ledger substrate (`list`/`resolve`) `drain` reads and mutates.
- **`.abcd/development/plans/2026-07-12-abcd-run-protocol.md`** + the ledger-drain run plan — the C-phase evidence that the fix loop works and that triage is the missing, human-shaped piece.
- **adr-25** (host-delegated LLM default) — the classifier's "is this self-contained?" judgement rides the host, no bundled judge.
- **adr-27** (autonomous-run receipt gating) — the fix path's review-gating discipline.

## SOTA

> _Per the [sota-per-intent principle](../../principles/sota-per-intent.md): existing alternatives + rough maturity, then the chosen path._

- **Issue triage / auto-routing.** Alternatives: label-based deterministic triage bots (GitHub Actions triage, *mature* but crude — they route on metadata, they cannot judge "is this a self-contained fix"); autonomous SWE agents that pick issues off a tracker and open PRs (*emerging/2025–26*, e.g. hosted "agent picks an issue" products — capable but opinionated, heavyweight, and they own the whole loop rather than plugging into an existing engine). → **Path 2 (basic-with-seam):** a deterministic pre-filter on ledger fields + a host-delegated LLM classification for the ambiguous call, dispatching to abcd's *own* existing engines (run, promote). No new dependency; the classifier is the only new logic.
- **Classifier confidence / fail-safe.** The judge-calibration direction (per-criterion, explicit low-confidence, report-not-act) argues for an ordinal confidence and a safe default — adopted as *design*: low confidence routes to the human side, never to an autonomous fix.
- **Verdict — Path 2.** No new dependency ⇒ no hard stop; the seams (itd-29 run, itd-46 promote, adr-25 host judge) are load-bearing and already exist ⇒ no bespoke-no-seam stop. **A `sota-researcher` confirmation on the autonomous-triage landscape is a pre-build gate** (the 2025–26 agent-picks-an-issue products move fast; confirm none is a better adopt-target than bespoke triage before building).

## Open Questions

- **Name of the deterministic design-STOP marker.** Does an issue carry an explicit `disposition`/`run_eligible` field (set at capture or by a first triage pass, durable and reusable), or does `drain` re-derive the classification every run? Leaning durable (record the judgement once, per reality-is-filable), which also lets `capture` set it at file time.
- **Cut-A promote-flag surface.** Where does the "needs promotion" flag live in cut A — a handoff line only, or a written marker on the issue — so the eventual `capture promote` (itd-46) can pick it up without re-triage?
- **Batch sizing / budget.** Does one `drain` invocation take all ready fixable issues, or cap per invocation and rely on the run's per-burst budget for pacing?
- **Relationship to `abcd spec start`.** Is `drain` its own verb, or a `--backlog ledger` mode of the run surface? (This intent assumes a distinct verb for the ledger-specific triage; revisit if the surfaces converge.)

## Audit Notes

_None yet — populated by `intent-fidelity-reviewer` when this intent ships._
