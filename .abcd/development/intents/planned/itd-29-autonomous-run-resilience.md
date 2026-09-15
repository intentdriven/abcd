---
id: itd-29
slug: autonomous-run-resilience
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
blocked_by: [itd-27]
builds_on: [itd-28]
severity: major
---

# Domain Experts Run A Spec Autonomously And Recover From Anything Without Touching Git

## Press Release

> **abcd collapses autonomous-run safety to a six-verb operating surface a domain expert can use without ever opening git.** Before `abcd spec start spc-X` kicks off the autonomous run through the pluggable run seam, a pre-flight budget check estimates token cost against remaining quota and refuses to start if the math doesn't add up. Mid-run telemetry surfaces "X% of daily budget consumed, Y tasks remaining" via spc-29-42i (telemetry spec). On a 429 rate-limit response the run catches it cleanly, writes a `RESUME-FROM` checkpoint to the native spec store, and exits. `abcd spec resume spc-X` picks up exactly where it stopped. `abcd spec rewind spc-X --to-task 3` undoes a wrong turn — soft by default (lets the user edit the task spec and re-run; keeps reviews visible for context), hard with `--hard` (full discard). Completed specs auto-merge to a `dev` trunk branch only when all reviews verdict `SHIP` and lint/smoke pass; promotion to `main` requires explicit `abcd spec ship spc-X`. When auto-rebase hits a conflict, the spec suspends and emits a single concrete next-step: `abcd spec resolve spc-X` walks the user through resolution. The domain expert never types `git reset`, never sees a branch name, never decides whether a 429 means "retry" or "give up."
>
> "I'd kicked off an autonomous run, gone to lunch, come back to find it had burned through my Opus budget on iteration 7 of a task that was already wrong," said Iris, product lead. "abcd's pre-flight check would have flagged the budget; the rewind would have undone iteration 6; resume would have picked up Sonnet for the rest. Instead I spent two hours with git and a model bill. Never again."

## Scope Reconciliation

Parts of this intent's scope overlap with adjacent run-seam specs:
- **Graceful 429/quota handling, checkpoint, clean exit, resume-on-reset** — spc-19 (infra-failure classification, no-burn backoff) + spc-35 (quota-window sleep-until-reset, cross-run markers, clean weekly exits). The run-seam side of scope items "Graceful 429 handling" and the checkpoint substrate exists.
- **Budget spreading** — spc-47 (iteration pacing, model-aware) covers part of the budget concern from the proactive side.
- **Checkpoints per spec** — the native spec-store checkpoint records exist (spc-41 consumes them).

The residual scope this intent owns is the OPERATOR SURFACE: the verbs, the pre-flight budget estimate, model auto-downgrade, rewind, and the trunk/auto-merge pattern. A **v1 cut** — `status` / `pause` / `resume` / standalone `preflight` riding the existing sentinels + checkpoints + markers — comes first; `rewind`, `ship`, `resolve`, auto-merge, and auto-downgrade stay in this intent for a later cut (the v2 trigger: first real demand for rewind or trunk promotion). Terminology note: the surface uses run/spec vocabulary, not `epic` (the itd-43 direction). The autonomous engine underneath is the **pluggable run seam** (Workflows / the companion harness / native loop), not a fixed loop ([adr-27](../../decisions/adrs/0027-autonomous-run-pluggable-seam.md)); checkpoints and reviews live in the **native spec store** ([adr-26](../../decisions/adrs/0026-native-spec-layer-ccpm-backend.md)).

## Why This Matters

abcd's near-term value is autonomous execution: a domain expert hands an intent to the engineering loop and walks away. Today, three things can go wrong, each requiring git literacy or model-quota intuition the domain expert shouldn't need:

1. **Rate-limit failure mid-run.** Anthropic API 429 / quota exhaustion mid-iteration leaves the repo in some intermediate state. The domain expert sees "the model stopped working" and has no clean recovery path.
2. **A spec completes wrong but later tasks built on the wrong one.** The domain expert wants to "go back" — currently means knowing about `git reset`, branches, low-level task-uncomplete commands, manual cleanup, possibly cross-branch reverts.
3. **Branch lifecycle stranding.** The autonomous run creates a `spc-X-foo` branch, completes it, and then nothing. Branch stays open, work is invisible to the rest of the repo, lifeboat doesn't see it. Domain expert never merges → branches pile up → confusion about "is this done?"

These are **failure modes of a system that doesn't yet exist.** The substrate (`/abcd:intent` + sub-verbs including `/abcd:intent grill`, lifeboat, spc-1 reviews, spc-2 review artefacts → the native review store) must ship first. The first time a domain expert runs an autonomous loop end-to-end and hits any of these, the texture becomes clear and this intent can be designed against real evidence — not guesses.

This intent **captures the concern now** so the project memory holds it. **Implementation depends on the substrate shipping first.** The triggers to revisit are listed below.

## What's In Scope

- **Six-verb operating surface for autonomous runs:**
  - `abcd spec start <id>` — pre-flight budget check + autonomous loop kickoff
  - `abcd spec status <id>` — progress, budget burn, blocker count, next concrete action
  - `abcd spec pause <id>` — clean suspend (writes checkpoint, exits loop)
  - `abcd spec resume <id>` — pick up from last checkpoint
  - `abcd spec rewind <id> --to-task <N>` — soft (default) or `--hard`
  - `abcd spec ship <id>` — explicit promotion from `dev` trunk to `main` after user confirmation
  - `abcd spec resolve <id>` — guided conflict resolution when auto-rebase fails
- **Pre-flight budget check**: estimate token cost (tasks × ~80k tokens × iteration count) against remaining quota; refuse to start if the math doesn't add up.
- **Mid-run budget telemetry** via spc-29-42i: surface "X% of daily budget consumed, Y tasks remaining" in `abcd spec status`.
- **Graceful 429 handling**: catch rate-limit response, write `RESUME-FROM` checkpoint, exit cleanly. No retry loops that burn remaining budget.
- **Optional model auto-downgrade**: domain-expert opt-in: "if Opus runs out, fall back to Sonnet for remaining tasks." Configurable per-project.
- **Checkpoint-per-task**: each task ending creates a tagged restore point; `rewind --to-task <N>` reverts via the tag, not raw git.
- **Soft vs hard rewind**: soft keeps reviews/work visible in the native review store for context; hard discards. Default soft.
- **Trunk pattern**: `dev` (or `staging`) is the auto-merge target when conditions met; `main` requires `abcd spec ship`. Conditions: all tasks done + all reviews verdict `SHIP` + lint passes + smoke passes + no human-flagged concern.
- **Auto-rebase before declaring done**: spec branch rebases on `dev` after each task; on conflict, spec auto-suspends with single next-step.
- **Audit trail**: every auto-merge writes an entry to the native review store (`<spec>/INDEX.md`: auto-merged-at, by, reviews-cited, smoke-tests-passed).
- **Ride the pluggable run seam**: the autonomous engine is a pluggable seam (Workflows / the companion harness / native loop); abcd verbs are the friendly surface layer above whichever engine is configured.
- **Pick up out-of-band merges (async human review).** A long run may open a *series* of PRs that a maintainer merges asynchronously. The run detects each merge, prunes the merged branch, re-targets/rebases any in-flight follow-on branch onto the updated base, and re-runs the gates before continuing — so the run stays in lockstep with the maintainer's merges without the user touching git. This extends the auto-rebase item above from the engine's own trunk to externally-merged PRs. Conflicts suspend to `abcd spec resolve` (never auto-resolved); never force-push; a red gate after a rebase is a hard stop. **MVP:** the host/loop performs the git via the `gh`+`git` it already has. **Promotion:** a read-only `abcd run reconcile --json` advisory verb computes the reconcile plan (which PRs merged, which branches are stale, what rebase is needed) so any host consumes it and the destructive git never enters abcd's transport-agnostic core.

## What's Out of Scope

