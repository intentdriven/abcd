---
id: itd-2609091014076309
slug: session-and-agent-worktrees-live-in-a-machine-scoped-store-t
spec_id: spc-2609301811532881
kind: standalone
suggested_kind: null
reclassification_history: []
refines: [itd-2609201916151817]
related_intents: [itd-118, itd-148]
severity: major
impact: additive
origin: researcher-authored
production_mode: hand-written
related_adrs: [adr-2609091248200336, adr-2609091248201071]
---

# Session and agent worktrees live in a machine-scoped store that abcd lists and reclaims, never beside the user's own projects

Typed links: `related_adrs` [adr-2609091248200336](../../decisions/adrs/2609091248200336-a-tool-never-creates-directories-in-user-owned-project-space.md) (the rule this store enacts), [adr-2609091248201071](../../decisions/adrs/2609091248201071-the-transcript-corpus-is-a-sibling-store-that-creates-itself.md) (the root-SHA-keyed sibling store whose shape this copies); `refines` [itd-2609201916151817](itd-2609201916151817-one-verb-takes-a-single-intent-from-ready-to-delivered-witho.md) (the build loop, whose lane primitive becomes this store's one primitive: its spec's piece 6 takes the store's verb once this ships); `related_intents` [itd-118](../drafts/itd-118-merged-work-leaves-no-residue-abcd-managed-repos-delete-a-pr.md) (the post-merge tidy that will call `prune`) and [itd-148](itd-148-every-change-starts-in-its-own-worktree-the-primary-checkout.md) (every change in its own worktree, which uses this store's verbs and is `blocked_by` this draft).

## Press Release

> **abcd keeps every session and agent worktree in one place on your machine, and can tell you what is there.** `abcd ahoy worktree add <name>` creates a worktree at `~/.abcd.noindex/worktrees/<root-sha>/<name>/`, keyed on the repository's root commit the way the history and transcript stores already are, and writes nothing beside your checkout. Bare `abcd ahoy worktree` lists what the store holds for this repository — name, branch, path, whether the tree is clean, whether its branch has merged — and names any worktree of the repository that sits outside the store; `abcd ahoy worktree list --all` walks every lane on the machine. `abcd ahoy worktree prune` reclaims a worktree whose branch has merged and whose tree is clean, moves the worktree's private notes into a dated archive rather than deleting them, reports each removal by name, and leaves everything else where it is, with the reason; a worktree that has gone quiet without merging is pointed out with a summary and goes only when you name it. The `/abcd` status board carries one line: how many worktrees the store holds for this repository and how many are reclaimable. Your project directory holds what you put there.
>
> "I came back from a run to find twenty-two new folders beside my projects, none of which I had made, and no way to tell which were still in use," said Maya, autonomous-development practitioner. "The isolation was right — parallel agents need separate checkouts. The location was not. Now the checkouts live where abcd keeps the rest of its machine state, one command lists them, one command clears the ones whose work has merged, and my folder is mine again."

## Why This Matters

Parallel agents need separate checkouts: `AGENTS.md`'s concurrent-sessions convention makes the checkout the unit of isolation, and the record's gates read the whole tree, so two sessions in one checkout fail each other's gates in both directions. Nothing says where a checkout goes, so each agent picks, and what an agent picks is git's default, a sibling directory. On 2026-09-01 twenty-one spent worktrees (1.4 GB) were removed by hand from the product thinker's project directory ([iss-2609020721142452](../../../work/issues/resolved/iss-2609020721142452-worktrees-for-parallel-lanes-are-created-one-directory-above.md)); on 2026-09-06 one session created twenty-two more, beside four unrelated projects, and `git worktree list` on that checkout named twenty-seven. The product thinker's objection, verbatim: "I don't want a user to be surprised that a folder is all of a sudden full of stuff."

That objection is the rule [adr-2609091248200336](../../decisions/adrs/2609091248200336-a-tool-never-creates-directories-in-user-owned-project-space.md) records: a tool never creates directories in space that belongs to the user, and agent and session scratch is machine-scoped. This intent is the capability that makes the rule followable — an agent that has somewhere to put a worktree, and a user who can see and reclaim what was put there.

