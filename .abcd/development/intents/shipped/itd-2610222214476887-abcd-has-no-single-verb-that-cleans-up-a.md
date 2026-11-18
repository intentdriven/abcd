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

Typed links: `refines` and `blocked_by` [itd-2609091014076309](itd-2609091014076309-session-and-agent-worktrees-live-in-a.md) (the worktree store: its prune is the one reclaim this command calls, never a second copy); `supersedes` [itd-118](../superseded/itd-118-merged-work-leaves-no-residue-abcd.md) (the seed for tidying what a merge leaves behind, which this command absorbs; the product thinker confirmed it on 2026-10-23); graduated from [iss-2610222001116066](../../../work/issues/resolved/iss-2610222001116066-abcd-has-no-single-verb-that-cleans-up-a.md); reservations through [iss-2610142200389875](../../../work/issues/resolved/iss-2610142200389875-the-reservations-ruleset-forbids.md) (the reservation sweep).

## Press Release

> **`abcd sweep` shows everything a repository has left lying around, and clears only what is provably finished.** Run bare, it changes nothing. It lists every worktree, local branch, reservation and waiting inbox report, and beside each one what it would do or why it would leave it alone. Run with `--apply`, it removes the worktrees and branches whose work has merged and whose trees are clean, gets the spent reservations cleared, and pulls the inbox reports into the ledger. Anything dirty, unmerged, locked by another tool, or in use by a live session stays where it is, named with the reason. A session that died with uncommitted work has that work saved as a patch before anything else happens, so nothing a person might still need is ever lost.
>
> "After the machine crashed I had a hundred and ninety worktrees, eighty-six reservation branches and a pile of reports, and every one needed its own check," said Maya, autonomous-development practitioner. "Now one command tells me what is finished and what still needs me, and the second run clears the finished part. The worktree with a dead session's half-written fix in it stayed put, and its diff was saved where I could find it."

## Why This Matters

Cleanup in abcd is a set of separate jobs, and nothing joins them up. Worktrees pile up in the machine store. The store's own reclaim ([itd-2609091014076309](itd-2609091014076309-session-and-agent-worktrees-live-in-a.md)) has shipped, but it clears only the worktrees in one repository's lane, and none of the other leftovers below. Merged branches stay behind ([itd-118](../superseded/itd-118-merged-work-leaves-no-residue-abcd.md) is a seed). Reservation branches are deleted only by a daily workflow ([iss-2610142200389875](../../../work/issues/resolved/iss-2610142200389875-the-reservations-ruleset-forbids.md)). Inbox reports wait until someone runs `/abcd:inbox`.

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

- **Building the worktree store's reclaim.** That is [itd-2609091014076309](itd-2609091014076309-session-and-agent-worktrees-live-in-a.md), which has shipped and which this command calls.
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

## SOTA

Surveyed on 2026-10-23 for the seed this intent supersedes,
[itd-118](../superseded/itd-118-merged-work-leaves-no-residue-abcd.md), and
carried here when it was superseded; sources, evidence tiers and the local fit
risks are in the [forge, CI and merge SOTA note](../../research/notes/2026-10-23-forge-ci-merge-sota.md)
(its itd-118 section). Where the path below differs from this intent's
rulings (the sweep is its own verb, and merged is judged against the freshly
fetched default branch by ancestry or patch-equivalence), the rulings stand.

- **The remote half is one forge setting.** GitHub's "Automatically delete
  head branches" is set by an admin; branch protection and repository rules
  "can also prevent branches being automatically deleted"; on a fork pull
  request the fork's setting counts; a deleted head branch is restorable from
  the closed pull request (GitHub docs). GitLab's project default offers
  deletion for "non-protected source branches" and is overridable per merge
  request (the merge request that added it).
- **The CLI tidies both ends at merge time**: `gh pr merge --delete-branch`
  deletes "the local and remote branch after merge".
- **Git's own verbs cover the tracking ref and the worktree, each with a
  refusal**: `git fetch --prune` (or `fetch.prune`) removes remote-tracking
  refs only and "doesn't actually care about branches"; `git worktree remove`
  refuses an unclean worktree without `--force` and a locked one without
  `--force` twice; `git worktree prune` clears administrative files left by a
  hand-deleted worktree.
