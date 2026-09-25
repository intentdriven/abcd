---
id: itd-2609091014076309
slug: session-and-agent-worktrees-live-in-a-machine-scoped-store-t
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-118]
severity: major
impact: additive
origin: researcher-authored
production_mode: hand-written
related_adrs: [adr-2609091248200336, adr-2609091248201071]
---

# Session and agent worktrees live in a machine-scoped store that abcd lists and reclaims, never beside the user's own projects

Typed links: `related_adrs` [adr-2609091248200336](../../decisions/adrs/2609091248200336-a-tool-never-creates-directories-in-user-owned-project-space.md) (the rule this store enacts), [adr-2609091248201071](../../decisions/adrs/2609091248201071-the-transcript-corpus-is-a-sibling-store-that-creates-itself.md) (the root-SHA-keyed sibling store whose shape this copies); `builds_on` [itd-118](../drafts/itd-118-merged-work-leaves-no-residue-abcd-managed-repos-delete-a-pr.md) (its worktree clause: the worktree a merge leaves behind lives in the store and is reclaimed by the verb this intent adds, so the post-merge tidy calls that verb rather than growing a second reclaim).

## Press Release

> **abcd keeps every session and agent worktree in one place on your machine, and can tell you what is there.** `abcd worktree add <name>` creates a worktree at `~/.abcd/worktrees/<root-sha>/<name>/`, keyed on the repository's root commit the way the history and transcript stores already are, and writes nothing beside your checkout. Bare `abcd worktree` lists what the store holds for this repository — name, branch, path, whether the tree is clean, whether its branch has merged — and names any worktree of the repository that sits outside the store; `abcd worktree list --all` walks every lane on the machine. `abcd worktree prune` reclaims a worktree whose branch has merged and whose tree is clean, reports each removal by name, and leaves everything else where it is, with the reason. The `/abcd` status board carries one line: how many worktrees the store holds for this repository and how many are reclaimable. Your project directory holds what you put there.
>
> "I came back from a run to find twenty-two new folders beside my projects, none of which I had made, and no way to tell which were still in use," said Maya, autonomous-development practitioner. "The isolation was right — parallel agents need separate checkouts. The location was not. Now the checkouts live where abcd keeps the rest of its machine state, one command lists them, one command clears the ones whose work has merged, and my folder is mine again."

## Why This Matters

Parallel agents need separate checkouts: `AGENTS.md`'s concurrent-sessions convention makes the checkout the unit of isolation, and the record's gates read the whole tree, so two sessions in one checkout fail each other's gates in both directions. Nothing says where a checkout goes, so each agent picks, and what an agent picks is git's default, a sibling directory. On 2026-09-01 twenty-one spent worktrees (1.4 GB) were removed by hand from the maintainer's project directory ([iss-2609020721142452](../../../work/issues/resolved/iss-2609020721142452-worktrees-for-parallel-lanes-are-created-one-directory-above.md)); on 2026-09-06 one session created twenty-two more, beside four unrelated projects; `git worktree list` on that checkout names twenty-seven. The maintainer's objection, verbatim: "I don't want a user to be surprised that a folder is all of a sudden full of stuff."

That objection is the rule [adr-2609091248200336](../../decisions/adrs/2609091248200336-a-tool-never-creates-directories-in-user-owned-project-space.md) records: a tool never creates directories in space that belongs to the user, and agent and session scratch is machine-scoped. This intent is the capability that makes the rule followable — an agent that has somewhere to put a worktree, and a user who can see and reclaim what was put there.

The shape is not new. `~/.abcd/history/<root-sha>/`, `~/.abcd/transcripts/<root-sha>/` and `~/.abcd/voyage/<root-sha>/` are each keyed on the repository's root-commit SHA, and `internal/core/history/location.go` states why: a checkout moves, is renamed, and is cloned twice on one machine, while its root commit changes under none of that. The key is proving itself on this machine as this is written — `~/.abcd/history/index.json` registers abcd under a checkout path that no longer exists, and every store keyed on the root commit is unaffected, because the path is a mutable label and the SHA is the key. `index.json` is the sole user-scope registry of every repository abcd knows on the machine, so a cross-repository listing is a walk that already exists. There is a de facto precedent on disk as well: `~/.abcd/worktrees/488a0aa9/phase-9` and `.../park-gates`, created by other tooling under abcd's own root commit. That is corroboration that the location is the natural one, not authority for the shape — the precedent abbreviates the key, and this store uses the full SHA the sibling stores use.