The shape is not new. `~/.abcd.noindex/history/<root-sha>/`, `~/.abcd.noindex/transcripts/<root-sha>/` and `~/.abcd.noindex/voyage/<root-sha>/` are each keyed on the repository's root-commit SHA, and `internal/core/history/location.go` states why: a checkout moves, is renamed, and is cloned twice on one machine, while its root commit changes under none of that. `index.json` is the sole user-scope registry of every repository abcd knows on the machine, so a cross-repository listing is a walk that already exists. There is a de facto precedent on disk as well: `~/.abcd.noindex/worktrees/488a0aa9/phase-9` and `.../park-gates`, laid by hand by an autonomous run under abcd's own root commit before the verb exists (DECISIONS.md, 2026-09-26). That is corroboration that the location is the natural one, not authority for the shape — the precedent abbreviates the key, and this store uses the full SHA the sibling stores use.

Two things separate the store from the same pile somewhere less visible, and the intent owns both.

**Discoverability.** A worktree under `~/.abcd.noindex/` is invisible to `ls` where the user works, which is the point, and so it has to be visible somewhere else. Git already knows every worktree of a checkout wherever it sits (`git worktree list --porcelain`), so the store adds no registry of its own. What stands in for one is a proof git can give: a worktree belongs to the store when git lists it as a worktree of this repository, its real path lies inside this repository's lane, and its own common directory is this checkout's. It adds a verb that reads git's answer and says which entries are in the store, which are outside it, and what state each is in; and a line on the `/abcd` status board, so a user who never types the verb still learns the count.

**Lifecycle.** A worktree whose branch has merged is garbage, and nothing today notices. Reclaiming is `abcd ahoy worktree prune`: it removes a worktree only when its branch is merged into the default branch and its tree is clean, moves the worktree's local tier into the notes archive first, and names what it declined and why. It runs when the user runs it; the status board's reclaimable count is what prompts them. Reclaiming automatically on merge is [itd-118](../drafts/itd-118-merged-work-leaves-no-residue-abcd-managed-repos-delete-a-pr.md)'s post-merge tidy; when that ships it calls this verb rather than inventing a second reclaim.

## What's In Scope