- **Squash and rebase merges leave no merge-base evidence**, so a local
  branch cannot be proven merged from git alone; the forge's pull-request
  state is the authority for "this branch landed" (inference from the merge
  methods this repository allows).

**Path 1**: adopt the forge setting (turned on by itd-106's reconciliation),
`--delete-branch` at land time and `fetch.prune`; abcd's native part is the
worktree in its own machine-scoped store, removed only when the forge
reports the branch merged and the tree is clean, with a report naming what
was left (dirty, locked, or no pull request). The independent fit-challenge is still owed before the build, and the questions below are kept for it.

Questions this pass raised for the fit-challenge before the build. They are kept here, not under Open Questions, so they do not hold a build the record already rules ready:

- What the sweep says about a branch with no pull request on the forge:
  "unknown", never "stale", under loud-staging.

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

<!-- abcd-review: INGESTED receipt=rcp-fe7b7d5fc3f5 -->
Fidelity review — receipt rcp-fe7b7d5fc3f5 (verifier intent-auditor claude-opus-5-5).

Provenance: intent-auditor@claude-opus-5-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:4160b7c451b574170d076c9ba19b604f197af248836c6b8e06c72fcee3f715fd
Input attestations: diff:spc-2610230730269295 landed commits 5ce7c0e88 f9df29257 527becba3 bd0524e47 6c00c054c ab6f7a214 a81c24e06 d4a18d9e9, spec close 4ce803a4a@-; repo:origin/main cb2e6157c (worktree audit-sweep-1118)@-;

Acceptance rollup: MET 6 · MET_WITH_CONCERNS 5 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: A bare run lists all five kinds with an action or keep reason, and tests hash the repo and home before and after and dispatch nothing. Concern: 'nothing changes' holds except for the fetch. Run does `fetch --prune` and the reservation fetch first, which moves remote-tracking refs, and the whole-dry-run tests call runSweep, which skips the fetch.
  evidence: internal/core/sweep/runsweep_test.go:75 — "if len(seen) != 5 {"
  evidence: internal/core/sweep/runsweep_test.go:162 — "func TestAWholeDryRunChangesNothing(t *testing.T) {"
  evidence: internal/core/sweep/sweep.go:372 — "gitutil.NetInputContext(ctx, repoRoot, "", "fetch", "--prune", "--quiet", claim.Remote)"
  evidence: .abcd/development/brief/04-surfaces/40-sweep.md:7 — "Bare, it changes nothing"
- ac-2 — MET: A squash-merged (patch-equivalent, not ancestor) clean worktree is judged 'merged, judged by content', removed through PruneOne, and named with outcome removed.
  evidence: internal/core/sweep/apply_test.go:54 — "func TestApplyReclaimsASquashMergedWorktreeThroughThePrune(t *testing.T) {"
  evidence: internal/core/sweep/apply_test.go:67 — "Detail: "merged, judged by content""
  evidence: internal/core/sweep/apply.go:134 — "if pr.Outcome == store.OutcomeReclaimed {"
- ac-3 — MET: A worktree at the default branch's tip with a process cwd inside it is kept with reason 'in use: process … works in it'. Liveness holds rank ahead of every merged or clean judgement. Under --apply, a fresh probe runs immediately before each removal (PreRemove). The store reads a worktree with no commit of its own as unmerged anyway, so the keep holds either way.
  evidence: internal/core/sweep/sweep_test.go:426 — "func TestAWorktreeOnTheDefaultBranchWithALiveDirectoryIsKept(t *testing.T) {"
  evidence: internal/core/sweep/sweep.go:580 — "case len(reasons) > 0:"
  evidence: internal/core/sweep/apply.go:106 — "PreRemove: func(abs string) (bool, string) {"
  evidence: internal/core/sweep/apply_test.go:89 — "func TestAWorktreeThatGainsALiveProcessBeforeRemovalIsKept(t *testing.T) {"
- ac-4 — MET_WITH_CONCERNS: For a dirty store worktree, the unstaged and staged patches and a tar of untracked files are written under the notes archive with a manifest. The worktree stays as it was, and the row names the home-redacted entry. Concern: this is narrower than 'a dirty worktree'. A dirty worktree outside the store is kept unsaved. So are a change git writes as a binary patch and a worktree mid-merge.
  evidence: internal/core/sweep/salvage_test.go:106 — "func TestApplySavesADirtyWorktreesChangesAndKeepsIt(t *testing.T) {"
  evidence: internal/core/sweep/salvage_test.go:133 — "r.Outcome != OutcomeSaved || !strings.Contains(r.Reason, fsutil.RedactHome(entry))"
  evidence: internal/core/sweep/sweep.go:582 — "case r.Where != store.WhereStore:"
  evidence: internal/core/sweep/salvage_test.go:421 — "func TestAChangeGitWritesAsABinaryPatchIsRefused(t *testing.T) {"
- ac-5 — MET: The embargo store is matched first, by storepath.IsStorePath or the sibling location, and kept as 'never read, saved or removed'. Tests show the probe never asks about it, and an apply over a detached embargo worktree holding an untracked file writes no salvage.
  evidence: internal/core/sweep/sweep.go:530 — "if r.Where == store.WhereSibling || storepath.IsStorePath(j.home, abs) {"
  evidence: internal/core/sweep/salvage_test.go:308 — "func TestTheEmbargoStoreAndAnOutsideWorktreeAreNeverSalvaged(t *testing.T) {"
  evidence: internal/core/sweep/sweep_test.go:291 — "if slices.Contains(asked, real) || slices.Contains(asked, emb) {"
- ac-6 — MET: A git-locked worktree is kept with the lock's whole reason. A JSON reason's owner is named ('locked by agent-tool'). Apply never asks the prune about it.
  evidence: internal/core/sweep/sweep.go:608 — "func lockedBy(reason string) string {"
  evidence: internal/core/sweep/sweep_test.go:267 — "kept(t, r, "locked by agent-tool")"
  evidence: internal/core/sweep/apply_test.go:202 — "kept(t, rowFor(t, res, "locked"), "locked by git: on a removable disk")"
- ac-7 — MET: A non-directory entry in the store lane is a stray row, kept as 'not a worktree; listed and never touched'. Apply acts only on remove and save-patch rows, so the stray stays.
  evidence: internal/core/sweep/sweep.go:537 — "row.Kind, row.State = RowStray, "an entry in the store's lane that is not a directory""
  evidence: internal/core/sweep/sweep_test.go:296 — "t.Run("a stray", func(t *testing.T) {"
  evidence: internal/core/sweep/apply.go:71 — "switch row.Action {"
- ac-8 — MET_WITH_CONCERNS: A merged origin branch is deleted under a lease and named, and one an open PR heads (or targets as its base) or a live local session holds is kept, naming the hold. Concern: 'no reservation naming it' is narrowed. A reservation holds an origin branch only through a local worktree of the copy that made it, or through its footprint over a local worktree on that branch. A reservation made on another machine or account adds no hold (iss-2611161232540905 open, dcn-2611161144499939).
  evidence: internal/core/sweep/remote_test.go:162 — "func TestAMergedBranchOnOriginIsDeletedUnderALeaseAndNamed(t *testing.T) {"
  evidence: internal/core/sweep/remote_test.go:253 — "func TestAMergedBranchSomethingStillNeedsIsKeptNamingIt(t *testing.T) {"
  evidence: internal/core/sweep/remote.go:540 — "reservation holds only the branch its own copy is on: holds finds it"
  evidence: .abcd/work/issues/open/iss-2611161232540905-a-reservation-records-no-branch-only-the.md:15 — "a reservation made on another machine or under another account holds no branch on origin"
- ac-9 — MET: Under apply, spent reservations trigger `gh workflow run reservation-sweep.yml` and the run listed afterwards is reported. A test proves the reservation's tip on origin is unchanged. With gh absent or the dispatch refused, the report names the next scheduled time from the workflow's cron. The workflow carries workflow_dispatch.
  evidence: internal/core/sweep/reservations.go:270 — "forgeCall(ctx, gh, root, "workflow", "run", reservationWorkflow, "--ref="+def)"
  evidence: internal/core/sweep/reservations_test.go:242 — "func TestApplyLeavesASpentReservationOnOrigin(t *testing.T) {"
  evidence: internal/core/sweep/reservations_test.go:339 — "func TestWithNoForgeClientTheNextScheduledTimeIsGiven(t *testing.T) {"
  evidence: .github/workflows/reservation-sweep.yml:29 — "workflow_dispatch:"
- ac-10 — MET_WITH_CONCERNS: In abcd's own checkout, every waiting report is promoted through report.Promote with issueschema.MachineRemedy, and the row names the record filed. Concern: in any other repository, every report is kept ('promoted only in abcd's own checkout'). 'Every one is promoted' therefore holds for abcd's checkout only.
  evidence: internal/core/sweep/inbox_test.go:113 — "func TestEveryWaitingReportIsPromotedWithTheAutomaticFilersRemedyAndNamed(t *testing.T) {"
  evidence: internal/core/sweep/inbox.go:117 — "filed := "promoted: filed as " + p.Capture + " at " + p.Path + ", with the remedy ""
  evidence: internal/core/sweep/inbox.go:151 — "a report is about abcd and is promoted only in abcd's own checkout"
  evidence: internal/core/sweep/inbox_test.go:258 — "func TestAReportIsPromotedOnlyInAbcdsOwnCheckout(t *testing.T) {"
