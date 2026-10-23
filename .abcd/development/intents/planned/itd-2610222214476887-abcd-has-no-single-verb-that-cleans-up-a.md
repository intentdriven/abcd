---
id: itd-2610222214476887
slug: abcd-has-no-single-verb-that-cleans-up-a
spec_id: spc-2610230730269295
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
refines: [itd-2609091014076309]
blocked_by: [itd-2609091014076309]
severity: minor
impact: additive
related_issues: [iss-2610222001116066]
origin: extracted-from-record
production_mode: hand-written
supersedes: [itd-118]
---

# One command shows everything a repository has left lying around, and clears what is provably finished

Typed links: `refines` and `blocked_by` [itd-2609091014076309](itd-2609091014076309-session-and-agent-worktrees-live-in-a.md) (the worktree store: its prune is the one reclaim this command calls, never a second copy); `supersedes` [itd-118](../superseded/itd-118-merged-work-leaves-no-residue-abcd.md) (the seed for tidying what a merge leaves behind, which this command absorbs; the product thinker confirmed it on 2026-10-23); graduated from [iss-2610222001116066](../../../work/issues/open/iss-2610222001116066-abcd-has-no-single-verb-that-cleans-up-a.md); reservations through [iss-2610142200389875](../../../work/issues/open/iss-2610142200389875-the-reservations-ruleset-forbids.md) (the reservation sweep).

## Press Release

> **`abcd sweep` shows everything a repository has left lying around, and clears only what is provably finished.** Run bare, it changes nothing. It lists every worktree, local branch, reservation and waiting inbox report, and beside each one what it would do or why it would leave it alone. Run with `--apply`, it removes the worktrees and branches whose work has merged and whose trees are clean, gets the spent reservations cleared, and pulls the inbox reports into the ledger. Anything dirty, unmerged, locked by another tool, or in use by a live session stays where it is, named with the reason. A session that died with uncommitted work has that work saved as a patch before anything else happens, so nothing a person might still need is ever lost.
>
> "After the machine crashed I had a hundred and ninety worktrees, eighty-six reservation branches and a pile of reports, and every one needed its own check," said Maya, autonomous-development practitioner. "Now one command tells me what is finished and what still needs me, and the second run clears the finished part. The worktree with a dead session's half-written fix in it stayed put, and its diff was saved where I could find it."

## Why This Matters

Cleanup in abcd is a set of separate jobs, and nothing joins them up. Worktrees pile up in the machine store, and the store's reclaim ([itd-2609091014076309](itd-2609091014076309-session-and-agent-worktrees-live-in-a.md)) is planned but not built. Merged branches stay behind ([itd-118](../superseded/itd-118-merged-work-leaves-no-residue-abcd.md) is a seed). Reservation branches are deleted only by a daily workflow ([iss-2610142200389875](../../../work/issues/open/iss-2610142200389875-the-reservations-ruleset-forbids.md)). Inbox reports wait until someone runs `/abcd:inbox`.

The 2026-10-22 crash recovery showed the cost:
- **Scale.** The checkout held 193 worktrees: 134 merged by ancestry, 5 merged by content, 54 unmerged. Origin held 86 reservation branches, 66 of them expired.
- **Time.** Triaging it by hand took an agent most of an evening. The result was a list of 131 worktrees safe to retire, which a person then had to approve one batch at a time.
- **Pitfalls.** The triage turned up the pitfalls a careless cleanup falls into, and each one is now a criterion below:
  - Worktrees a live session had made that evening looked merged, because they sat exactly on the default branch.
  - The embargo store is a worktree full of untracked private files, by design.
  - Twenty-six worktrees carried another tool's lock.
  - Two dead sessions had left real, unlanded work uncommitted, one of them a fix for a major issue that existed nowhere else.
  - One branch was merged by content but not by ancestry.

The person asked for one verb that does everything a good cleanup needs. The value is in two properties together: it sees everything, and it removes only what it can prove is finished.

## What's In Scope