Two things separate the store from the same pile somewhere less visible, and the intent owns both.

**Discoverability.** A worktree under `~/.abcd/` is invisible to `ls` where the user works, which is the point, and so it has to be visible somewhere else. Git already knows every worktree of a checkout wherever it sits (`git worktree list --porcelain`), so the store adds no registry of its own. It adds a verb that reads git's answer and says which entries are in the store, which are outside it, and what state each is in; and a line on the `/abcd` status board, so a user who never types the verb still learns the count.

**Lifecycle.** A worktree whose branch has merged is garbage, and nothing today notices. Reclaiming is `abcd worktree prune`: it removes a worktree only when its branch is merged into the default branch and its tree is clean, and it names what it declined and why. It runs when the user runs it; the status board's reclaimable count is what prompts them. Reclaiming automatically on merge is [itd-118](../drafts/itd-118-merged-work-leaves-no-residue-abcd-managed-repos-delete-a-pr.md)'s post-merge tidy; when that ships it calls this verb rather than inventing a second reclaim.

## What's In Scope

- **`abcd worktree add <name> [--branch <branch>]`** creates a worktree at `~/.abcd/worktrees/<root-sha>/<name>/` for the repository the caller stands in, on a new branch named after the worktree unless `--branch` names an existing one, and prints the path. The store is created on first use the way the transcript store is: each level made individually and re-verified as a real directory, never through a symlink, never with a single recursive make. A name already in the lane refuses. Nothing is written beside the checkout.
- **Bare `abcd worktree` and `abcd worktree list`** render the lane for this repository: name, branch, path (home-redacted on a machine stream), clean or dirty, merged or unmerged. Every worktree git reports for the checkout is a row; one outside the store is marked as outside and never touched. `--all` walks every lane under `~/.abcd/worktrees/`, labelling each through `~/.abcd/history/index.json` where the root commit is registered and by SHA where it is not; the worktree's own `.git` file, not the registry's path label, says which checkout a lane belongs to. `--json` emits the same rows.
- **`abcd worktree prune [--dry-run]`** removes each worktree in the lane whose branch is merged into the repository's default branch and whose tree is clean, runs git's own metadata prune for it, and reports each removal by name. A dirty tree, an unmerged branch, a worktree outside the store, and a directory in the lane that git does not recognise as a worktree of this repository are each left in place and named with the reason. The branch is left standing.
- **One line on the `/abcd` status board**: the count of worktrees in the lane and the count reclaimable, absent when the lane is empty.
- **The concurrent-sessions convention in `AGENTS.md` points at the verb**, and the plugin page carries the surface, so an agent asked for isolation has a place to put it and no reason to pick one.

## What's Out of Scope

- **Deleting branches, tracking refs or remote branches.** That is the rest of itd-118's post-merge tidy; prune reclaims the directory and git's record of it, and nothing else.
- **Moving a worktree that already sits beside a checkout.** The store never moves what it did not create. A sibling worktree is listed as outside and left alone; retiring it is the user's own `git worktree remove`, and the twenty-seven on the maintainer's machine are a hand cleanup, not a migration.
- **Worktrees other tools create**, in the store or out of it. The precedent lanes stay as they are.
- **Any change to the isolation rule itself.** The checkout stays the unit of isolation; this intent changes where a checkout lands.

## Mechanism

We expect a machine-scoped, root-SHA-keyed store with a list verb and a reclaim verb to end the surprise because the surprise is location, not existence: the same isolation, in a directory abcd owns, found by asking abcd. It fails if agents keep creating worktrees by hand outside the store, and that failure is visible rather than silent — the list verb marks every such worktree as outside, and the status board counts them.

## Scope Conditions

- The verbs act on worktrees created through abcd or by an agent following its conventions. A worktree the user placed by hand is theirs: listed as outside, never moved, never reclaimed.
- The store is under the caller's own home and needs no authority the caller lacks. Where the home is unwritable, or a level of the chain is a symlink, `add` refuses and creates nothing.
- The lane is keyed on the root commit, so a re-founded repository gets a new lane, as the history store gives it a new entry.
- "Merged" is judged against the repository's default branch. The repository allows squash and rebase merges, which leave no ancestry, so the spec decides whether a patch-identical branch or a branch whose upstream is gone counts; until it does, only an ancestry merge reclaims, which errs toward leaving a worktree in place.
- One machine. The store is never synced, and its paths never enter a committed file (the privacy-hygiene rule already refuses `/Users/<name>/`).

## Acceptance Criteria