- ac-11 — MET_WITH_CONCERNS: Removal is store.PruneOne only. A fetch of origin and the reservations comes first, and a failed fetch refuses the run. A local branch is deleted with `branch -d` and an origin branch only with --force-with-lease on the judged tip; no `worktree remove --force` and no process kill. --json is the root persistent flag. Codes are 0/1/2. Concerns: a bare run exits 0 when --apply would act, not when something was acted on. An applied run also writes outside ~/.abcd.noindex/ and the local tier: ledger records (ac-10), local branch refs and origin deletes, which the brief says only for the bare form.
  evidence: internal/core/sweep/apply.go:52 — "prune = store.PruneOne"
  evidence: internal/core/sweep/sweep.go:368 — "func fetchFirst(ctx context.Context, repoRoot string) (fetched bool, err error) {"
  evidence: internal/core/sweep/branches.go:329 — "gitutil.LocalGit(bj.root, "branch", "-d", "--", name)"
  evidence: internal/core/sweep/rows.go:228 — "case !r.Applied && row.Action != ActionKeep && row.Action != "":"
  evidence: internal/surface/cli/sweep_test.go:92 — "func TestSweepVerbRowsJSONAndExitCodes(t *testing.T) {"
  evidence: .abcd/development/brief/04-surfaces/40-sweep.md:83 — "Bare, it writes only under the machine's abcd home and the local tier,"