- **`abcd sweep`, dry run by default.** It lists every worktree of this repository (in the store and outside it), every local branch, every reservation branch on origin, and every report waiting in the inbox. Each row says what `--apply` would do, or why it would leave the item. It writes nothing, and `--json` emits the same rows.
- **`abcd sweep --apply` acts only on what is provably spent:**
  - a worktree whose branch is merged, by ancestry or by patch-equivalence, and whose tree is clean. It is removed through the store's prune core, which archives its notes first.
  - a local branch merged the same way, with no worktree on it.
  - reservations, through the reservation-sweep workflow (below).
  - inbox reports (below);
  - a merged pull-request branch on origin that nothing still needs (below).
  Each removal is reported by name. Nothing is forced, so git's own re-check at the moment of removal closes the gap between judging an item and removing it.
- **Left for a person, named with the reason:**
  - a dirty tree;
  - unmerged or unpushed work;
  - a worktree git holds locked, naming the lock's owner where the reason says;
  - anything a live session holds;
  - a recently active worktree;
  - the embargo store;
  - anything the sweep cannot classify.
- **Live means proven.** A worktree counts as held by a live session when any of these name it:
  - a live reservation's footprint;
  - a running process's working directory;
  - the build loop's run state.
  Recent activity on its own is a hint, but it also keeps the worktree.
  Where the process check cannot be made (the tool that reads it is missing,
  times out or answers unreadably), every worktree it could not clear counts
  as possibly live and is left, named with that reason.
- **A dead session's work is saved first.** A dirty worktree no live session holds has its uncommitted diff and untracked files saved as a patch in the notes archive, `~/.abcd.noindex/notes/<root-sha>/`, the store's archive. The worktree itself is kept, and the report says where the patch went. The embargo store is never saved this way; nothing is copied out of it.
- **Merged pull-request branches on origin.** Under `--apply` a pull-request branch on origin whose pull request has merged is deleted once nothing still needs it: no open pull request heads it, no reservation names it, and no running session works on it. The report names each one. A branch the sweep cannot show is merged, the default branch, release and tag refs and reservation branches are never deleted.
- **Reservations are never deleted directly.** Only the reservation-sweep App may delete them. Under `--apply` the sweep dispatches the `reservation-sweep` workflow (it takes `workflow_dispatch`) and reports the run. Where it cannot dispatch, the report says when the daily run will clear them.
- **Inbox reports are filed into the ledger.** Under `--apply` every waiting report is promoted, each filed with the automatic filer's remedy so the drain never acts on one nobody has read. Promotion writes a record and runs nothing a report says, so a report's untrusted text stays inert.
- **Strays are listed, never deleted.** A file or directory in the store that is not a worktree, such as a loose diff, is listed as a stray.

## What's Out of Scope

- **Building the worktree store's reclaim.** That is [itd-2609091014076309](itd-2609091014076309-session-and-agent-worktrees-live-in-a.md), which this command waits on and calls.
- **Removing anything that is not provably finished**, by default or by flag. Quiet unmerged worktrees go only through the store's own named reclaim (`ahoy worktree prune --yes <name>`).
- **Lifting another tool's git lock.**
- **Running on its own.** The sweep runs when a person runs it. A scheduled or post-merge sweep is a later decision.
- **Other people's work on origin.** It never deletes a branch it cannot show is merged, and never one another person holds.

## Mechanism

We expect one dry-run-first command to replace the hand triage, because the triage was mostly mechanical: applying the same few proofs (merged, clean, not live, not locked, not special) to hundreds of items. The few judgement calls were all the same call: leave it and name it. It fails if a person still has to check items the sweep marked as spent, and we would see that as a person re-checking the dry run's spent rows by hand.

## Scope Conditions

- One repository at a time, on one machine, run by a person or an agent acting for them. <!-- cond: cond-2610230730267813 -->
- "Merged" is judged against the repository's default branch as just fetched from origin, by ancestry or by patch-equivalence. A branch rewritten after its merge stays unmerged. <!-- cond: cond-2610230730264974 -->
- Liveness can only be proven for sessions on this machine. A session on another machine is seen only through its reservation on origin. <!-- cond: cond-2610230730264235 -->

## Acceptance Criteria

