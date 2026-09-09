---
id: itd-2609091034175565
slug: nothing-tells-an-agent-that-a-record-it-is-about-to-fix-has
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
promoted_from: iss-2609020716570699
origin: extracted-from-record
production_mode: hand-written
---


# A record says who is working on it before anyone else starts, on this machine and on the shared branch

Typed links: `refines` [iss-2608220750029993](../../../work/issues/open/iss-2608220750029993-session-presence-detection-for-shared-checkouts-each-live-se.md) (the session-presence lease this intent gives a home and a surface); `promoted_from` [iss-2609020716570699](../../../work/issues/open/iss-2609020716570699-nothing-tells-an-agent-that-a-record-it-is-about-to-fix-has.md) (the pushed-claim mechanism, whose scope the maintainer's ruling widens below); composes with [itd-2609091014076309](itd-2609091014076309-session-and-agent-worktrees-live-in-a-machine-scoped-store-t.md) (the machine-scoped worktree store this intent's lane sits beside and reads).

## Press Release

> **Before an agent fixes a record, the record tells it who is already on it.** `abcd claim <record-id>` marks an issue or intent as taken by the session that runs it, in two places at once: a lease in the machine-scoped lane for the repository, which every session on the machine can read the moment it is written, and a `claimed_by` stamp on the record itself, which travels with the branch and reaches everyone else when it is pushed. `abcd <record-id>` — the question "what is this and what is my next move" — answers with the claim first: which session, which worktree, which branch, since when, and whether that session is still alive. `capture resolve`, `capture promote` and `intent plan` refuse a record a live peer holds and name the peer; a claim whose session has ended, or whose branch is merged or gone, is reported as stale and blocks nothing. On the shared branch, a pull request whose `Resolves` trailer names a record already terminal on the default branch fails a required check before anyone reads the diff, and a claim pushed alone loses its race as a rejected merge rather than as a duplicated night's work.
>
> "I had two sessions running in sibling worktrees, and one of them was about to fix an issue the other had fixed an hour earlier, unpushed," said Maya, an autonomous-development practitioner who runs several agent sessions against one record. "Nothing on either side could see the other. Now the first thing a session learns about a record is who holds it, and the answer is there before anything is pushed, because the lease lives on the machine and not on the remote."

## Why This Matters

Two collisions happened on one night in the autonomous run of 2026-09-01: a peer session re-fixed two issues a paused branch had already fixed, and two of its open pull requests duplicated merged work. The resolution gate refused the push, which is the gate doing its job, and it is also the first moment anything said no — after the fix was written, tested and reviewed ([iss-2609020716570699](../../../work/issues/open/iss-2609020716570699-nothing-tells-an-agent-that-a-record-it-is-about-to-fix-has.md)). The recorded remedy is three remote-mediated parts: a claim stamped into the record and pushed alone, a duplicate-guard check on the pull request, and `capture resolve` refusing a record already terminal on the fetched default branch.

The maintainer's own collisions, on 2026-09-09, were purely local. Two sessions held worktrees off one checkout; nothing had been pushed; one session nearly minted a record a peer had already captured in its own worktree, and one nearly created a worktree over a peer's. The push queue arbitrates between forks of published history and has nothing to say about two agents in sibling worktrees an hour before anyone pushes, so the recorded design would have caught neither. **The ruling is that this intent covers both the local case and the pushed case**, and the widening is the design problem: what is the claim surface for sessions that share a machine and not yet a remote?

`AGENTS.md`'s concurrent-sessions convention already asks a session to scan for peers before mutating git state, and both collisions happened in sessions that had that convention in context. A convention that asks an agent to look somewhere abcd never renders is the vigilance-only rung [iss-2608220750029993](../../../work/issues/open/iss-2608220750029993-session-presence-detection-for-shared-checkouts-each-live-se.md) names; this intent is the armed rung it asks for, and the claim is what the lease says the session is doing.

