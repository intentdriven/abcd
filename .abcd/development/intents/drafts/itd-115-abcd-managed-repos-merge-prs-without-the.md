---
id: itd-115
slug: abcd-managed-repos-merge-prs-without-the
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
---

# A Ready PR Merges Without Ever Wedging BEHIND

## Press Release

> **abcd-managed repos stop losing time to out-of-sync pull requests.** Onboarding
> configures a **merge queue** by default: a PR marked ready enters the queue,
> the forge tests it *combined with the current base*, and merges it in order the
> moment it is green — no one runs `update-branch` by hand, no PR sits `BEHIND`
> forever because another merged first, and CI runs against the state that will
> actually land instead of re-running on every manual re-sync. Where a queue is
> not available, abcd falls back to a protocol-level auto-update — poll, and
> re-sync a behind PR automatically — so the toil is still gone. Either way the
> strict merge gate stays intact: it is what catches a duplicate record id minted
> by two parallel sessions, and abcd never trades that safety for convenience.
>
> "Three of my routines were opening PRs in parallel and every one of them kept
> stalling `BEHIND` — I was hand-updating branches and burning CI to land a
> one-line change," said Carol, a facilitator. "Now they queue and land
> themselves, and I never think about it."

## Why This Matters

Under a ruleset that requires up-to-date branches, GitHub's auto-merge **never
updates the branch itself**, so the instant one PR merges, every other open PR is
`BEHIND` and wedged — armed, all checks green, and going nowhere ([iss-172]).
The manual fix (`update-branch`) re-runs the whole matrix and often just loses
the next race. Across parallel agents this is constant, and it wastes both the
facilitator's attention and CI minutes on nothing.

The tempting fix — drop "require up-to-date" — is **the wrong one, and the record
says so.** [iss-172] records the invariant: strict-up-to-date is what forces a
PR's *merged* result to be re-gated before it lands, and that gate is load-bearing
because record-id minting is serialised only *within* a checkout, not across
checkouts — two branches can each mint the same id, each pass the record gate in
isolation, and the duplicate exists only in the merge. Strict is what catches it.
Relaxing strict resolves the stall by removing that gate.

A **merge queue** resolves the stall the *right* way: it re-runs the required
checks against the projected merge and lands PRs in order — so it preserves
exactly the gate strict provides, automatically, without the manual churn. It is
strict's safety minus the toil.

## SOTA

Anchors: **GitHub Merge Queue** (GA 2023; the forge-native answer to this exact
problem), and the **merge-train** lineage it descends from — Bors, GitLab merge
trains, Zuul's speculative gating.

**Declared path: 1 — adopt the SOTA forge feature.** The merge queue is a ruleset
setting plus a CI trigger (`merge_group`), not a code dependency, so adopting it
carries no `go.mod` cost — the new-dependency stop does not apply; the only cost
is the config abcd already owns for managed repos. abcd's role is to *configure*
it during onboarding, not to reimplement queuing. Where the queue is unavailable
(older forge, a plan without it), the fallback is **[iss-172]'s rung 1** — the
protocol-level poll-and-`update-branch`, which preserves strict. The
relax-strict option is deferred: it becomes safe only once [itd-114]
(collision-proof ids) removes the duplicate-id risk by construction; until then
strict is never relaxed. The independent fit-challenge runs at plan time.

## Acceptance Criteria

> _BDD (the [itd-1] discipline)._

- **Given** a managed repo with the merge queue configured, **when** two PRs are
  ready concurrently, **then** they land in order with no manual `update-branch`
  and neither sits `BEHIND`, and the required checks run against the projected
  merge (the gate preserved).
- **Given** onboarding configures the merge queue, **when** it sets up CI,
  **then** the required checks also trigger on the `merge_group` event — without
  which the queue cannot gate — and abcd verifies this rather than assuming it.
- **Given** a repo where the merge queue is unavailable, **when** a PR falls
  `BEHIND`, **then** abcd's rung-1 auto-update brings it current (poll +
  `update-branch`), preserving strict — never by relaxing strict.
- **Given** [itd-114] (collision-proof ids) has not shipped, **when** the policy
  is configured, **then** strict-up-to-date is left intact (the duplicate-id gate
  stays); relaxing it is out of scope until collision-proof ids land.

## Decomposition (itd-84 hand-run, 2026-08-16)

Verdict **FILE-AS-IS with flags**:

| Part | Type | Home |
|------|------|------|
| The managed-repo merge policy (queue default + rung-1 fallback) | capability | this intent |
| Adopt GitHub Merge Queue; the `merge_group` CI trigger; the rung-1 fallback | SOTA declaration (path 1) | this intent's SOTA section |
| "Strict is the duplicate-id gate; never relaxed until ids are collision-proof" | trust rule | **already recorded** as [iss-172]'s invariant — this intent RESPECTS it, does not re-declare |
| Ruleset + CI-trigger wiring on managed repos | plumbing | the itd-92 / itd-106 onboarding surface |

Typed links: `refines` [iss-172] (delivers its rung-2 "scaffolded CI / merge
queue" ask; iss-172 resolves when this ships), `refines` [itd-92]
(branch-protection setup) and itd-106 (CI setup), and **`refines` [itd-114]** —
itd-114 is what later *unlocks* safely relaxing strict.

**No reversal flag.** The earlier "just relax strict" idea *would* have reversed
[iss-172]'s invariant; this design was corrected to respect it, so nothing is
reversed — the correction is the point.

## Open Questions

- **Merge-queue availability across repo tiers.** The queue (like rulesets) may
  need a paid plan on private repos — the same tier gap the 2026-07-29 ruleset
  sweep hit. Where unavailable, rung-1 is the whole answer; state that explicitly.
- **The rung-1 credential question** (carried from [iss-172]): a scaffolded
  auto-update *workflow* needs a PAT/App because the default `GITHUB_TOKEN`'s
  pushes don't trigger CI. The agent-token protocol form has no such need. Decide
  whether the scaffolded rung is in scope here or stays with the itd-107 grill.
- **Queue batching config** (how many PRs per CI run) — a tuning knob, defer.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._

[iss-172]: ../../../work/issues/open/iss-172-abcd-managed-repos-need-pr-queue-hygiene.md
[itd-1]: ../disciplines/itd-1-acceptance-gates.md
[itd-92]: itd-92-abcd-verifies-branch-protection-on.md
[itd-114]: itd-114-abcd-mints-collision-proof-record-ids.md
