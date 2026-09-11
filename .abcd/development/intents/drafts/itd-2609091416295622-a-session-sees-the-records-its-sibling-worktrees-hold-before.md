---
id: itd-2609091416295622
slug: a-session-sees-the-records-its-sibling-worktrees-hold-before
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: major
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-2609091034175565, itd-2609091416304128, itd-2609091014076309]
---

# A session sees the records its sibling worktrees hold before it mints or fixes one

Typed links: `related_intents` [itd-2609091034175565](itd-2609091034175565-nothing-tells-an-agent-that-a-record-it-is-about-to-fix-has.md) (the claim, the lease and the refusals — the record this one was split from on the maintainer's ruling of 2026-09-09, and the record the promotion trail from iss-2609020716570699 runs through), [itd-2609091416304128](itd-2609091416304128-capture-resolve-refuses-a-record-the-default-branch-has-alre.md) (the upstream-terminal refusal, split out the same day), [itd-2609091014076309](itd-2609091014076309-session-and-agent-worktrees-live-in-a-machine-scoped-store-t.md) (the worktree store, whose list verb renders the same worktrees this listing reads). Prose cross-references, not typed links, because no schema field carries the relation ([iss-2609091256264547](../../../work/issues/open/iss-2609091256264547-three-of-the-four-mandated-typed-relations-cannot-be-written.md)): [iss-2609020716570699](../../../work/issues/open/iss-2609020716570699-nothing-tells-an-agent-that-a-record-it-is-about-to-fix-has.md), the source of the split, whose `promoted_to` names the claim record and not this one; and [iss-2608220750029993](../../../work/issues/open/iss-2608220750029993-session-presence-detection-for-shared-checkouts-each-live-se.md), the presence lease, of which this listing is the read-only cousin and not the delivery.

## Press Release

> **Before a session captures or fixes a record, it can see what its sibling worktrees already hold.** Bare `abcd peers` reads the worktrees git lists for this checkout — the primary and every linked worktree that share its common dir — and, for each sibling, reads the issue ledger straight off that worktree's disk and reports the difference from this tree: records open there and absent here, which means a peer has captured something this session has not seen, and records open here and terminal there, which means a peer has already resolved what this session is about to fix. Each row names the sibling's path, its branch and the record ids, with each record's title where the sibling's file can be read. `abcd <record-id>` on an open record says, before the next moves, whether a sibling holds it terminal. The `/abcd` status board carries one line whenever the difference is non-empty. Nothing is written: no claim, no lease, no session key, no hook, no staleness threshold. What a sibling worktree holds on disk is already there to be read, and this verb reads it.
>
> "I had two sessions running in sibling worktrees, and one of them was about to capture an issue the other had captured an hour earlier, unpushed," said Maya, an autonomous-development practitioner who runs several agent sessions against one record. "The file was sitting on the disk the whole time, two directories over. Now the first thing a session does before it mints or fixes anything is look, and the look is one read-only command."

## Why This Matters

Every collision on record so far happened between sessions that could have read each other's files. On 2026-09-01 a peer session re-fixed two issues that a paused branch had already fixed and not pushed ([iss-2609020716570699](../../../work/issues/open/iss-2609020716570699-nothing-tells-an-agent-that-a-record-it-is-about-to-fix-has.md)). On 2026-09-09 two sessions held worktrees off one checkout, nothing pushed: one nearly minted a record its peer had already captured, and one nearly created a worktree over a peer's. In each case the record, or the worktree, existed on disk in a sibling of the tree the session was working in, and nothing rendered it. `AGENTS.md`'s concurrent-sessions convention asks a session to scan for peers before mutating git state, and both 2026-09-09 collisions happened with the convention in context — the convention points at the harness's session listing, which abcd never renders and no verb consults, and which says nothing about records anyway.

The claim record this was split from proposed a machine-scoped lease, a `claimed_by` stamp on the record, refusals in the write verbs and a pushed duplicate guard. Two adversarial reviews found the expensive parts unsound as drafted — the stamp makes a claimed issue record invisible to any peer on an older binary, the refusals fire after the fix is written, no staleness rule is safe in both directions for two sessions in one worktree, and the pushed half costs a merge-queue pass per claim — and found that the collisions on record needed none of it. The maintainer ruled a split into three: this listing, the upstream-terminal refusal ([itd-2609091416304128](itd-2609091416304128-capture-resolve-refuses-a-record-the-default-branch-has-alre.md)), and the claim itself ([itd-2609091034175565](itd-2609091034175565-nothing-tells-an-agent-that-a-record-it-is-about-to-fix-has.md)), which stays a draft carrying its open questions. This is the record that can ship soonest, because it adds no state and no write: `git worktree list --porcelain` is already parsed in `internal/core/banlist/worktree.go`, with the candidate confirming the relationship from its own `--git-common-dir`, and the ledger's three status folders are already read by the capture surfaces. The listing is a diff of two reads abcd already makes.

The boundary is stated plainly. A worktree says where a peer could be, never whether one is there — the maintainer's checkout lists twenty-seven, most of them spent — so this listing is about records, not presence. A record on a branch no worktree has checked out, a stash or an unsaved buffer is invisible to it, and a second clone of the repository has its own common dir and is not a sibling. A record already terminal on the default branch is the upstream case and belongs to the refusal intent. What is left is exactly the shape every local collision on record took, and the claim record proceeds only if a collision survives this listing and that refusal.

## What's In Scope

- **Bare `abcd peers`** (working name; the interview settles it) renders one block per sibling worktree of this checkout: the path (home-redacted on any stream), the branch, and two rows of record ids. `open there, absent here` lists every id under the sibling's `.abcd/work/issues/open/` with no record of that id in any status folder of this tree. `open here, terminal there` lists every id under this tree's `open/` that sits under the sibling's `resolved/` or `wontfix/`. Each id carries the record's title where the sibling's file is readable, and the id alone where it is not. `--json` emits the same blocks.
- **Worktree enumeration is git's answer, confirmed.** The sibling set is `git worktree list --porcelain` run from this checkout, and a candidate is read only when its own `--show-toplevel` is itself and its own `--git-common-dir` is this checkout's, the confirmation the banlist resolver already makes; a listed directory that fails the confirmation is a row that says so and is not read. A listed path that no longer exists is a row that says the worktree is missing. Neither is an empty diff.
- **Read-only, by construction.** The verb writes nothing in this tree, in any sibling, in `.git/` or under the home directory; both trees' `git status` are byte-identical before and after.
- **`abcd <record-id>` on an open issue** says, ahead of `next_moves`, when a sibling worktree holds that id terminal, naming the sibling's branch — the surface an agent already consults for "what is my next move".
- **One line on the `/abcd` status board**: the count of sibling worktrees and the count of ids across the two rows, present only when the diff is non-empty.
- **`AGENTS.md`'s concurrent-sessions convention names the verb** as the scan-before-mutating step, beside the harness's session listing it points at today, and the plugin surface carries the verb.

## What's Out of Scope

- **Any write.** No claim, no lease, no stamp, no hook, no session key. Who is working on a record, and whether they are still there, is the claim record's question.
- **Presence or liveness.** A sibling worktree with a record in it says the record exists, not that a session does.
- **A second clone of the repository.** A clone has its own common dir; `git worktree list` from here does not name it, and nothing here goes looking.
- **A record that is not on a sibling's disk.** A branch checked out nowhere, a stash, an unsaved editor buffer.
- **A record already terminal on the default branch**, which the refusal intent judges against the last-fetched `origin/main`.
- **Judging that two differently-worded records describe one observation.** The listing shows a peer's record by id and title; the similarity judgement is [itd-87](itd-87-recurrence-escalation-in-capture.md)'s recurrence work.
- **Creating, moving or reclaiming worktrees**, which the worktree store intent carries; this verb reads the list and names every worktree it lists, in the store or outside it, the same way.

## Mechanism

We expect a read-only listing to stop the local collisions on record because each was a failure of visibility and not of will: the peer's record was on disk in a sibling worktree before the collision, the convention that asked the session to look pointed at a surface abcd never renders, and a listing over the same disk — rendered on the board a session already reads, on the record dispatch it already consults, and named by the convention — puts the record in front of the session before the mint or the fix. It is falsified if a session that ran `abcd peers`, or saw the board line or the dispatch notice, still minted a duplicate of a listed record or re-fixed a record listed as terminal in a sibling; the render's own output and the two records' timestamp ids tell that case apart from a session that never looked. It is also falsified, differently, if the next collision on record takes a shape the listing cannot see — a branch checked out nowhere, a second clone — which is the case for the claim record and not for a wider listing.

## Scope Conditions

- Holds for the worktrees that share this checkout's common dir, which is the set `git worktree list` enumerates from here. A second clone of the repository on the same machine has its own common dir and is outside the boundary; nothing here sees it, and a collision with one is not this intent's to catch.
- Holds only for a record that is on disk in a sibling's working tree, committed or not. A record on a branch no worktree has checked out, in a stash or in an unsaved buffer is invisible, so the 2026-09-01 paused-branch case is inside the boundary only where that branch was checked out in a worktree at the time.
- Holds at the scale of the maintainer's checkout — twenty-seven worktrees, each ledger some hundreds of records: a directory listing per status folder per sibling, and a record body opened only for its title. Behaviour past a few hundred worktrees, or on a ledger on a network filesystem, is unmeasured.
- Holds where the caller can read the sibling's directory. A worktree under another user's home, or on a path that has gone, is a row stating that, and the missing-tree and empty-ledger cases are never rendered alike.
- macOS and Linux, the two CI legs; the porcelain listing's path form on Windows is untested.
- Assumes the sibling's ledger uses the committed layout — `.abcd/work/issues/{open,resolved,wontfix}/` with the id in the filename — so a sibling at a commit before that layout existed contributes no rows and says so.

## Acceptance Criteria

- **Given** two worktrees off one checkout, a record captured under the sibling's `open/` and absent from every status folder of this tree, **when** `abcd peers` runs in this tree, **then** one row under that sibling names the record id and its title, the block names the sibling's branch and its home-redacted path, the exit code is 0, and `git status --porcelain` in both trees is byte-identical to before the run.
- **Given** a record open in this tree and present under the sibling's `resolved/` or `wontfix/`, **when** `abcd peers` runs, **then** the record appears in the `open here, terminal there` row for that sibling, naming which terminal folder holds it.
- **Given** a sibling worktree whose directory has been deleted without `git worktree remove`, **when** the verb runs, **then** the block for it says the worktree is missing, carries no rows, and the verb still exits 0.
- **Given** a directory that `git worktree list` names but whose own `--git-common-dir` is not this checkout's, **when** the verb runs, **then** the directory is not read and its block says the confirmation failed.
- **Given** a checkout with no linked worktrees, **when** the verb runs, **then** it reports no siblings, exits 0, and the `/abcd` board carries no peers line.
- **Given** a non-empty diff, **when** the `/abcd` board renders in this tree, **then** one line states the count of sibling worktrees and the count of ids across the two rows; **given** an empty diff, the line is absent.
- **Given** an open record that a sibling holds under a terminal folder, **when** `abcd <record-id>` runs on it, **then** the render says so, naming the sibling's branch, before `next_moves`.
- **Given** `--json`, **when** the verb runs, **then** the payload carries the same blocks and rows, and no value in it carries an unredacted home path; the privacy-hygiene rule passes over a captured payload.
- **Given** two separate clones of one repository on one machine, **when** the verb runs in either, **then** neither lists the other, and the render states that the boundary is the checkout's common dir.
- **Given** a sibling whose record file is unreadable or malformed, **when** the verb runs, **then** the id still appears, without a title, and the verb exits 0.
- **Given** `AGENTS.md`'s concurrent-sessions convention, **when** it is read, **then** the scan-before-mutating step names `abcd peers` beside the harness's session listing.

## Prior Art

- `internal/core/banlist/worktree.go` — `git worktree list --porcelain` parsed, with the candidate confirming the relationship from its own `--git-common-dir`; the resolver this listing reuses, and the reason a listed directory is never trusted by its path alone.
- [iss-2609020716570699](../../../work/issues/open/iss-2609020716570699-nothing-tells-an-agent-that-a-record-it-is-about-to-fix-has.md) — the 2026-09-01 collisions; its `promoted_to` names the claim record, and this listing is the piece of that record's scope that needed no claim.
- [iss-2608220750029993](../../../work/issues/open/iss-2608220750029993-session-presence-detection-for-shared-checkouts-each-live-se.md) — the presence lease and the "vigilance-only" rung it names; this listing is a read-only step on that ladder, not the lease.
- [itd-2609091014076309](itd-2609091014076309-session-and-agent-worktrees-live-in-a-machine-scoped-store-t.md) — the worktree store's list verb renders one row per worktree from the same git answer; the interview decides whether the ledger diff is a column of that render or its own verb.
- `AGENTS.md` § Concurrent sessions — the convention this verb gives a surface to.
- [itd-87](itd-87-recurrence-escalation-in-capture.md) — the near-duplicate judgement this intent does not make.

## Open Questions

- **The verb's name and home.** `abcd peers` reads well but promises presence it does not deliver; a sub-verb of the worktree store's listing (`abcd worktree`, with the ledger diff as columns) keeps one verb for one git answer. The interview decides.
- **Whether intent drafts join the diff.** A peer minting a draft intent is the same collision one store over; the listing could read `.abcd/development/intents/drafts/` by the same filename rule. Kept out of the first cut so the criteria stay sharp; the interview may widen.
- **Whether the sibling's record file is opened for its title.** Opening a foreign worktree's file is a read of user-controlled bytes; the id comes from the filename and needs no read. The title is the convenience, and the reader must refuse symlinks and cap the read the way every other record reader here does.
- **Impact.** Additive on its face; left unset until the interview judges it.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