- **Given** a checkout of a repository with commits, **when** `abcd worktree add <name>` runs, **then** a worktree exists at `~/.abcd/worktrees/<root-sha>/<name>/` on the named branch, its path is printed, and the checkout's parent directory holds nothing it did not hold before.
- **Given** the store does not yet exist, **when** `add` runs, **then** each level is created individually and verified as a real directory, and a symlink at any level refuses the whole creation with nothing written.
- **Given** a lane holding two worktrees and a third worktree of the same repository beside the checkout, **when** bare `abcd worktree` runs, **then** three rows render, the third marked as outside the store, each carrying its branch, clean-or-dirty state and merged-or-unmerged state.
- **Given** lanes for two registered repositories and one unregistered root commit, **when** `abcd worktree list --all` runs, **then** the registered lanes render under their `index.json` names and the unregistered one under its SHA, and a registry `path` label that no longer exists changes nothing in the rows.
- **Given** a worktree whose branch is merged into the default branch and whose tree is clean, **when** `prune` runs, **then** the directory is gone, git no longer lists it, the branch still exists, and the removal is reported by name.
- **Given** a worktree with uncommitted changes, one on an unmerged branch, and one outside the store, **when** `prune` runs, **then** all three remain, each named with its reason, and the exit code says nothing was reclaimed unless something was.
- **Given** a directory inside the lane that git does not recognise as a worktree of this repository, **when** `prune` runs, **then** it is left untouched and reported — the store never deletes what it cannot prove it created.
- **Given** a repository whose lane holds worktrees, **when** the `/abcd` board renders in that checkout, **then** one line names the count held and the count reclaimable; **given** an empty lane, the line is absent.
- **Given** the checkout has been moved to another path, **when** `abcd worktree` runs from the moved checkout, **then** the same lane renders, because the key is the root commit and not the path.
- **Given** `AGENTS.md`'s concurrent-sessions convention, **when** it is read, **then** it names the verb as the way a session gets its own checkout, and names no sibling-directory form.

## Prior Art

- [adr-2609091248200336](../../decisions/adrs/2609091248200336-a-tool-never-creates-directories-in-user-owned-project-space.md) — the rule: a tool never creates directories in user-owned project space; agent and session scratch is machine-scoped. This intent is its enforcement on the write side.
- [adr-2609091248201071](../../decisions/adrs/2609091248201071-the-transcript-corpus-is-a-sibling-store-that-creates-itself.md) and `internal/core/history/location.go` — the sibling store this copies: user-level, root-SHA-keyed as a directory, self-creating through one seam, never through a symlink. The reasoning for the key is written there and is not repeated here.
- [adr-35](../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md) — the voyage store moved to `~/.abcd/` on the same ground, and embark's destination gate "never overwrites a directory abcd did not produce": the same stance, at the destination, that prune takes in the lane.
- [iss-2609020721142452](../../../work/issues/resolved/iss-2609020721142452-worktrees-for-parallel-lanes-are-created-one-directory-above.md) — the captured defect and its three options; this intent is its option 2, with the list and prune verbs and the merged-branch rule the issue asked for.
- [itd-118](../drafts/itd-118-merged-work-leaves-no-residue-abcd-managed-repos-delete-a-pr.md) — the post-merge tidy, which names the worktree among the residue a merge leaves; this intent supplies the store it tidies and the verb it calls.
- `AGENTS.md` § Concurrent sessions — the isolation rule this intent gives a location to.
- `git worktree list --porcelain` and `git worktree prune` — the primitives; the verbs wrap them and add the store, the state columns and the refusal reasons, and no registry of their own.
- [`durable-state-lives-where-the-platform-says-it-survives`](../../principles/durable-state-lives-where-the-platform-says-it-survives.md) — adjacent: that rule chooses a home by what survives the platform's lifecycle; this store's home is chosen by whose space it is. The store satisfies both, and neither is the other.

## Open Questions

- Whether the verb is `abcd worktree` or a sub-verb of `ahoy`, which already owns the rest of the machine-scope layout. A top-level verb reads better at the prompt; a sub-verb keeps one owner for `~/.abcd/`.
- Whether the plugin surface should carry `add` as a host-run step, so a harness's own checkout-isolation feature lands in the store rather than wherever the harness defaults to. The prose stays host-agnostic either way.
- Whether the lane is `0o700` like the transcript store. A worktree holds the same bytes as the checkout, so the checkout's own mode is the nearer precedent.
- Whether prune treats a squash-merged branch as merged (see Scope Conditions); the conservative default is stated there and the spec may widen it.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