### The claim surface for co-located sessions

Three surfaces were weighed. The trade-offs are stated so the interview settles what this draft cannot.

**The shared `.git`.** Worktrees off one checkout share one object store and one `.git/worktrees/` directory, and `git worktree list --porcelain` enumerates them; abcd already reads exactly that, and confirms the relationship from the candidate's own `--git-common-dir`, in the banlist's primary-worktree resolver. It is the surface that matches the hazard precisely: the sessions that can collide locally are the ones that share a common dir. Against it: `.git/` is git's space, and a tool writing its own files there is the trespass [adr-2609091014087993](../../decisions/adrs/2609091014087993-a-tool-never-creates-directories-in-user-owned-project-space.md) forbids, one owner over; a worktree outlives its session, so the listing says where a peer could be and never whether one is there (the maintainer's checkout lists twenty-seven, most spent); and a second clone of the same repository on the machine has a different common dir and is invisible. Verdict: read it, never write it. The worktree list is how a lease is joined to a checkout, not where the lease lives.

**The machine-scoped `~/.abcd/` lane, keyed on the root commit.** The house convention for per-machine state: `history/`, `transcripts/` and `voyage/` are each keyed on the repository's root-commit SHA, and [itd-2609091014076309](itd-2609091014076309-session-and-agent-worktrees-live-in-a-machine-scoped-store-t.md) puts the worktree store beside them. Every worktree and every clone of one repository on a machine shares the key, so a lease there is visible to precisely the sessions that share a ledger and not a remote — clones included, which the shared `.git` cannot reach. It is space abcd owns by the ADR's own terms (agent and session scratch is machine-scoped), it is outside every tree scan the gates walk, and the session-start hook already resolves this store for the repository on every session it sees. Against it: it is one machine, by construction, and says nothing across machines — which is the pushed half's job, and the reason both halves are one intent. This draft proposes it as the lease's home.

**The harness's own session registry.** The one surface that knows a session exists before it writes anything, and the way the maintainer found the peer session on the night. It is host-specific, and `AGENTS.md`'s boundary keeps abcd host-agnostic; [itd-22](itd-22-harness-portability.md) records that every port re-derives the host's payload schema by hand. Reading the registry from the core would bind the core to a host, so it is disqualified as a surface. It is not disqualified as a source: the host already tells abcd that a session exists, through the session-start hook, and hands it an opaque `session_id` in the payload the hook entrypoints read. That payload is the host-profile seam itd-22 names, so a lease written at session start rides the seam abcd already stands on and needs no adapter. Where no hook fires — a bare terminal, a host without hooks — the lease is written by the first abcd write verb the session runs, and the record says so rather than pretending the session was known from its start.

So the design has one lease per session in the machine lane, joined to a worktree and a branch by git's own answer, refreshed by the hooks that already fire on every prompt, and listing the records the session has claimed; and one `claimed_by` stamp per record for the pushed case. `abcd claim` writes both. Folder membership stays the status signal: a claim is a stamp on an open record, never a folder, and the record is terminal the day the same move that resolves it removes the claim.

## What's In Scope

