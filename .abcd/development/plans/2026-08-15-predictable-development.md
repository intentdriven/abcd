# Predictable development — QA machinery plan and run queue (2026-08-15)

**Status:** the second of the three forward plans settled at the 2026-08-15
maintainer grill; runs after
[`2026-08-15-plugin-user-safety.md`](2026-08-15-plugin-user-safety.md).
Consumed by the generic protocol at
[`2026-07-12-abcd-run-protocol.md`](2026-07-12-abcd-run-protocol.md). Absorbs
the consistency remainder of
[`2026-07-29-v0.5.0-security-and-consistency.md`](2026-07-29-v0.5.0-security-and-consistency.md).

**Admission test.** An item qualifies only if it speeds abcd's own build loop
or makes abcd-managed-repo behaviour predictable **without human action** —
the machinery side of the grill's touch-vs-machinery boundary. Anything the
facilitator directly invokes or reads belongs in
[`2026-08-15-facilitator-experience.md`](2026-08-15-facilitator-experience.md);
anything an installer can be hurt by belongs in the plugin-user-safety plan.
The ledger stays the backlog of record.

**Framing (maintainer grill, 2026-08-15).** abcd is designed while being
built with it, so work that accelerates the loop compounds: every automated
check makes the other two plans cheaper and safer to execute, and every
consistency guarantee is one fewer surprise a product thinker or facilitator
has to be warned about. The bar for "predictable" is that what abcd *will do*
in a managed repo is knowable from the record, and drift between record and
behaviour is detected by machinery, not by a person noticing.

## Run contract

Identical to the plugin-user-safety plan's (gate, budget, trailer, reviewers,
strike limit, one PR per item, ledger move in the fixing PR; auto-merge never
inherited, authorised per cycle). Security review applies where a trust
boundary is touched — in this plan that is any item changing CI workflows or
subprocess/environment handling.

## Workstream A — verification milestone

- **Promote [itd-109](../intents/drafts/itd-109-a-two-part-verification-suite-for-abcd.md)
  via `/abcd:intent plan`** (human-paired: the planning interview runs the
  adversarial fit-challenge and the itd-84 decomposition). The intent is
  grill-complete — criteria as the single source, inline assertions,
  sha-keyed receipts, `abcd verify`, environment classes, gating,
  calibration-first rollout. This is the plan's headline: it is the machinery
  that lets acceptance criteria check themselves.
- **[itd-84](../intents/disciplines/itd-84-intent-decomposition.md) is
  adopted (maintainer, 2026-08-15) at the MVP rung**: the
  decompose-before-filing principle and the hand-run protocol in the
  `/abcd:intent` surface page are live, so the queued intent promotions
  (itd-109 here, itd-111 in the plugin-user-safety plan) run the hand-run
  decomposition step and grade it into the calibration note
  (`../research/notes/2026-08-15-decomposition-calibration.md`). The
  capture-time agent rung stays deferred until ~50 graded captures exist.

## Workstream B — the loop stops fighting itself

1. **[iss-219](../../work/issues/resolved/iss-219-dev-install-tests-read-host-path.md)**
   (minor, outsized friction) — the ahoy dev-install tests read the host
   PATH, so four tests fail on any machine with abcd installed and every push
   from this machine needs a PATH-stripping workaround. Isolating PATH in
   those tests retires a standing tax on every single push.
   Autonomous-eligible.
2. **[iss-209](../../work/issues/open/iss-209-every-dependabot-pr-that-bumps-a-pinned.md)**
   (process) — every dependabot action-bump PR fails `TestSelfScaffoldParity`
   and can never go green alone. Scope is exactly the issue body's recorded
   stopping point (the manual `make scaffold-sync` half); the `workflow_run`
   automation is a reviewed **dead end** — re-attempting it is a STOP.
3. **[iss-213](../../work/issues/open/iss-213-several-agents-sharing-one-git-worktree.md)**
   (process) — several agents sharing one worktree invalidated a verification
   result and nearly lost committed work. The deliverable is a protocol/guard
   change (worktree isolation as the default for parallel agents), recorded
   where the run protocol lives. Human-paired: it changes how runs are driven.