Gap audit:
- honoured:
  - Dry run by default; --apply acts only on what is provably spent
    evidence: internal/surface/cli/sweep.go:100 — "act on what is provably spent; without it the sweep changes nothing but the fetch"
  - One reclaim primitive: the store's prune, which archives notes first
    evidence: internal/core/sweep/apply.go:124 — "got, err := prune(repoRoot, filepath.Base(filepath.Clean(src.Abs)), opts)"
  - A dead session's dirty work is saved as a patch before anything else, and the worktree kept
    evidence: internal/core/sweep/salvage.go:145 — "its changes are saved as a patch to the notes archive, and the worktree kept"
  - Reservations are never deleted directly; the workflow is dispatched or its schedule named
    evidence: internal/core/sweep/reservations_test.go:370 — "func TestAFailedDispatchFallsBackToTheSchedule(t *testing.T) {"
  - Live means proven: reservation, process cwd and build-run proofs, failing closed when the process check cannot be made
    evidence: internal/core/sweep/sweep_test.go:450 — "func TestAFailedProcessCheckKeepsEveryWorktreeAndSaysSoOnce(t *testing.T) {"
  - A report's untrusted text stays inert
    evidence: internal/core/sweep/inbox_test.go:217 — "func TestHostileReportTextIsInertInTheRow(t *testing.T) {"