- **`abcd claim <record-id>`** on an open issue or a draft or planned intent: appends the record to the running session's lease and stamps `claimed_by` — the git identity as `account`, the branch, an opaque session key, and `claimed_at` — into the record's frontmatter. The issue reader drops a record carrying a key it does not know, so the key joins the schema in the same change. A record another live session holds refuses and names the session, its worktree and its branch; a stale claim is replaced and the replacement reports what it replaced. `abcd claim release <record-id>` withdraws a claim the caller holds.
- **A session lease in `~/.abcd/<lane>/<root-sha>/`**, written at session start where a hook fires and at the first abcd write verb otherwise; carrying the opaque session key, the worktree path (home-redacted on any stream), the branch, `started_at`, `last_seen` and the claimed records; refreshed by the prompt hook that already runs on every prompt; ended by the session-end hook, which writes locally and never touches the network. The lease never names the host.
- **`abcd <record-id>` renders the claim first**, from both sources: the record's own stamp, and the lane's leases, each marked live or stale and with the reason for stale. `abcd peers` (working name) renders every lease in the lane for this repository — session, worktree, branch, claimed records, last seen — and, for each live peer, the record ids present in that peer's ledger and absent from this tree, so a record a peer minted is visible before it is pushed. The `/abcd` status board carries one line: live peers and records they hold.
- **The write verbs consult the lease.** `capture resolve`, `capture wontfix`, `capture promote` and `intent plan` refuse a record a live peer holds, naming the peer; a stale claim blocks nothing and is reported alongside the write.
- **The pushed half, as recorded and corrected.** `claimed_by` reaches the remote only as a commit; the convention is that the claim commit travels alone, ahead of the fix, so a second claim on the same record is a conflict on one frontmatter line and the loser learns at the queue rather than at review. A required check fails a pull request whose `Resolves` or `Resolved-by` trailer names a record already terminal on the default branch, or whose claim stamp names a session other than the one that resolves it without a release in between. `capture resolve` refuses a record already terminal on the local `origin/main` ref as last fetched, states how old that ref is, and performs no fetch of its own.
- **Staleness is a rule, not a timer alone.** A lease is stale when its session-end has been written, when its `last_seen` is older than a threshold the spec fixes, or when its worktree no longer exists; a pushed claim is stale when its branch is merged into or gone from the default branch. A stale claim is reported, never enforced, and is replaced by the next claimant.
- **`AGENTS.md`'s concurrent-sessions convention names the verb** as the scan-before-mutating step, and the plugin surface carries the claim, release and peers verbs.

## What's Out of Scope

- **Judging that two records describe one observation.** The near-duplicate capture is caught here only where the peer's record already exists and is listed; a similarity judgement at capture time is [itd-87](itd-87-recurrence-escalation-in-capture.md)'s recurrence work.
- **A lock.** Every refusal names a live session and can be overridden by releasing the claim from either side; nothing here holds after a session has ended.
- **Cross-machine presence without a push.** Two machines learn of each other's claims when the claim commit lands, and no sooner.
- **Reading any host's session registry, transcript store or process table.** The host reaches abcd through the hooks it fires; what it does not say, abcd does not know, and the lease records the rung it was learned at.
- **Moving or reclaiming worktrees**, which [itd-2609091014076309](itd-2609091014076309-session-and-agent-worktrees-live-in-a-machine-scoped-store-t.md) carries; this intent reads the worktree list and writes nothing into it.

## Mechanism

We expect a claim to be read, where the convention was not, because it is rendered on the surface an agent already consults before touching a record — `abcd <record-id>` is the question "what is my next move" — and because the write verbs that complete the work refuse against it, so a session that never asked still cannot resolve, promote or plan past a live peer without saying so. The convention failed not because sessions declined to coordinate but because it pointed at a surface abcd never renders and no verb consults; both collisions happened with the convention in context. We expect the local half to land before the pushed half can, because a lease written at session start into a machine lane is visible in the same second to every session that shares the ledger, while a pushed claim is visible only after a queue whose cheapest passage is minutes. It is falsified if a session that ran `abcd <record-id>` on a claimed record, or was refused by a write verb, still duplicated the work — a render that was ignored, or a refusal overridden by releasing a live peer's claim — and the lease's `last_seen` and the refusal's own output make that case distinguishable from a claim that was never written.

## Scope Conditions