- **Given** a repository with merged worktrees, merged branches, spent reservations and waiting inbox reports, **when** `abcd sweep` runs with no flag, **then** every item is listed with the action `--apply` would take or the reason it would be left, and nothing changes.
- **Given** a worktree whose branch is merged into the default branch by patch-equivalence but not by ancestry, and whose tree is clean, **when** `abcd sweep --apply` runs, **then** it is removed and named in the report.
- **Given** a worktree whose tip is exactly the default branch and in which a running process has its working directory, **when** `abcd sweep --apply` runs, **then** it is left and named as held by a live session, although it reads as merged.
- **Given** a dirty worktree no live session holds, **when** `abcd sweep --apply` runs, **then** its uncommitted diff and untracked files are saved as a patch in the notes archive, the worktree is kept, and the report names where the patch went.
- **Given** the embargo store, a detached worktree with untracked private files, **when** `abcd sweep` runs with or without `--apply`, **then** it is listed as the embargo store and left alone, and nothing is copied out of it.
- **Given** a worktree another tool has locked with git, **when** `abcd sweep --apply` runs, **then** it is left, naming the lock and its owner where the reason gives one.
- **Given** a loose file in the store that is not a worktree, **when** `abcd sweep --apply` runs, **then** it is listed as a stray and left.
- **Given** a pull-request branch on origin whose pull request has merged, with no open pull request heading it, no reservation naming it and no running session working on it, **when** `abcd sweep --apply` runs, **then** it is deleted on origin and named in the report; **given** any of those still holds it, **then** it is listed and left, naming what holds it.
- **Given** spent reservation branches on origin, **when** `abcd sweep --apply` runs, **then** the sweep dispatches the `reservation-sweep` workflow and reports the run, and deletes no reservation branch itself; **given** the workflow cannot be dispatched, **then** the report says when the daily run will clear them.
- **Given** waiting inbox reports, **when** `abcd sweep --apply` runs, **then** every one is promoted into the ledger, each filed with the automatic filer's remedy, and the report names each record it filed.
- Technical:
  - Worktree removal goes through the store's prune core. There is one reclaim primitive.
  - The sweep judges against the default branch freshly fetched from origin.
  - It never forces a removal and never stops a process.
  - `--json` emits the rows.
  - Exit codes: 0 when something was acted on, 1 when the run completed and acted on nothing, 2 on a refusal or a fault.
  - It writes only under `~/.abcd.noindex/` and the checkout's local tier.

## Open Questions

_None open; both were answered by the product thinker on 2026-10-23 (Decisions)._

## Decisions

- Refines and is blocked by the worktree store (itd-2609091014076309): one reclaim primitive, the store's prune, which this command calls. Decided by the technical facilitator on delegation, on the one-canonical-primitive principle.
- Reservations are never deleted directly. `--apply` dispatches the reservation-sweep workflow and reports the run, or the report says when the daily run will clear them. Decided by the technical facilitator on delegation (the 2026-10-15 ruling that abcd resolves a gap itself rather than hand the person a manual step).
- Dry run by default. `--apply` acts only on provably spent items, and liveness is proven by a reservation, a process's working directory or run state. Decided by the technical facilitator on delegation, from the 2026-10-22 crash triage.
- The process check reads `/proc/<pid>/cwd` on Linux and runs `/usr/sbin/lsof` by absolute path on macOS, with a deadline, a bounded read and defensive parsing. It never uses cgo, since the released binaries are cross-compiled. It fails closed: a missing, timed-out or unreadable probe leaves the worktree in place as possibly live. Decided by the technical facilitator on delegation.

Ruled by the product thinker in the planning walk on 2026-10-23, relayed by the coordinator:

- **Inbox:** `--apply` files every waiting report, each with the automatic filer's remedy, rather than only reports named on the command line.
- **The older merged-branch seed (itd-118) is superseded** by this intent, which covers all it asked for.
- **Merged pull-request branches on origin are cleared under `--apply`** once nothing still needs them (no open pull request, no reservation, no running session), and named in the report; there is no separate opt-in for them. This replaces the delegated decision above that held origin branches behind `--remote`.
- **Expired and released reservations** are cleared by the reservation-sweep workflow the sweep starts, which deletes them right then; the product thinker kept that over the sweep deleting them itself.
- The press release, mechanism, scope conditions and every acceptance criterion stand as written, the merged-branch criterion as edited above, and the technical list is accepted whole.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: after a crash, one command shows what is finished and what still needs the person; shown wrong if they still have to check the finished items by hand
- pursued: cleanup stops taking an evening; shown wrong if a cleanup ever removes work someone needed
