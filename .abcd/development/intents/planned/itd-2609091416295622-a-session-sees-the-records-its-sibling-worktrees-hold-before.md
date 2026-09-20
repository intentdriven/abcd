---
id: itd-2609091416295622
slug: a-session-sees-the-records-its-sibling-worktrees-hold-before
spec_id: spc-2609202056480020
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609091034175565]
severity: major
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-2609091416304128, itd-2609091014076309, itd-148, itd-2609150819440345, itd-2609201916151817]
---

# A session sees the records its peers hold before it mints or fixes one

## Press Release

> **Before a session captures, fixes or files anything, it can see what every peer already holds.** One read-only reader answers the question from two sources today, the sibling worktrees' disks and the local branches, and from the register later. It reaches a session three ways: `capture resolve`, the record dispatcher and `intent audit` name the peer that holds a record when they cannot find it here, instead of answering not found; the status board carries one line whenever a peer holds something this tree does not; and one read-only command prints the whole picture on demand. It shows issues and intent drafts alike, skips peers whose branch is merged or whose worktree is gone, and says out loud when a peer's ledger is in a state it will not read. Nothing is written: no claim, no lease, no session key. The implement verb reads the same picture before it picks a record.
>
> "I had two sessions running in sibling worktrees, and one of them was about to capture an issue the other had captured an hour earlier, unpushed," said Maya, an autonomous-development practitioner who runs several agent sessions against one record. "The file was sitting on the disk the whole time, two directories over. Now the resolve tells me who has it, before I have done anything."

## Why This Matters

Every collision on record so far happened between sessions that could have read each other's files. On 2026-09-01 a peer session re-fixed two issues that a paused branch had already fixed and not pushed. On 2026-09-09 two sessions held worktrees off one checkout, nothing pushed: one nearly minted a record its peer had already captured, and one nearly created a worktree over a peer's. On 2026-09-18, in a managed repository, a lane's `capture resolve` answered not found for a record that existed one worktree over, an audit agent working from a stale worktree reported three shipped intents as existing nowhere, and lanes merged each other's branches to resolve records. All of it is in [iss-2609020716570699](../../../work/issues/open/iss-2609020716570699-nothing-tells-an-agent-that-a-record-it-is-about-to-fix-has.md) and its corroboration. In each case the record existed on the machine, and nothing rendered it; the convention in `AGENTS.md` that asks a session to scan for peers points at a surface abcd never renders.

The claim record this was split from proposed a machine-scoped lease, a `claimed_by` stamp, refusals in the write verbs and a pushed duplicate guard. Two adversarial reviews on 2026-09-09 (their application is the DECISIONS.md ruling of that date) found the expensive parts unsound for a human session and found that the collisions on record needed none of it. The maintainer ruled a split into the listing (this record), the upstream-terminal refusal and the claim. On 2026-09-20 the product thinker settled the four records' shape (below): this listing is the read side, the claim the write side, the register where both live across machines, and the implement verb the consumer.

## Decisions

Settled with the product thinker on 2026-09-20:

1. **Four records, linked.** This listing (read), the claim `itd-2609091034175565` (write), the register `itd-2609150819440345` (where claims and holdings live, across accounts and across machines over a network such as a tailnet), and the implement verb `itd-2609201916151817` (claims, checks, implements or drops and picks the next). Each is planned on its own; this one first.
2. **The primitive: both sources through one reader.** The sibling worktrees' disks (sees uncommitted captures) and the local branches through the common git dir (sees a branch checked out nowhere), behind one reader the upstream-terminal refusal and, later, the register share.
3. **The home: the refusals, the board, and a standalone read command.** `capture resolve`, `abcd <record-id>` and `intent audit` name the peer holding a record they cannot find here; the board carries one line; one read-only command (working name `peers`; the name is settled at build time) prints the whole picture.
4. **Spent peers are skipped.** A worktree whose directory is gone, or whose branch is merged into the default branch, is left out of the diff and named in a count.
5. **A peer whose ledger holds one id in two status folders is marked, not read**, and the other peers render normally; the block names the id and the remedy.
6. **Issues and intent drafts** are both listed, by the same filename rule.
7. **Relation to `itd-148`/`spc-42`:** this is a distinct read primitive that the worktree render consumes; it does not ship that render early.
8. The always-on provenance line on every ledger verb is its own record, `iss-2609202053570475`.

## What's In Scope

- **One reader, two sources.** Given this checkout, the reader enumerates the peers git names (`git worktree list --porcelain`, each candidate confirmed from its own `--git-common-dir`; every local branch through the common dir) and, per peer, reads the ids under `.abcd/work/issues/{open,resolved,wontfix}/` and `.abcd/development/intents/drafts/` by filename, opening a record only for its title through the same guarded reader every record read uses.
- **The diff.** Per live peer: Records open there and absent here; records open here and terminal there; drafts there and absent here. Each id with its title where the file is readable, the id alone where it is not.
- **The not-found paths.** `capture resolve`, `abcd <record-id>` and `intent audit`, when the record is not in this tree and a peer holds it, refuse naming the peer's branch and path and the folder that holds it.
- **The board line.** The count of live peers and the count of ids across the diff, present only when the diff is non-empty.
- **The standalone command**, text and `--json`, home paths redacted on every stream.
- **`AGENTS.md`'s concurrent-sessions convention names the command** as the scan-before-mutating step.