- Co-located sessions: the local half covers every session on one machine whose repository shares a root commit — worktrees off one checkout and separate clones alike — and no session on another machine. Across machines only the pushed half speaks, and only after the claim commit has landed on the default branch.
- One machine, one user: the lane is under the caller's own home, so two operating-system users on one machine do not see each other's leases; that is the same boundary the history and worktree stores draw, and the pushed half is what crosses it.
- The lease is written at session start only where the host fires a session-start hook carrying a session key; a host that fires none, or a bare terminal, writes it at the first abcd write verb, and the lease records which rung it was learned at. Reading a host's own registry is never part of the mechanism.
- A stale claim from a session that died is reported and replaced, never enforced: a session-end that was written, a `last_seen` older than the spec's threshold, a worktree that no longer exists, or a pushed claim whose branch is merged or gone, each makes the claim stale. The threshold errs toward reporting a live session as stale rather than a dead one as live, because a stale claim that blocks is the lock file this intent exists not to add.
- The pushed claim's arbitration is the queue's conflict on one frontmatter line, and it costs a pull request per claim at whatever checks the repository requires. A claim-only pull request confined to `.abcd/work/` runs the always-on lanes only under the existing classifier, and the price is stated rather than hidden.
- A claim is a stamp on an open record and a line in a lease; it is never a folder, a lock file in the tree, or a write into `.git/`, and it is removed by the same move that makes the record terminal. Folder membership remains the one canonical status signal.
- The lease and the stamp name the host nowhere: an opaque session key and the git identity are the whole of the who. The record family is a site input, and a host name on a record would reach the published site.

## Acceptance Criteria

- **Given** two sessions in sibling worktrees off one checkout, nothing pushed, and the first has run `abcd claim iss-N`, **when** the second runs `abcd iss-N`, **then** the render leads with the claim — session key, worktree, branch, since when, live — before the record's own next moves.
- **Given** the same two sessions, **when** the second runs `abcd claim iss-N`, `capture resolve iss-N`, `capture promote iss-N` or `intent plan itd-N` on the record the first holds, **then** the verb refuses, names the holding session, its worktree and branch, and writes nothing.
- **Given** two separate clones of one repository on one machine, **when** one claims a record, **then** the other sees the claim on `abcd <record-id>` and `abcd peers`, because the lane is keyed on the root commit and not on the checkout.
- **Given** a peer session that has captured a record in its own worktree and not pushed it, **when** this session runs `abcd peers`, **then** the peer's record id appears under that peer as present in its ledger and absent from this tree.
- **Given** a session whose host fires a session-start hook with a session key, **when** the session starts, **then** a lease exists in the lane before the session has written anything else; **given** a host that fires no hook, **when** the session first runs an abcd write verb, **then** the lease is written then, marked as learned at first write.
- **Given** a lease whose session-end has been written, or whose `last_seen` is older than the threshold, or whose worktree no longer exists, **when** any session runs `abcd <record-id>` or a write verb on a record it claims, **then** the claim is reported as stale with the reason and refuses nothing, and a new `abcd claim` replaces it and reports what it replaced.
- **Given** a pushed `claimed_by` whose branch is merged into or deleted from the default branch, **when** the record is rendered or written, **then** the claim is stale, reported, and blocks nothing.
- **Given** a live claim held by this session, **when** `capture resolve` or `capture wontfix` moves the record to a terminal folder, **then** the stamp leaves the record and the lease no longer lists it in the same write; no terminal record carries a claim.
- **Given** two branches that each stamp `claimed_by` on the same open record, **when** the second reaches the merge queue after the first has merged, **then** the queue rejects it as a conflict on the record, and the rejection names the record.
- **Given** a pull request whose `Resolves` trailer names a record already terminal on the default branch, **when** the required check runs, **then** it fails before review and names the record and the commit that made it terminal.
- **Given** a record already terminal on the local `origin/main` ref, **when** `capture resolve` runs against the open copy in this tree, **then** it refuses, names the terminal path and the age of the ref it judged by, and performs no fetch.
- **Given** the lease file, the record stamp, any render and any JSON, **when** they are read, **then** none names a host, none carries an absolute home path unredacted, and the session key is opaque; the privacy-hygiene rule and the harness-leak rule pass over a tree holding claimed records.
- **Given** a repository with no lane and no leases, **when** every write verb runs, **then** its behaviour is byte-identical to today's: an absent lane is an empty one, never a refusal.
- **Given** `AGENTS.md`'s concurrent-sessions convention, **when** it is read, **then** the scan-before-mutating step names `abcd peers` and `abcd claim`, and the harness's session listing is no longer the only place it points.