- **`abcd ahoy worktree add <name> [--branch <branch>]`** creates a worktree at `~/.abcd.noindex/worktrees/<root-sha>/<name>/` for the repository the caller stands in, on a new branch named after the worktree unless `--branch` names an existing one, and prints the path. The store is created on first use the way the transcript store is: each level made individually and re-verified as a real directory, never through a symlink, never with a single recursive make, and each level owned by the caller with no group or other write bit. A name that is not one plain path segment (`../x`, `-x`, an empty or dotted name) refuses, a name already in the lane refuses, and a repository with no root commit refuses. The build loop's lane code (`laneWorktree`, `ensureStore`, `adoptWorktree` in `internal/core/implement/loop/lane.go`) already does this for its own lanes; it moves into the store's package and the loop calls it, so there is one primitive and not two. Nothing is written beside the checkout.
- **Bare `abcd ahoy worktree` and `abcd ahoy worktree list`** render the lane for this repository: name, branch, path, clean or dirty, merged or unmerged. On a machine stream a store row's path is home-redacted and an outside row's path uses the display form that stays recognisable wherever it lives. Every worktree git reports for the checkout is a row; one outside the store is marked as outside and never touched. `--all` walks every lane under `~/.abcd.noindex/worktrees/`, labelling each through `~/.abcd.noindex/history/index.json` (read through the loader that already scrubs it) where the root commit is registered and by SHA where it is not; the worktree's own `.git` file, not the registry's path label, says which checkout a lane belongs to. A lane whose checkout has moved, so that its `.git` pointer is dead, renders as "checkout not found; run `git worktree repair` from the checkout" under its SHA, with no clean or merged column. A directory under `~/.abcd.noindex/worktrees/` whose name is not a full root commit (the hand-laid eight-digit form) renders as a "not a lane" row rather than being skipped, naming this repository's full-key lane where the abbreviation matches it and saying its worktrees are retired by hand. `--json` emits the same rows.
- **`abcd ahoy worktree prune [--dry-run]`** removes each worktree that belongs to the store (by the proof above) whose branch is merged into the repository's default branch and whose tree is clean, moves its local tier (below) into the notes archive, runs git's own metadata prune for it, and reports each removal by name. A dirty tree, an unmerged branch, a worktree outside the store, a directory in the lane that fails the proof, a worktree git holds locked (with git's lock reason), the worktree prune is run from, and a lane the build loop's state file still holds are each left in place and named with the reason. Prune never forces a removal, so git's own re-check at the moment of removal closes the gap between judging a worktree clean and removing it. `--dry-run` removes nothing and reports the same rows; `--json` emits them. The branch is left standing. "Merged" means the forge records the branch's pull request as merged, or every commit of the branch is already on the default branch by content (patch-equivalence); with no forge reachable, patch-equivalence alone decides. Exit codes: 0 when at least one worktree was reclaimed, 1 when the run completed and reclaimed nothing (an empty lane included), 2 on a refusal or fault.
- **The notes archive.** A worktree's `.abcd/.work.local/` is ignored by git, so a tree git calls clean can still hold a handover note, a per-machine banlist, logs and scratch. Prune never deletes it: before removing the worktree it moves the tier into a sibling store, `~/.abcd.noindex/notes/<root-sha>/<UTC timestamp>-<worktree name>/`, created through the same canonical directory primitive as the transcript store (each level `0o700`, never through a symlink), redacted on write the way transcripts are, and accompanied by a manifest recording the worktree name, branch, source path (home-redacted), time and file list. The report names where the notes went. If the move fails, the worktree is kept and named with the reason.
- **Quiet worktrees.** An unmerged worktree in the lane with no activity for 14 days (repo-configurable) and no open pull request (where a forge is reachable; without one, the 14 days alone) is listed by `prune` as a candidate with a dossier: dirty state first, then branch subjects, diff summary, referenced records and last activity, and a recommendation. It is removed only when named on the command line (`prune --yes <name>`, repeatable); the core never prompts. A named removal takes the same notes-archive step and the same refusals as a merged one, a dirty tree included: prune never forces, so uncommitted work is committed or discarded by the person first, and the dossier shows it so they know.
- **One line on the `/abcd` status board**: the count of worktrees in the lane and the count reclaimable, absent when the lane is empty. Both counts come from the worktree and merged-branch scan the bare board already runs for `peers`, never from a status read per lane, and the line is the CLI board's alone: it is not part of the status block the site renders.
- **The concurrent-sessions convention in `AGENTS.md` points at the verb, and the plugin page carries it as a host-run step**: an agent that needs an isolated checkout (a lane, a subagent, a peer session) runs `abcd ahoy worktree add <name>` and works in the printed path, never a host's default worktree location. The prose stays host-agnostic.
- **The worktree directory is `0o700`**, like the store levels above it: `add` sets the mode after git creates the directory.

## What's Out of Scope

- **Deleting branches, tracking refs or remote branches.** That is the rest of itd-118's post-merge tidy; prune reclaims the directory and git's record of it, and nothing else.
- **Moving a worktree that already sits beside a checkout.** The store never moves what it did not create. A sibling worktree is listed as outside and left alone; retiring it is the user's own `git worktree remove`, and the sibling worktrees already on the product thinker's machine are a hand cleanup, not a migration.
- **Worktrees other tools create**, in the store or out of it. The precedent lanes stay as they are.
- **Retention of the notes archive.** Nothing reclaims archived notes; a later tidy of `~/.abcd.noindex/notes/` is its own decision.
- **Any change to the isolation rule itself.** The checkout stays the unit of isolation; this intent changes where a checkout lands.

## Mechanism

We expect a machine-scoped, root-SHA-keyed store with a list verb and a reclaim verb to end the surprise because the surprise is location, not existence: the same isolation, in a directory abcd owns, found by asking abcd. It fails if agents keep creating worktrees by hand outside the store, and that failure is visible rather than silent — the list verb marks every such worktree as outside, and the status board counts them.

## Scope Conditions

- The verbs act on worktrees created through abcd or by an agent following its conventions. A worktree the user placed by hand is theirs: listed as outside, never moved, never reclaimed. <!-- cond: cond-2609301811538239 -->
- The store is under the caller's own home and needs no authority the caller lacks. Where the home is unwritable, or a level of the chain is a symlink, `add` refuses and creates nothing. <!-- cond: cond-2609301811538333 -->
- The lane is keyed on the root commit, so a re-founded repository gets a new lane, as the history store gives it a new entry. <!-- cond: cond-2609301811536785 -->
- "Merged" is judged against the repository's default branch, by the forge's recorded pull-request state or by patch-equivalence, so a squash or rebase merge counts; with no forge reachable, patch-equivalence alone decides, and a branch rewritten after its merge so that its content no longer matches stays unmerged. <!-- cond: cond-2609301811539227 -->
- One machine. The store is never synced, and its paths never enter a committed file (the privacy-hygiene rule already refuses `/Users/<name>/`). <!-- cond: cond-2609301811533865 -->

## Acceptance Criteria

- **Given** a checkout of a repository with commits, **when** `abcd ahoy worktree add <name>` runs, **then** a worktree exists at `~/.abcd.noindex/worktrees/<root-sha>/<name>/` on the named branch, its path is printed, and the checkout's parent directory holds nothing it did not hold before.
- **Given** the store does not yet exist, **when** `add` runs, **then** each level is created individually and verified as a real directory, and a symlink at any level refuses the whole creation with nothing written.
- **Given** a store level owned by another account or writable by group or others, **when** `add` runs, **then** it refuses and writes nothing.
- **Given** a name `../x`, `-x`, or a name already in the lane, or a repository with no commits, **when** `add` runs, **then** it refuses naming the reason and writes nothing; **given** `--branch` naming an existing branch, the worktree is on that branch and no new branch is made.
- **Given** the build loop creates a lane, **when** it does, **then** it goes through the same store primitive `add` uses.
- **Given** a lane holding two worktrees and a third worktree of the same repository beside the checkout, **when** bare `abcd ahoy worktree` runs, **then** three rows render, the third marked as outside the store, each carrying its branch, clean-or-dirty state and merged-or-unmerged state.
- **Given** lanes for two registered repositories and one unregistered root commit, **when** `abcd ahoy worktree list --all` runs, **then** the registered lanes render under their `index.json` names and the unregistered one under its SHA, and a registry `path` label that no longer exists changes nothing in the rows; a lane whose checkout has moved renders as "checkout not found" with the repair hint, and a directory named by an abbreviated root commit renders as "not a lane", naming this repository's full-key lane when the abbreviation matches it and saying its worktrees are retired by hand.
- **Given** a worktree whose branch is merged into the default branch and whose tree is clean, **when** `prune` runs, **then** the directory is gone, git no longer lists it, the branch still exists, and the removal is reported by name.
- **Given** a clean worktree whose branch was squash-merged (no ancestry, every change on the default branch by content), **when** `prune` runs with no forge reachable, **then** it is reclaimed.
- **Given** a reclaimable worktree whose `.abcd/.work.local/` holds a handover note and a log, **when** `prune` runs, **then** both files exist, redacted, under `~/.abcd.noindex/notes/<root-sha>/<timestamp>-<name>/` beside a manifest naming the worktree, branch, time and files, the report names that path, and nothing from the tier is deleted; **given** the archive cannot be written, the worktree is kept and named with the reason.
- **Given** an unmerged worktree with no activity for 15 days and no open pull request, **when** `prune` runs, **then** it is listed as a quiet candidate with its dossier (dirty state first) and left in place; **when** `prune --yes <name>` runs, **then** its notes are archived and it is removed, and no other quiet candidate is touched; **given** the named candidate holds uncommitted changes, it is kept and named with that reason, because prune never forces.
- **Given** a store refusal (a symlinked level, another account's directory), **when** `prune` runs, **then** it exits 2 and removes nothing.
- **Given** a worktree with uncommitted changes, one on an unmerged branch, and one outside the store, **when** `prune` runs, **then** all three remain, each named with its reason, and the exit code is 1; **given** the same lane plus one merged, clean worktree, the exit code is 0.
- **Given** a directory inside the lane that git does not list as a worktree of this repository, or whose common directory is another checkout's, **when** `prune` runs, **then** it is left untouched and reported — the store never deletes what fails its proof of belonging.
- **Given** a merged, clean worktree that git holds locked, one that is prune's own working directory, and one the build loop's state file still holds, **when** `prune` runs, **then** all three remain, each named with its reason, and no removal is forced.
- **Given** a lane holding a reclaimable worktree, **when** `prune --dry-run` runs, **then** nothing is removed and the rows are the ones a real run would report; `--json` emits the same rows.
- **Given** a repository whose lane holds worktrees, **when** the `/abcd` board renders in that checkout, **then** one line names the count held and the count reclaimable; **given** an empty lane, the line is absent; the line never appears in the site's rendered status block, and rendering it starts no per-lane status read.
- **Given** the checkout has been moved to another path, **when** `abcd ahoy worktree` runs from the moved checkout, **then** the same lane renders, because the key is the root commit and not the path.
- **Given** `abcd ahoy worktree add <name>` has run, **when** the new worktree's directory is inspected, **then** its mode is `0o700`.
- **Given** `AGENTS.md`'s concurrent-sessions convention, **when** it is read, **then** it names the verb as the way a session gets its own checkout, and names no sibling-directory form.
- **Given** the plugin page for `ahoy`, **when** it is read, **then** it carries the host-run step: an agent needing an isolated checkout runs `abcd ahoy worktree add <name>` and works in the printed path, never a host default location.

## Prior Art

- [adr-2609091248200336](../../decisions/adrs/2609091248200336-a-tool-never-creates-directories-in-user-owned-project-space.md) — the rule: a tool never creates directories in user-owned project space; agent and session scratch is machine-scoped. This intent is its enforcement on the write side.
- [adr-2609091248201071](../../decisions/adrs/2609091248201071-the-transcript-corpus-is-a-sibling-store-that-creates-itself.md) and `internal/core/history/location.go` — the sibling store this copies: user-level, root-SHA-keyed as a directory, self-creating through one seam, never through a symlink. The reasoning for the key is written there and is not repeated here.
- [adr-35](../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md) — the voyage store moved to `~/.abcd.noindex/` on the same ground, and embark's destination gate "never overwrites a directory abcd did not produce": the same stance, at the destination, that prune takes in the lane.
- [iss-2609020721142452](../../../work/issues/resolved/iss-2609020721142452-worktrees-for-parallel-lanes-are-created-one-directory-above.md) — the captured defect and its three options; this intent is its option 2, with the list and prune verbs and the merged-branch rule the issue asked for.
- [itd-118](../drafts/itd-118-merged-work-leaves-no-residue-abcd-managed-repos-delete-a-pr.md) — the post-merge tidy, which names the worktree among the residue a merge leaves; this intent supplies the store it tidies and the verb it calls.
- `AGENTS.md` § Concurrent sessions — the isolation rule this intent gives a location to.
- `git worktree list --porcelain` and `git worktree prune` — the primitives; the verbs wrap them and add the store, the state columns and the refusal reasons, and no registry of their own.
- [`durable-state-lives-where-the-platform-says-it-survives`](../../principles/durable-state-lives-where-the-platform-says-it-survives.md) — adjacent: that rule chooses a home by what survives the platform's lifecycle; this store's home is chosen by whose space it is. The store satisfies both, and neither is the other.

## Decisions

Ruled at this draft's planning interview, 2026-09-30 (the product thinker and technical facilitator, one question at a time):

1. **Decomposition: file as is.** One intent; the store primitive and an invariant citing adr-2609091248200336 go to the brief's internals; no new ADR or principle.
2. **Merged means** the forge's recorded pull-request state or patch-equivalence, working without a forge (the ruling itd-148's 2026-08-26 interview made, carried here).
3. **Quiet worktrees** (14 days, repo-configurable, no open pull request) are surfaced with a dossier and removed only when named (itd-148's 2026-08-26 ruling, carried here).
4. **The local tier is moved, never deleted**, into a notes archive that is a sibling of the transcript store, recorded and timestamped.
5. **The verb is a sub-verb of `ahoy`**, which owns both stores.
6. **The plugin page carries `add` as a host-run step.**
7. **The worktree directory is `0o700`.**
8. **Exit codes:** 0 reclaimed something, 1 reclaimed nothing, 2 refusal or fault.
9. **A short-key directory's row names the full-key lane.**
10. **The store's root is spelled under the renamed home** (2026-10-04, recorded with spc-2610031309233367's step 4, no interview): `~/.abcd.noindex/worktrees/<root-sha>/<name>/` and `~/.abcd.noindex/notes/<root-sha>/`, the location [adr-2610031751065746](../../decisions/adrs/2610031751065746-the-worktree-store-lives-under-the-renamed-home-abcd-noindex.md) sets when it supersedes adr-2609091248200336's `~/.abcd/` spelling. The trust rule and everything else in this record are unchanged.

## Open Questions

_None open._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: working copies keep piling up with nothing to list or clear them; wrong if, after it ships, copies still accumulate outside the store.
