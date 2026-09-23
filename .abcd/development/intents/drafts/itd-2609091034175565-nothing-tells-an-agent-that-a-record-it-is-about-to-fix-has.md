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
related_intents: [itd-2609091416295622, itd-2609091416304128, itd-2609091014076309]
related_adrs: [adr-2609091248200336]
---

# A record says who is working on it before anyone else starts, on this machine and on the shared branch

Typed links: `promoted_from` [iss-2609020716570699](../../../work/issues/open/iss-2609020716570699-nothing-tells-an-agent-that-a-record-it-is-about-to-fix-has.md) (the pushed-claim mechanism, whose remedy this record carries and whose scope the maintainer's ruling of 2026-09-09 first widened and then split); `related_intents` [itd-2609091416295622](../shipped/itd-2609091416295622-a-session-sees-the-records-its-sibling-worktrees-hold-before.md) (the read-only sibling-worktree listing, split out of this record), [itd-2609091416304128](itd-2609091416304128-capture-resolve-refuses-a-record-the-default-branch-has-alre.md) (the upstream-terminal refusal, split out of this record), [itd-2609091014076309](itd-2609091014076309-session-and-agent-worktrees-live-in-a-machine-scoped-store-t.md) (the machine-scoped worktree store this record's lane would sit beside and read); `related_adrs` [adr-2609091248200336](../../decisions/adrs/2609091248200336-a-tool-never-creates-directories-in-user-owned-project-space.md) (agent and session scratch is machine-scoped — the rule the lane's home rests on). Prose cross-reference, not a typed link, because no schema field carries the relation ([iss-2609091256264547](../../../work/issues/open/iss-2609091256264547-three-of-the-four-mandated-typed-relations-cannot-be-written.md)): [iss-2608220750029993](../../../work/issues/open/iss-2608220750029993-session-presence-detection-for-shared-checkouts-each-live-se.md), the session-presence lease this record would give a home and a surface.

**Status: not ready.** This draft carries three open questions that gate its scope, stated as questions under Open Questions and answered nowhere in this record. The press release below states the shape the draft proposes so a reader can see what is being asked; any of the three answers may change it.

## Press Release

> **Before an agent fixes a record, the record tells it who is already on it.** `abcd claim <record-id>` marks an issue or intent as taken by the session that runs it: a lease in a machine-scoped lane for the repository, which every session on the machine can read the moment it is written, and — where the readers on every peer can tolerate it — a `claimed_by` stamp on the record itself, which travels with the branch. `abcd <record-id>` answers with the claim first: which session, which worktree, which branch, since when, and whether the claim is trusted or stale. A claim whose session cannot be shown to be alive blocks nothing and says why. On the shared branch, a claim pushed alone loses its race as a rejected merge rather than as a duplicated night's work — if the price of that pass is one the repository will pay.
>
> "I had two sessions running in sibling worktrees, and one of them was about to fix an issue the other had fixed an hour earlier, unpushed," said Maya, an autonomous-development practitioner who runs several agent sessions against one record. "Nothing on either side could see the other. The listing tells me what a peer's tree holds; what I still want is for the record to say who holds it, and for that to mean something after the peer's session has been killed, slept, or cleared."

## Why This Matters

Two collisions happened on one night in the autonomous run of 2026-09-01: a peer session re-fixed two issues a paused branch had already fixed, and two of its open pull requests duplicated merged work. The resolution gate refused the push, which is the gate doing its job, and it is also the first moment anything said no — after the fix was written, tested and reviewed ([iss-2609020716570699](../../../work/issues/open/iss-2609020716570699-nothing-tells-an-agent-that-a-record-it-is-about-to-fix-has.md)). The recorded remedy was three remote-mediated parts: a claim stamped into the record and pushed alone, a duplicate-guard check on the pull request, and `capture resolve` refusing a record already terminal on the fetched default branch.

The maintainer's own collisions, on 2026-09-09, were purely local. Two sessions held worktrees off one checkout; nothing had been pushed; one session nearly minted a record a peer had already captured in its own worktree, and one nearly created a worktree over a peer's. The push queue arbitrates between forks of published history and has nothing to say about two agents in sibling worktrees an hour before anyone pushes, so the recorded design would have caught neither. The first ruling widened this intent to cover both cases.

### The split

Two adversarial reviews then read the widened draft, and the maintainer ruled it into three records. The findings that drove the ruling are recorded here because they are constraints on this record's design, not on the other two:

- **A stamp on an issue record is invisible to an older reader.** `internal/core/issueschema/issueschema.go` holds the issue schema's closed allow-list, and the ledger reader refuses and skips a record carrying a key outside it; `internal/core/lint/schema.go` mirrors the same set into the `record_schema` rule, so the committed-ledger gate refuses what the reader refuses. Adding `claimed_by` to the set updates the binary that is rebuilt — not the plugin-root binary a hosted session runs, not the released binary on a second account, not a peer on an older plugin cache. On this machine the plugin cache holds five vintages from v0.6.6 to v0.7.1 beside a source checkout that runs `go run`; version skew is the steady state, not the exception. During skew, the peer being warned cannot see the stamped record at all, and its own record-lint gate fails on a tree that holds one. (The intent store parses leniently and carries no closed set, so a stamp on an intent has no such reader problem; the collisions on record were issues.)
- **The refusals fire after the work.** `AGENTS.md` requires `capture resolve` in the same change as the fix, so a refusal there lands after the fix is written, tested and reviewed — the same working minute as the push gate the source record complains about. `capture promote` and `intent plan` are before the work, but they are not the verbs the collisions ran through.
- **No staleness threshold is safe in both directions** for the hazard the presence issue names, two sessions in one worktree: the worktree-exists test and the branch-merged test are identical for both sessions, and a host may fire its session-end event when the human clears the context and stays at the keyboard, so the draft's most authoritative death signal can be written by a session that is not dead.
- **The pushed half costs a merge-queue pass per claim.** Measured on this repository's runs, the merge-group `ci` leg takes fifteen to sixteen minutes; a claim-only pull request confined to `.abcd/work/` stands down the optional lanes but not the merge group. That is the serial price before work may begin, per claim.
- **Every local collision on record is covered without a claim.** The 2026-09-09 collisions are a sibling worktree's ledger, readable from disk; the 2026-09-01 duplicated pull requests are a record terminal on the default branch, readable from the last-fetched ref. The listing ([itd-2609091416295622](../shipped/itd-2609091416295622-a-session-sees-the-records-its-sibling-worktrees-hold-before.md)) and the refusal ([itd-2609091416304128](itd-2609091416304128-capture-resolve-refuses-a-record-the-default-branch-has-alre.md)) take those two shapes with no state and no write.

What remains here is the claim: the thing that says who holds a record and whether they are still there. It is filed as not ready because the three questions below are unanswered and each one gates the scope, and it proceeds on evidence — a collision that survives the listing and the refusal — rather than on the design's own momentum.

### The claim surface for co-located sessions

Three surfaces were weighed for the lease. The trade-offs are kept so the interview settles what this draft cannot.

**The shared `.git`.** Worktrees off one checkout share one common dir, and `git worktree list --porcelain` enumerates them; abcd already reads exactly that in the banlist's primary-worktree resolver. Against writing there: `.git/` is git's space, and a tool writing its own files there is the trespass [adr-2609091248200336](../../decisions/adrs/2609091248200336-a-tool-never-creates-directories-in-user-owned-project-space.md) forbids, one owner over; a worktree outlives its session, so the listing says where a peer could be and never whether one is there; and a second clone has a different common dir and is invisible. Verdict: read it, never write it.

**The machine-scoped `~/.abcd/` lane, keyed on the root commit.** The house convention for per-machine state: `history/`, `transcripts/` and `voyage/` are each keyed on the repository's root-commit SHA, and the worktree store intent puts worktrees beside them. Every worktree and clone of one repository on a machine shares the key, so a lease there is visible to precisely the sessions that share a ledger and not a remote. It is space abcd owns by the ADR's terms, it is outside every tree scan the gates walk, and the session-start hook already resolves this store for the repository. Against it: one machine, by construction. This draft proposes it as the lease's home.

**The harness's own session registry.** The one surface that knows a session exists before it writes anything. It is host-specific, and `AGENTS.md`'s boundary keeps abcd host-agnostic; [itd-22](itd-22-harness-portability.md) records that every port re-derives the host's payload schema by hand. Disqualified as a surface, not as a source: the host already hands abcd an opaque `session_id` through the session-start hook, which is the host-profile seam itd-22 names. Where no hook fires, a lease can be written by the first abcd write verb the session runs, and the record says so.

### A hard constraint on any stamp design

Whatever the interview decides about the pushed half, a `claimed_by` key on an issue record cannot ship in one release. The migration is two releases, and the order is fixed by the reader: **release N teaches every reader to tolerate the key** — `issueschema.Known` gains it, `record_schema` stops reporting it, and nothing writes it — and **release N+1 writes it**, only after every peer that shares a ledger has taken release N. Until then the local lease is the whole of the claim, and no session writes the stamp. A design that stamps in the same release it introduces the key makes the claimed record invisible to the peer it exists to warn, and fails that peer's gates besides; the review's finding is recorded here as the boundary, so that no later draft of this record has to rediscover it.

## What's In Scope

- **`abcd claim <record-id>`** on an open issue or a draft or planned intent writes a lease entry naming the record; `abcd claim release <record-id>` withdraws one the caller holds. A record another session holds under a trusted claim refuses and names the session, its worktree and its branch; a claim that is not trusted is replaced, and the replacement reports what it replaced.
- **A session lease** in the machine lane the interview confirms, carrying an opaque session key, the worktree path (home-redacted on any stream), the branch, `started_at`, `last_seen` and the claimed records; written at session start where a hook fires with a session key, and at the first abcd write verb otherwise; never naming the host.
- **`abcd <record-id>` renders the claim first**, marked trusted or not, with the reason where not, ahead of the record's own next moves.
- **Refusals in the write verbs**, on whichever surface the first open question settles, naming the holding session and writing nothing.
- **The `claimed_by` stamp for the pushed case**, only under the two-release migration above, and only if the third open question is answered yes.
- **`AGENTS.md`'s concurrent-sessions convention names the claim verb** beside the listing, once the verb exists.

## What's Out of Scope

- **The sibling-worktree listing** — [itd-2609091416295622](../shipped/itd-2609091416295622-a-session-sees-the-records-its-sibling-worktrees-hold-before.md).
- **The refusal of a record terminal at the last-fetched default branch** — [itd-2609091416304128](itd-2609091416304128-capture-resolve-refuses-a-record-the-default-branch-has-alre.md).
- **Judging that two records describe one observation** — [itd-87](itd-87-recurrence-escalation-in-capture.md).
- **A lock.** Every refusal names a session and can be overridden by releasing the claim from either side; nothing here holds after a session has ended.
- **Cross-machine presence without a push.** Two machines learn of each other's claims when a claim commit lands, and no sooner.
- **Reading any host's session registry, transcript store or process table.** The host reaches abcd through the hooks it fires; what it does not say, abcd does not know.
- **Moving or reclaiming worktrees**, which [itd-2609091014076309](itd-2609091014076309-session-and-agent-worktrees-live-in-a-machine-scoped-store-t.md) carries.

## Mechanism

We expect a claim to be read, where the convention was not, because it is rendered on the surface an agent already consults before touching a record — `abcd <record-id>` is the question "what is my next move" — and because the write verbs refuse against it, so a session that never asked still cannot complete the work past a trusted peer's claim without saying so. We expect the local half to be worth building only if a collision survives the listing and the refusal, because every collision on record is one of those two shapes; a claim built ahead of that evidence is a lock file with better manners. It is falsified if a session that ran `abcd <record-id>` on a claimed record, or was refused by a write verb, still duplicated the work; and it is falsified from the other side if a claim that blocked names a session that was not there — which is the case the second open question exists to bound, and the reason a claim that cannot be trusted blocks nothing.

## Scope Conditions

- The local half holds for sessions on one machine, under one operating-system user, whose repositories share a root commit: worktrees off one checkout and separate clones alike. Two users on one machine do not see each other's leases; across machines only the pushed half speaks, and only after the claim commit has landed.
- The stamp holds only among peers whose reader knows the key. Until every peer that shares a ledger runs a release whose issue schema tolerates `claimed_by`, no session writes it, and the local lease is the whole of the claim.
- The lease is written at session start only where the host fires a session-start hook carrying a session key; a host that fires none, or a bare terminal, writes it at the first abcd write verb. Reading a host's own registry is never part of the mechanism.
- Whether a claim is trusted is decided by the liveness rule the second open question settles; until it is settled, nothing here refuses, because a refusal on an untrusted claim is the lock file this record exists not to add.
- The pushed half costs a pull request and a merge-queue pass per claim, fifteen to sixteen minutes on this repository's measured runs, serial, before the work may begin; it holds only for a repository that accepts that price, and the third open question is whether this one does.
- Folder membership remains the one canonical status signal. A claim is a line in a lease and, under the migration, a stamp on an open record; never a folder, never a lock file in the tree, never a write into `.git/`; and it leaves the record in the same move that makes the record terminal.
- The lease and the stamp name the host nowhere: an opaque session key and the git identity are the whole of the who, because the record family is a site input and a host name on a record would reach the published site.

## Acceptance Criteria

The criteria below hold whatever the three open questions decide; criteria that depend on an answer — the refusal surface, the liveness rule, the pushed half — are absent on purpose and are written at the interview.

- **Given** a peer running a released binary whose issue schema predates the stamp key, **when** it reads a ledger holding a claimed record, **then** the record is visible on every capture surface and its record-lint gate passes — which is to say, no release writes the stamp before the release that tolerates it has reached every peer.
- **Given** a repository with no lane and no leases, **when** every write verb runs, **then** its behaviour is byte-identical to today's: an absent lane is an empty one, never a refusal.
- **Given** two sessions in sibling worktrees off one checkout, nothing pushed, and the first has run `abcd claim iss-N`, **when** the second runs `abcd iss-N`, **then** the render leads with the claim — session key, worktree, branch, since when, trusted or not — before the record's own next moves.
- **Given** a claim held by this session, **when** the record moves to a terminal folder, **then** the lease no longer lists it in the same write, and where a stamp exists it leaves the record in the same move; no terminal record carries a claim.
- **Given** a claim that the liveness rule cannot show to be alive, **when** any verb reads or writes the record, **then** the claim is reported with the reason it is not trusted and refuses nothing, and a new `abcd claim` replaces it and reports what it replaced.
- **Given** the lease file, any stamp, any render and any JSON, **when** they are read, **then** none names a host, none carries an unredacted home path, and the session key is opaque; the privacy-hygiene rule and the harness-leak rule pass over a tree holding claimed records.

## Prior Art

- [iss-2609020716570699](../../../work/issues/open/iss-2609020716570699-nothing-tells-an-agent-that-a-record-it-is-about-to-fix-has.md) — the source record: the three remote-mediated parts, of which the stamp and the duplicate guard stay here under the open questions, and the resolve-time refusal is now its own record.
- [iss-2608220750029993](../../../work/issues/open/iss-2608220750029993-session-presence-detection-for-shared-checkouts-each-live-se.md) — the lease, and its open question of where it lives: the issue rules out the per-worktree local tier because the hazard is two sessions in one checkout, and this draft answers with the machine lane keyed on the root commit.
- [adr-2609091248200336](../../decisions/adrs/2609091248200336-a-tool-never-creates-directories-in-user-owned-project-space.md) and [itd-2609091014076309](itd-2609091014076309-session-and-agent-worktrees-live-in-a-machine-scoped-store-t.md) — agent and session scratch is machine-scoped, keyed on the root commit, listable and reclaimable; the lane copies the shape.
- [adr-2609091248201071](../../decisions/adrs/2609091248201071-the-transcript-corpus-is-a-sibling-store-that-creates-itself.md) and `internal/core/history/location.go` — the root-commit key and the self-creating store; the session-start hook already resolves it, which is the seam the lease rides.
- `internal/core/issueschema/issueschema.go` and `internal/core/lint/schema.go` — the closed issue schema and its lint mirror; the reason a stamp is a two-release migration.
- `internal/core/banlist/worktree.go` — abcd already reads `git worktree list --porcelain` and confirms a worktree from its own `--git-common-dir`.
- [itd-22](itd-22-harness-portability.md) — the host-profile seam the hook payload is; the reason the harness registry is a source and never a surface.
- [itd-87](itd-87-recurrence-escalation-in-capture.md) — the near-duplicate judgement this intent does not make.

## Open Questions

The first three gate scope; planning does not proceed past them.

- **What surface the refusal lives on, given `capture resolve` is the last step.** `AGENTS.md` puts the resolve in the same change as the fix, so a refusal there is after the fix. The surfaces before the fix are `abcd <record-id>` (a render, which cannot refuse), `capture promote` and `intent plan` (which are not the verbs the collisions ran through), and the first commit on the branch (a hook, which the source record's own reasoning about lock files argues against). A refusal that lands where the work is already done saves a push and not a night; whether that is worth a claim is the question.
- **Whether any liveness signal survives SIGKILL, sleep and a context clear.** A killed session writes no session-end; a sleeping machine writes no heartbeat and is not dead; a host may fire its session-end event on a context clear with the human still at the keyboard. For two sessions in one worktree — the hazard the presence issue names — the worktree-exists and branch-merged tests are identical for both, so they distinguish nothing. Every threshold errs in one direction: short, and a live session's claim is reported stale and replaced; long, and a dead session's claim blocks live work. If no signal survives all three, a claim can only ever report and never refuse, and the record is a smaller thing than its title.
- **Whether the pushed half survives its own price.** A claim commit pushed alone costs a pull request and a merge-queue pass — fifteen to sixteen minutes measured on this repository — serially, before work begins, per claim; and the stamp it carries needs the two-release migration above before any peer can read it. The duplicated pull requests on record are caught earlier and free by the upstream-terminal refusal. What the pushed claim buys on top is arbitration between two machines before either pushes a fix; whether a case on record needs that is what decides.
- **Where the lease lives, finally.** This draft proposes the machine lane keyed on the root commit; the interview confirms or overrules it, and decides whether the lane is its own store or a subtree of the worktree store.
- **The verb's home.** `abcd claim` as a top-level verb, or `capture claim` keeping the ledger's writes under one owner.
- **Whether every claim stamps the record, or only `--push`** — moot until the third question and the migration are settled.
- **The claim's account member.** The git identity is the obvious value; whether it is required, and what a session with no identity configured writes, is undecided.
- **Impact.** Additive on its face; left unset until the interview judges it.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
