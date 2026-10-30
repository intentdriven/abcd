---
id: itd-2610150602547271
slug: a-repository-declares-whether-its
spec_id: spc-2610161609350616
kind: standalone
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
> person, a maintainer's ready pull request waits for the facilitator's
> approval: abcd says that approval is needed, and arms the merge only once it
> is given, whoever asks. With abcd, a maintainer's pull request that has
> passed its gates and its review pair goes straight into the merge queue, or
> has auto-merge armed where the forge has no queue, with no one asked, and
> the status line says merging is autonomous. A contributor's pull request
> waits for a maintainer's approval in either mode. A fresh install starts
> with a person, so installing abcd changes nothing about who merges.
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

- A repository abcd manages, with `.abcd/config.json` and a GitHub remote the <!-- cond: cond-2610161609359243 -->
  `gh` client can reach.
- One facilitator mode per repository. A team of product thinkers (itd-99) <!-- cond: cond-2610161609350397 -->
  and per-branch modes are outside it.
- Merges abcd itself starts, through `abcd land`, the build loop, or a shell <!-- cond: cond-2610161609354767 -->
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
  set it. While the mode is `auto`, abcd's row in the host's status line
  says merging is autonomous, through the status line the person consented
  to and nothing else.
- **Who is a maintainer.** One seam answers it for a pull request's author:
  their permission on the repository as the forge reports it. Write,
  Maintain and Admin are maintainers; Read and Triage are contributors. A
  custom role counts as the role it is based on, and an author with no
  permission on the repository (the forge answers that it knows none) is a
  contributor. A permission that cannot be read at all is neither: abcd
  says the author's role is unverified and arms nothing.
- **Who approves.** An approval of a pull request's head commit is either a
  forge review by a maintainer other than its author, or the technical
  facilitator's approval recorded through abcd's own approval act, which a
  person runs and the shell guard refuses to an agent. The second exists
  because the facilitator usually authors the pull requests abcd opens, and
  the forge refuses an author's approval of their own. A contributor's pull
  request is armed only after such an approval, in either mode. A
  maintainer's pull request is armed at once in `auto`; in `human` it is
  armed only after such an approval, and until then abcd says the
  facilitator's approval is needed.
  The product thinker never approves a merge. abcd enforces this itself,
  since the forge lets a pull request merge with no approval, and it adds no
  approval rule or code-owner file to the forge.
- **The merge gate.** `mergeGate`, the one function `abcd land` and the build
  loop ask before arming a merge (introduced by iss-2610150524246898 with AM1
  as its stand-in), reads the mode. `auto`: queue or arm at once by the
  ruleset mirror's method. With no queue gating the default branch, arm
  auto-merge, and where the forge refuses that, say so. `human`: arm only a
  head the technical facilitator has approved, and otherwise say the pull
  request waits for the facilitator's approval. Either mode applies the
  approvals above first.
- **The hard rule.** While the mode is `human`, `abcd guard` refuses
  `gh pr merge --auto`, `gh pr merge` aimed at a queue, and the GraphQL
  `enqueuePullRequest` mutation, whoever runs them. It reads the mode through
  the same resolver.
- **Proof of review.** In `auto`, the merge gate arms only a head that a
  review receipt names: one receipt per commit, written when the review pair's
  verdicts for that commit pass the stop rule, and found from any worktree the
  way preflight receipts are.
- **abcd's own repository** sets `facilitator.mode: auto` in its committed
  config, in the same change, so its sessions keep queueing.

## What's Out of Scope

- The rest of itd-97: the composed "when" of solo mode.
- Changing any repository's forge settings, rulesets or queue.
- Gating the push of a non-default branch. A push is not a merge
  (ruling below).

## Acceptance Criteria

- **Given** a repository with no `facilitator.mode` in any layer and a
  maintainer's open pull request with no approval of its head, **when**
  `abcd land` reaches the passing branch, **then** it arms nothing, and says
  the pull request waits for the technical facilitator's approval and why.