- diverged:
  - Inbox reports are filed into the ledger: only in abcd's own checkout; elsewhere every report is kept
    evidence: internal/core/sweep/inbox.go:151 — "a report is about abcd and is promoted only in abcd's own checkout"
  - No reservation names it: a reservation holds an origin branch only through a local worktree of its copy
    evidence: .abcd/development/brief/04-surfaces/40-sweep.md:52 — "holds the branch of a local worktree that copy is on; a reservation made on"
  - Exit 0 when something was acted on: a bare run exits 0 when --apply would act
    evidence: .abcd/development/brief/04-surfaces/40-sweep.md:81 — "It exits 0 when the run acted on something (bare: when the applying"
  - Running on its own is out of scope: an automatic sweep after landings and drains shipped under a later ruling
    evidence: internal/core/sweep/sweep.go:413 — "// runHalves is runSweep, with automatic the sweep abcd runs on its own"
  - Every worktree in and outside the store is judged: worktrees outside the store are listed but never removed or salvaged
    evidence: internal/core/sweep/sweep.go:583 — "return keep("outside the store; never touched", ev...)"
  - Writes only under ~/.abcd.noindex/ and the local tier: an applied run files ledger records and deletes refs
    evidence: .abcd/development/brief/04-surfaces/40-sweep.md:86 — "own checkout, files the waiting inbox reports as ledger records."
- missing:
  - Never deletes a branch another person holds: a reservation made on another machine or account adds no hold on a merged origin branch (reservation records no branch; remedy open)
    evidence: .abcd/work/issues/open/iss-2611161232540905-a-reservation-records-no-branch-only-the.md:15 — "for a merged branch it still works on the reservation adds no hold"

Scope-condition dispositions:
- cond-2610230730267813 — survived: The verb resolves one checkout's toplevel and sweeps that repository alone, and the process probe reads this machine's /proc or lsof only.
  evidence: internal/surface/cli/sweep.go:111 — "root, err := gitutil.Toplevel(cwd)"
  evidence: internal/core/sweep/liveness.go:682 — "var hostLsof = lsofCommand{path: "/usr/sbin/lsof", timeout: 5 * time.Second, limit: 4 << 20}"
- cond-2610230730264974 — survived: Merged is judged after a fresh `fetch --prune` against the default ref as fetched, by ancestry or patch-equivalence. The store also accepts the forge's MERGED record, but only at the exact current tip, so a branch rewritten after its merge stays unmerged.
  evidence: internal/core/sweep/sweep.go:372 — ""fetch", "--prune", "--quiet", claim.Remote"
  evidence: internal/core/ahoy/worktree/merged.go:154 — "// The forge's answer counts only at the local tip: a branch that gained"
  evidence: internal/core/ahoy/worktree/merged.go:166 — "if ok, err := gitutil.IsAncestor(dir, tip, def); err == nil && ok {"
- cond-2610230730264235 — narrowed: For worktrees, another machine's session is seen through its live reservation's footprint overlapping a local worktree's changes. For branches on origin, a reservation holds only through a local worktree on that branch, matched by the copy that made it or by that footprint overlap. A reservation records no branch, so one made on another machine or account adds no hold on a merged origin branch.
  narrowing: Holds for worktrees on this machine, which a remote reservation's footprint can hold. For branches on origin, a session elsewhere is seen only through a local worktree on that branch. A reservation from another machine or account with no such local worktree adds no hold, and the merged branch is deleted (iss-2611161232540905 open).
  evidence: internal/core/sweep/liveness.go:365 — "// ways is named once. A reservation carries no branch, so none is matched to"
  evidence: internal/core/sweep/remote.go:520 — "for _, wt := range rj.wts {"
  evidence: .abcd/development/brief/04-surfaces/40-sweep.md:53 — "another machine or under another account holds no branch on origin, so"
  evidence: .abcd/work/decisions/dcn-2611161144499939-decided-by-the-technical-facilitator.md:6 — "Consequence: a reservation made on another machine holds no branch on origin by name"
<!-- abcd-review-end receipt=rcp-fe7b7d5fc3f5 -->

## Grounds

- pursued: after a crash, one command shows what is finished and what still needs the person; shown wrong if they still have to check the finished items by hand
- pursued: cleanup stops taking an evening; shown wrong if a cleanup ever removes work someone needed