4. **Tail (later, in this order):
   [iss-48](../../work/issues/open/iss-48-e2e-behavioural-scenarios.md)**
   (behavioural end-to-end coverage) and
   **[itd-75](../intents/drafts/itd-75-cli-eval-harness.md)** (the CLI eval
   harness from drop-in fixtures) — both feed the same goal as itd-109 and
   should be sequenced against its plan once it exists, not before.

## Workstream C — the record cannot drift silently

5. **[iss-192](../../work/issues/resolved/iss-192-brief-surface-drift-v042-release-gate.md)**
   (major) — the v0.4.2 crosscheck found 26 brief-surface discrepancies; the
   brief lags the shipped surface. Fix the drift *and* note which sentences
   only the periodic crosscheck guards, per CONTEXT.md's trust-the-binary
   sharp edge.
6. **[iss-180](../../work/issues/wontfix/iss-180-itd-85-shipped-but-undelivered.md)**
   (major, process) —
   [itd-85](../intents/drafts/itd-85-audit-verb.md) shipped whole but still
   sits in `drafts/`; the lifecycle move is the fix, and whatever let a
   shipped intent sit unmoved is the finding to record.
7. **[iss-218](../../work/issues/open/iss-218-record-tier-tool-naming-convention.md)**
   (minor) — the record-tier tool-naming convention is prose, not a detector;
   give it one (lint-family rule), fixture-first.
8. **[iss-96](../../work/issues/open/iss-96-now-that-transcripts-are-captured.md)**
   (minor, verification milestone) — carried from the v0.5.0 plan unchanged:
   re-check the transcript-scanner coverage gaps against the landed pattern
   set; close or re-scope, never close on assumption.
9. **[itd-106](../intents/drafts/itd-106-abcd-sets-up-the-ci-a-repo-requires-and.md)**
   — abcd sets up the CI a repo requires and reports what it did: the
   cross-repo consistency intent. A later promotion (its own grill first);
   listed so the plan names where "managed repos behave alike" is headed.
10. **The itd-84 deterministic pre-pass** (admitted 2026-08-15, with the
    adoption) — the plain-Go lexical candidate-finder plus atomicity smell,
    as a subverb on the existing `intent` family. Design decides the front
    door (a `decompose` subverb versus a check inside `intent ready`) and
    whether it runs in preflight — itd-84's own open question. Human-paired
    design, then a normal TDD arc. Collision:
    [iss-210](../../work/issues/resolved/iss-210-lone-token-subverb-guess-writes-a-record.md)
    lives in exactly this verb-family parsing — fix it first or in the same
    arc, never around it.

## Ordering and collisions

- B1 (iss-219) first: it cheapens every subsequent push in every workstream.
- A's promotion interview precedes B4's tail — iss-48/itd-75 must be
  sequenced against itd-109's plan, not raced with it.
- C5 (iss-192) touches the brief; any release cut in the meantime re-runs the
  crosscheck, so land C5 outside a release window or coordinate with the
  release-gate README's two-commit receipt flow (never edit that flow — STOP
  condition 3).
- B2 (iss-209) and the plugin-user-safety plan's Cut B both touch
  `release.yml`'s scaffold-parity constraint; whoever takes either reads the
  other's collision note first (recorded in the install-experience plan).

## STOP conditions (this plan)

1. **itd-84's agent rung before calibration.** The discipline is adopted at
   the MVP rung only: building or running the automated capture-time
   validator before the hand-run protocol is calibrated (~50 graded
   captures, per itd-81) is a STOP — the hand-run protocol is the gate
   until then.
2. **iss-209's dead end stays dead.** Any `workflow_run`-style automation
   re-attempt is a STOP; the four preconditions recorded in the issue body
   are the only door.
3. **No release-gate or receipt-contract edits** ride any item here.
4. **Missing or ambiguous record** — fail closed, never synthesise.
5. **Security-review BLOCK** stops the change (standing rule).

## Explicitly out

Installer-reachable defects (plugin-user-safety plan); everything the
facilitator touches (facilitator-experience plan); the shelved scanner
adjacency mechanism and its structural successor iss-229 (owned by the
plugin-user-safety plan's structural tier).