- **Building a bespoke autonomous engine** — the run seam is pluggable (Workflows / the companion harness / native loop); abcd provides the operator surface over the configured engine, not a new engine.
- **Distributed multi-machine autonomous runs** — single-machine only.
- **Auto-merge to `main`** — explicit `abcd spec ship` always required for main-branch promotion.
- **Cross-spec conflict auto-resolution** — when two parallel autonomous specs both touch the same file, the second to rebase suspends and the user resolves manually via `abcd spec resolve`.
- **Real-time budget caps** — pre-flight check + mid-run telemetry only; no automatic mid-run cutoff (the model decides; the user sees and intervenes).
- **Budget-aware task reordering** — if budget is tight, the loop runs tasks in declared order, not in some optimised order.
- **Multi-tenant budget pooling** — single user's quota only.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** a user runs `abcd spec start spc-X`, **when** the pre-flight check estimates token cost > remaining quota, **then** the command refuses to start with a clear cost-vs-budget breakdown and a recommendation (downgrade model, defer some tasks, or top up).
- **Given** an autonomous run encounters a 429 response, **when** the run catches it, **then** a `RESUME-FROM` checkpoint is written to the native spec store, the run exits 0, and `abcd spec resume spc-X` picks up at the exact iteration it stopped.
- **Given** a spec has completed 3 of 5 tasks and the user runs `abcd spec rewind spc-X --to-task 3`, **when** the command completes, **then** task 3 is marked incomplete, tasks 4–5 (if any) are reverted, and the user sees `abcd spec status spc-X` reflecting the new state — without any git command being typed by the user.
- **Given** a spec completes with all tasks done + all reviews verdict `SHIP` + lint passes + smoke passes, **when** the auto-merge condition is checked, **then** the spec branch auto-merges to `dev` and an audit entry is written to the native review store (`<spec>/INDEX.md`).
- **Given** a spec auto-rebase hits a conflict, **when** the loop detects it, **then** the spec suspends, emits a single concrete next-step (`abcd spec resolve spc-X`), and `abcd spec status` reflects the suspension reason.
- **Given** the user runs `abcd spec ship spc-X`, **when** the command completes, **then** the user is shown a confirmation prompt summarising what's about to merge to `main` and only proceeds on explicit confirmation.
- **Given** a domain expert who has never used git from the terminal, **when** they walk through start → 429 → resume → wrong-task discovery → rewind → re-run → ship, **then** they complete the flow without typing any git command and without consulting git documentation.
- **Given** a long autonomous run has opened PRs #A and #B and the maintainer merges #A out-of-band, **when** the run next checks in, **then** it prunes #A's merged branch, rebases the in-flight #B branch onto the updated base, re-runs `make preflight`, and continues — no git typed by the user; **and** if the rebase conflicts or the gate fails, the run suspends with a single next-step (`abcd spec resolve`) rather than force-pushing or pushing through.

## SOTA

> _Per the [sota-per-intent principle](../../principles/sota-per-intent.md):
> existing alternatives + rough maturity, then the chosen path. Adding this is
> still a manual step — the `intent_sota` lint gate is the principle's unbuilt
> promotion rung._

- **Autonomous run engine.** Alternatives: agent-run frameworks / harness loops
  (*usable–mature*, moving fast). → **Path 2**: the pluggable run seam
  ([adr-27](../../decisions/adrs/0027-autonomous-run-pluggable-seam.md)) — a native
  thin-loop floor with opt-in Workflows / companion-harness backends. No core
  dependency.
- **Branch / merge reconciliation.** Alternatives: stacked-PR tooling — Graphite
  (*mature*), git-town (*mature*), `ghstack` (*usable*); merge automation —
  Mergify (*mature*), GitHub-native auto-merge (*mature, already on the host*).
  Each owns the git and assumes its own workflow. → **Path 2 / host-owns-git**:
  the host/loop performs prune + rebase + retest with the `gh`+`git` it already
  has; abcd stays read-only via a future `abcd run reconcile --json`. GitHub
  auto-merge is already used for `docs:`/`chore:` PRs — the easy-external-onboard
  for the merge half. Conflict-resolution and force-push are deliberately
  excluded: a human decides (fail-closed).