## What's Out of Scope

- Any write: No claim, no lease, no stamp, no hook, no session key (the claim record's).
- Presence or liveness: A peer holding a record says the record exists, not that a session does.
- Peers beyond this checkout's common dir: A second clone, another account, another machine (the register's).
- A record in a stash or an unsaved buffer.
- A record already terminal on the default branch (the upstream-terminal refusal's).
- Judging that two differently worded records describe one observation (`itd-87`'s).
- Creating, moving or reclaiming worktrees (the worktree store's and `itd-148`'s).

## Mechanism

We expect a read-only view of what peers hold to stop the local collisions on record because each was a failure of visibility and not of will: The peer's record existed on the machine before the collision, and the verb that collided answered not found rather than naming it. It is falsified if a session that was shown a peer's holding, in a refusal, on the board or in the command's output, still minted a duplicate of it or re-fixed a record shown as terminal elsewhere; the render's own output and the two records' timestamp ids tell that case apart from a session that never looked. It is also falsified, differently, if the next collision on record comes from a peer this reader cannot see, a second clone or another machine, which is the register's case.

## Scope Conditions

- Holds for the peers that share this checkout's common git dir: Its worktrees and its local branches. A second clone, another account and another machine are the register's boundary. <!-- cond: cond-2609202056488592 -->
- Holds for a record on a peer's disk or in a peer's branch; a stash or an unsaved buffer is invisible. <!-- cond: cond-2609202056489916 -->
- Holds at the scale of this checkout, some thirty worktrees and a few hundred branches, each ledger some hundreds of records; behaviour past that, or on a network filesystem, is unmeasured. <!-- cond: cond-2609202056486227 -->
- Holds where git answers for the peer under abcd's isolated environment; a peer git refuses (another uid's checkout, unless `~/.abcd/trusted-roots` re-admits it) is named and not read. <!-- cond: cond-2609202056485097 -->
- macOS and Linux; the porcelain listing's path form on Windows is untested. <!-- cond: cond-2609202056483851 -->
- Assumes the peer's ledger uses the committed layout, so a peer at a commit before that layout existed contributes no rows and says so. <!-- cond: cond-2609202056489557 -->

## Acceptance Criteria

- **Given** a record captured under a sibling worktree's `open/` and absent from every status folder here, **when** the standalone command runs, **then** one row under that peer names the id and its title, the block names the peer's branch and home-redacted path, the exit code is 0, and `git status --porcelain` in both trees is byte-identical to before.
- **Given** a record open here and present under a peer's `resolved/` or `wontfix/`, **when** the command runs, **then** the record appears under that peer naming which terminal folder holds it.
- **Given** a record open here and resolved on a local branch that no worktree has checked out, **when** the command runs, **then** that branch appears as a peer and the record under it.
- **Given** an intent draft under a peer's `drafts/` with no draft of that id here, **when** the command runs, **then** it appears under that peer as a draft.
- **Given** `capture resolve <iss-N>` for a record not in this tree that a peer holds, **when** it runs, **then** the refusal names the peer's branch, path and folder instead of not found; the same for `abcd <iss-N>` and `intent audit <itd-N>`.
- **Given** a peer whose branch is merged into the default branch, or whose worktree directory is gone, **when** the command runs, **then** it contributes no rows and is counted in a skipped-peers line.
- **Given** a peer whose ledger holds one id in two status folders, **when** the command runs, **then** that peer's block says it was not read, names the id and the remedy, and every other peer renders normally.
- **Given** a directory `git worktree list` names whose own common dir is not this checkout's, or for which git refuses to answer, **when** the command runs, **then** it is not read and its block says why.
- **Given** a checkout with no peers, **when** the command runs, **then** it reports none, exits 0, and the board carries no line.
- **Given** a non-empty diff, **when** the board renders, **then** one line states the live-peer count and the id count; **given** an empty diff, the line is absent.
- **Given** `--json`, **when** the command runs, **then** the payload carries the same blocks, and no value carries an unredacted home path.
- **Given** a peer's record file that is unreadable or malformed, **when** the command runs, **then** the id appears without a title and the command exits 0.
- **Given** `AGENTS.md`'s concurrent-sessions convention, **when** it is read, **then** the scan-before-mutating step names the command beside the harness's session listing.

## Typed Links

- **builds_on `itd-2609091034175565`** (the claim): The record this was split from on 2026-09-09; the write side this read side answers.
- **refines `itd-2609091416304128`** (the upstream-terminal refusal): Shares the reader; it judges against the fetched default branch, this against local peers.
- **refines `itd-2609091014076309`** (the worktree store) and **`itd-148`** (every change in its own worktree): Their renders list worktrees; this reader gives them the ledger columns and does not ship their render.
- **built on by `itd-2609150819440345`** (the register): Adds the network source to this reader.
- **built on by `itd-2609201916151817`** (the implement verb): Reads this picture before it picks a record.
- Prose cross-references, because no schema field carries the relation (`iss-2609091256264547`): `iss-2609020716570699` (the collisions), `iss-2608220750029993` (the presence lease this is not).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: the pilot and the big run put four to seven lanes on one checkout this week, and every collision on record was a failure of visibility, not of will; we expect a read-only view of what peers hold, delivered in the verbs' own refusals, to stop them; shown wrong if a lane that was shown a peer's holding still duplicates or re-fixes it, or if the next collision comes from a peer this reader cannot see