## Prior Art

- [iss-2609020716570699](../../../work/issues/open/iss-2609020716570699-nothing-tells-an-agent-that-a-record-it-is-about-to-fix-has.md) — the source record: the three remote-mediated parts, kept here as the pushed half, with the claim stamp's `harness` member withdrawn and the `capture resolve` refusal judged against the local ref without a fetch.
- [iss-2608220750029993](../../../work/issues/open/iss-2608220750029993-session-presence-detection-for-shared-checkouts-each-live-se.md) — the lease, and its open question of where it lives: the issue rules out the per-worktree local tier because the hazard is two sessions in one checkout, and this draft answers with the machine lane keyed on the root commit.
- [adr-2609091014087993](../../decisions/adrs/2609091014087993-a-tool-never-creates-directories-in-user-owned-project-space.md) and [itd-2609091014076309](itd-2609091014076309-session-and-agent-worktrees-live-in-a-machine-scoped-store-t.md) — agent and session scratch is machine-scoped, keyed on the root commit, listable and reclaimable; the lane copies the shape and sits beside the worktree store.
- [adr-2609090717039680](../../decisions/adrs/2609090717039680-the-transcript-corpus-is-a-sibling-store-that-creates-itself.md) and `internal/core/history/location.go` — the root-commit key and the self-creating store; the session-start hook already resolves the history store for the repository, which is the seam the lease rides.
- `internal/core/banlist/worktree.go` — abcd already reads `git worktree list --porcelain` and confirms a worktree from its own `--git-common-dir`; the lease's join to a checkout reuses it.
- `scripts/check-issue-resolution.sh` — RS001 already tells the "record already terminal at the base" shape apart at push time; the duplicate-guard check and the resolve-time refusal move that answer earlier.
- [itd-22](itd-22-harness-portability.md) — the host-profile seam the hook payload is; the reason the harness registry is a source and never a surface.
- [itd-87](itd-87-recurrence-escalation-in-capture.md) — the near-duplicate judgement this intent does not make.

## Open Questions

- **Where the lease lives, finally.** This draft proposes a machine lane keyed on the root commit, and states why the shared `.git` is read and not written; the interview confirms or overrules that, and decides whether the lane is its own store or a subtree of the worktree store itd-2609091014076309 proposes.
- **The verb's home.** `abcd claim` as a top-level verb reads best at the prompt and spans issues and intents; a `capture claim` sub-verb keeps the ledger's writes under one owner and leaves intents to `intent`. The plugin page follows whichever is chosen.
- **Whether every claim stamps the record, or only `--push`.** A local-only claim costs nothing and covers the maintainer's collisions; stamping the record makes a commit the session must carry. One verb writing both is simplest; the interview decides whether the stamp is default or opt-in.
- **The staleness threshold**, and whether the prompt hook is the heartbeat or a session's own writes suffice. A hook fires per prompt on a hosted session and never on a bare terminal, so the two rungs may need two thresholds.
- **What the required check reads.** Whether it fails only on a terminal record, or also on a `claimed_by` naming a different session than the resolver's, and whether that second rule is a check or a warning.
- **The near-duplicate near-miss.** Listing a peer's unpushed record ids is cheap and would have shown the maintainer's duplicate; whether that listing belongs here or waits for itd-87 is a scope decision.
- **The claim's account member.** The git identity is the obvious value; whether it is required, and what a session with no identity configured writes, is undecided.
- **Impact.** Additive on its face; left unset until the interview judges it.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