- **Verdict — Path 2, no new dependency.** The seam to a mature external
  (Graphite / Mergify, or GitHub-native auto-merge) stays open; no path-1 or
  path-3 hard stop.

## Revisit Triggers

This intent moves from `drafts/` to `planned/` when ANY of the following happens after the substrate ships:

1. **First user reports lost work** from an autonomous run that died mid-iteration.
2. **First user reports mystery 429** that they couldn't diagnose without help.
3. **First user reports stranded branch** where the autonomous run completed but they didn't know how to merge.
4. **First user reports rewind difficulty** that took >30 minutes to undo a wrong autonomous turn.
5. **Two consecutive autonomous runs** of substrate specs that succeed end-to-end (proving the run seam works enough to surface its own failure modes at scale).

The first user to hit (1)–(4) is also asked to record the texture in the `.abcd/work/issues/` ledger so this intent designs against real evidence when promoted.

## Open Questions

- **Verb namespace**: `abcd spec <verb>` (recommended for namespace clarity) vs `abcd <verb>` (shorter but collides with future surfaces). Decide at plan-review.
- **Trunk branch name**: `dev` vs `staging` vs `main-staging` vs project-configurable. Recommend project-configurable with `dev` default.
- **Rewind semantics for `--hard`**: discard branch entirely vs keep branch but tag a "rewound-from" marker. Recommend tag-and-keep so the user can `git reflog` if they realise they wanted the work back.
- **Auto-downgrade decision logic**: Opus → Sonnet → Haiku on budget exhaustion vs single-step Opus → Sonnet only. Probably single-step initially; multi-step if proven needed.
- **Telemetry / multi-model substrate**: this intent assumes a telemetry surface (cost/budget visibility) and a multi-model orchestration surface (for auto-downgrade). Earlier drafts referenced legacy abcd-repo IDs (`spc-29-42i`, `spc-9-kbe`) which do NOT exist in `abcd-cli` — those are speculative substrate from the legacy roadmap, not declared in this brief. **Action for plan-review:** identify or create the actual specs (in the native spec store) that deliver the telemetry + multi-model substrate, then list them as hard dependencies here.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._

## References

- Coordinates with: `itd-1` (acceptance gates), `itd-3` (modular rules loader, may govern budget rules), `itd-9` (schema migration, for checkpoint format), `itd-28` (RP reviews → the native review store, audit-trail integration target via `spc-2-move-repoprompt-review-artifacts-into`).
- Builds on: a future native spec for budget/cost surfacing (likely a `spc-N-budget` spec); the pluggable run seam.
- Implemented by: `spc-35-ralph-quota-window-resilience` — the 429/quota implementation. spc-35 delivers the window-aware quota classifier (`QUOTA_WEEKLY`/`QUOTA_FIVE_HOUR`), the `RATE_LIMITED` completion marker, and the window-aware sleep that this intent's "graceful 429 handling" acceptance describes. spc-35 is the concrete substrate behind this intent's rate-limit scope.
- The out-of-band-merge-pickup scope (the added scope bullet + AC + SOTA) was surfaced during the itd-80 intent-lifecycle build; the design rationale (host-owns-git MVP → a read-only `abcd run reconcile --json` advisory verb, conflicts/force-push excluded) is recorded in the itd-80 run-learnings note under `research/notes/` (2026-07-11).
- The **auto-merge guardrails** in this intent's scope (trunk-only, SHIP-verdict-gated not CI-gated, audit entry, conflict→human, `ship` promotes to `main`) are recorded as a standing decision — `.abcd/work/DECISIONS.md` (2026-07-13) — so the trust-boundary rule outlives this intent's own lifecycle; it graduates to a brief invariant + an ADR when the v2 cut is built. The **presentation** of this and every run surface follows the [`facilitator-default-thinker-optional`](../../principles/facilitator-default-thinker-optional.md) principle.
