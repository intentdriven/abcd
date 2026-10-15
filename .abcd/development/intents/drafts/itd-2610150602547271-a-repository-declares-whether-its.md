---
id: itd-2610150602547271
slug: a-repository-declares-whether-its
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-97]
severity: major
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# The facilitator's mode gates the merge

## Press Release

> **A repository says who merges, and abcd keeps to it.** A repository
> declares whether its technical facilitator is a person or abcd. With a
> person, a ready pull request waits for them: abcd never queues it or arms
> auto-merge on it, whoever asks. With abcd, a pull request that has passed
> its gates goes straight into the merge queue, or has auto-merge armed where
> the forge has no queue, with no one asked. A fresh install starts with a
> person, so installing abcd changes nothing about who merges.
>
> "I run my repository alone, so I told abcd it is the facilitator," said
> Alice, a solo founder. "Now every branch that passes its checks and its
> review pair is in the queue a minute later, four at a time." Carol, a tech
> lead, keeps the default: "Nothing merges on my team until one of us has read
> it, and abcd can't be talked past that."

## Why This Matters

iss-2610150524246898 asks abcd to queue every ready pull request at once,
default on. The product thinker refined it on 2026-10-15: when the
facilitator is abcd, every ready pull request auto-merges or is queued; when
the facilitator is a person, that person reviews and approves, and the merge
waits for them. Today nothing records which one a repository has. The build
loop and `abcd land` stand in with ruling AM1: they arm the merge where a
ruleset requires an approval. That stand-in is looser than it looks. A
code-owner rule counts as an approval requirement for every pull request,
though the forge demands the owner's approval only for owned paths. So in
abcd's own repository a change touching only `internal/` merges after CI with
no person reading it.

itd-97 makes the whole facilitator a mode. This intent is its first slice:
the declaration, the place it is shown, and the one gate that obeys it, the
merge. The rest of itd-97 (scheduling, self-firing reviews, triage) composes
later behind the same declaration.

## Mechanism

We expect this to keep merges where the repository's owners want them,
because there is one place that says who merges, one function that every
merging path asks, and a guard that refuses the merge commands themselves.
So a path that forgets to ask is still stopped.

## Scope Conditions

- A repository abcd manages, with `.abcd/config.json` and a GitHub remote the
  `gh` client can reach.
- One facilitator mode per repository. A team of product thinkers (itd-99)
  and per-branch modes are outside it.
- Merges abcd itself starts, through `abcd land`, the build loop, or a shell
  command the guard judges. A person merging in the forge's web page is
  untouched.

## What's In Scope

- **The declaration.** The key `facilitator.mode` in `.abcd/config.json`,
  with the values `human` and `auto`, read through `internal/core/layered`.
  The bundled default is `human`. `~/.abcd.noindex/config.json` may also set
  it, and `human` from any layer wins: a person can always take the role
  back on their own machine, and no repository file can take it from them.
  (This is the same off-wins rule as `land.push` and `land.enqueue`, and the
  rules loader's kill switch.)
- **Shown.** The bare `abcd` status board names the mode and the layer that
  set it.
- **The merge gate.** `mergeGate`, the one function `abcd land` and the build
  loop ask before arming a merge (introduced by iss-2610150524246898 with AM1
  as its stand-in), reads the mode. `auto`: queue or arm at once by the
  ruleset mirror's method. With no queue gating the default branch, arm
  auto-merge, and where the forge refuses that, say so. `human`: arm nothing,
  and say the pull request waits for the facilitator.
- **The hard rule.** While the mode is `human`, `abcd guard` refuses
  `gh pr merge --auto`, `gh pr merge` aimed at a queue, and the GraphQL
  `enqueuePullRequest` mutation, whoever runs them. It reads the mode through
  the same resolver.
- **abcd's own repository** sets `facilitator.mode: auto` in its committed
  config, in the same change, so its sessions keep queueing.

## What's Out of Scope

- The rest of itd-97: the composed "when" of solo mode.
- Changing any repository's forge settings, rulesets or queue.
- Gating the push of a non-default branch. A push is not a merge
  (ruling below).
- Proving the review pair ran. "Ready" means preflight, the review pair, the
  stop rule and CI. This slice trusts the caller for the review pair (see Open
  Questions).

## Acceptance Criteria

- **Given** a repository with no `facilitator.mode` in any layer, **when**
  `abcd land` reaches a passing branch with an open pull request, **then** it
  arms nothing, and says the pull request waits for the facilitator and why.
- **Given** `facilitator.mode: auto` in `.abcd/config.json` and a merge queue
  gating the default branch, **when** `abcd land` reaches a passing branch with
  an open pull request, **then** the pull request is in the queue when land
  exits, with no earlier entry awaited.
- **Given** `auto` in the repository's config and `human` in
  `~/.abcd.noindex/config.json`, **when** anything resolves the mode, **then**
  it is `human`, and the board names the machine file as the layer that set
  it.
- **Given** the mode is `human`, **when** an agent runs `gh pr merge 123
  --auto` (or the enqueue mutation) through a host with the guard hook,
  **then** the guard refuses it and names the setting that would allow it.
- **Given** the mode is `auto`, **when** the build loop lands a lane, **then**
  it arms by `mergeGate` exactly as `abcd land` does, through one code path.
- **Given** either mode, **when** the bare `abcd` board renders, **then** it
  names the facilitator mode and its layer.
- **Given** a value other than `human` or `auto` in any layer, **when** the
  mode resolves, **then** the read is refused loudly, naming the file, and
  nothing is armed.

## Open Questions

- **How does `auto` know the review pair ran?** Today the review pair leaves
  receipts only for build-loop lanes. A hand or cloud branch carries no
  machine-checkable proof. Options: require a review receipt keyed to the
  head commit, or trust the caller for this slice. A facilitator question for
  planning.
- **Does `human` keep AM1's arming where the forge itself holds the merge for
  an approval?** itd-97's criterion says refuse arming outright. This slice
  follows it. Confirm in planning.

## Decisions

- 2026-10-15, the product thinker: a fresh install's facilitator is a
  person (`human` by default). A repository turns `auto` on deliberately, and
  abcd's own repository sets it in its config. This supersedes the "AI by
  default" clause of adr-2609151528057260 under the security standard rule
  (abcd never changes how a repository works). The superseding record is
  filed with this intent.
- 2026-10-15, the product thinker (relayed by the release-cutting session):
  in `auto`, a ready pull request merges with no human step. So a revertable
  merge is facilitator work, not an action reserved for the product thinker.
  Auto-merge never means unreviewed: the reviews are automated, not skipped.
- 2026-10-15, the product thinker: abcd and the repositories it manages take
  one code path, with no special case for abcd ("if it works for abcd, it
  should work for abcd-managed repos").
- 2026-10-15, agreed with the release-cutting session: the push of a
  non-default branch after a passing preflight is not gated by the mode.
  The facilitator's approval point is the merge.
- 2026-10-15, the product thinker: until this intent ships, `abcd land` keeps
  AM1 as the merge gate's stand-in.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