- **Given** the mode is `human` and a maintainer other than the author has
  approved the head commit of a maintainer's pull request, **when** `abcd
  land` reaches the passing branch, **then** it arms the merge by the ruleset
  mirror's method.
- **Given** either mode and a pull request whose author has Read or Triage
  permission, or none, on the repository, **when** `abcd land` or the build
  loop reaches it with no maintainer's approval of its head commit, **then**
  it arms nothing and says a maintainer's approval is needed; **when** a
  maintainer other than the author has approved that head, **then** the
  mode decides as for a maintainer's pull request.
- **Given** an approval of an earlier commit and none of the head, **when**
  the merge gate reads the approvals, **then** it counts none.
- **Given** an author with a custom repository role, **when** the seam reads
  their permission, **then** they count as the role it is based on.
- **Given** an author's permission the forge cannot be asked for, or a
  custom role whose base cannot be read, **when** the merge gate reaches the
  pull request, **then** it arms nothing, whatever approvals there are, and
  says the author's role is unverified.
- **Given** the mode is `human` and a maintainer's pull request authored
  from the technical facilitator's own account, **when** the facilitator
  records their approval of its head through abcd's approval act, **then**
  the next land arms it; **when** an agent runs that act through a host with
  the guard hook, **then** the guard refuses it.
- **Given** `facilitator.mode: auto` in `.abcd/config.json` and a merge queue
  gating the default branch, **when** `abcd land` reaches a passing branch with
  an open pull request, **then** the pull request is in the queue when land
  exits, with no earlier entry awaited.
- **Given** `facilitator.mode: auto` and no review receipt names the branch's
  head commit, **when** `abcd land` reaches its open pull request, **then** it
  arms nothing and says the head has no recorded review pair, naming the
  commit.
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
- **Given** the mode is `auto` in a managed checkout with abcd's status line
  installed, **when** the host refreshes the status line, **then** abcd's row
  says merging is autonomous; **given** `human`, **then** the row has no such
  element. The read asks the network nothing and stays inside the status
  line's time budget.
- **Given** a value other than `human` or `auto` in any layer, **when** the
  mode resolves, **then** the read is refused loudly, naming the file, and
  nothing is armed.

## Open Questions

_None recorded yet._

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
  one code path, with no special case for abcd, because building abcd also
  tests the process its managed repositories follow.
- 2026-10-15, agreed with the release-cutting session: the push of a
  non-default branch after a passing preflight is not gated by the mode.
  The facilitator's approval point is the merge.
- 2026-10-15, the product thinker: until this intent ships, `abcd land` keeps
  AM1 as the merge gate's stand-in.

- 2026-10-16, the technical facilitator on the product thinker's delegation
  (planning): `auto` arms only a head a review receipt names, because the
  product thinker ruled on 2026-10-15 that automated merging is never
  unreviewed merging (the reviews are automated, not skipped), and a hand or
  cloud branch otherwise carries no proof the review pair ran.
- 2026-10-16, the technical facilitator on the product thinker's delegation
  (planning): `human` refuses arming outright, also where the forge would hold
  the merge for an approval, as itd-97's criterion for the human facilitator
  requires; a person merges.
- 2026-10-16, the product thinker: this intent moves to planned.

- 2026-10-29, the product thinker: approving a merge is the technical
  facilitator's act, never the product thinker's, whose part is to try the
  outcome and judge it. In `auto` nothing waits for an approval; in `human`
  abcd tells the facilitator their approval is needed and arms once it is
  given. This supersedes the 2026-10-16 ruling that `human` refuses arming
  outright. No forge approval rule or code-owner file is added for it, and
  the autonomous mode shows in the status line as well as on the board.
- 2026-10-29, the product thinker (refining the same day): a maintainer is
  anyone with Write, Maintain or Admin permission on the repository, and a
  contributor anyone with Read or Triage. A contributor's pull request needs
  a maintainer's approval in either mode; a maintainer's goes through in
  `auto` and waits for the facilitator's approval in `human`. Who is a
  maintainer is read from the repository permission level. The contributor
  rules are carried on by itd-2610290858115394, which builds on this intent.
- 2026-10-29, the technical facilitator on the product thinker's delegation
  (records): AM1 is retired as the merge gate's stand-in, but its reading of
  approvals is kept for a contributor's pull request and for `human`, and it
  counts only an approval of the head commit by a maintainer other than the
  author. `auto` keeps its review-receipt condition. A custom role is judged
  by its base role, and an author the forge reports no role for is a
  contributor.
- 2026-10-29, the technical facilitator on the product thinker's delegation
  (agreed with the maintainer-gate intent's author): a permission that
  cannot be read is not a contributor's, since a token that reads reviews
  but not permissions would then arm on an approval; it arms nothing and is
  called unverified. In `human` the facilitator's approval is also an act
  abcd records for the head commit, run by a person and refused to an agent
  by the guard, because the facilitator authors most pull requests and the
  forge refuses an author's own approval; a second maintainer is not
  required.
- 2026-10-29, the technical facilitator on the product thinker's delegation
  (records): the status line shows the mode only while it is `auto`, since
  `human` is every fresh install's default and an element there would say
  nothing new; it goes through the consented status line alone, as nothing
  else in the host's settings calls abcd.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: one declared facilitator mode, one merge gate every merging path asks, and a guard on the merge commands keep merges where a repository's owners want them; shown wrong if a merge abcd starts happens while the mode is human with no approval of the head by the facilitator, if a contributor's pull request merges with no maintainer's approval of its head, or if auto arms a head no review receipt names
