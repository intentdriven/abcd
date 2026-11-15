---
id: itd-2609091014076309
slug: session-and-agent-worktrees-live-in-a
spec_id: spc-2609301811532881
kind: standalone
suggested_kind: null
reclassification_history: []
refines: [itd-2609201916151817]
related_intents: [itd-118, itd-148, itd-2610161621364075]
severity: major
impact: additive
origin: researcher-authored
production_mode: hand-written
related_adrs: [adr-2609091248200336, adr-2609091248201071]
related_issues: [iss-2610230014564359, iss-2610231003000222, iss-2610231011465325, iss-2610231125096250]
---

# Session and agent worktrees live in a machine-scoped store that abcd lists and reclaims, never beside the user's own projects

Typed links: `related_adrs` [adr-2609091248200336](../../decisions/adrs/2609091248200336-a-tool-never-creates-directories-in-user.md) (the rule this store enacts), [adr-2609091248201071](../../decisions/adrs/2609091248201071-the-transcript-corpus-is-a-sibling-store.md) (the root-SHA-keyed sibling store whose shape this copies); `refines` [itd-2609201916151817](../planned/itd-2609201916151817-one-verb-takes-a-single-intent-from.md) (the build loop, whose lane primitive becomes this store's one primitive: its spec's piece 6 takes the store's verb once this ships); `related_intents` [itd-118](../superseded/itd-118-merged-work-leaves-no-residue-abcd.md) (the post-merge tidy that will call `prune`) and [itd-148](../planned/itd-148-every-change-starts-in-its-own-worktree.md) (every change in its own worktree, which uses this store's verbs and is `blocked_by` this draft).

## Press Release

> **abcd keeps every session and agent worktree in one place on your machine, and can tell you what is there.** `abcd ahoy worktree add <name>` creates a worktree at `~/.abcd.noindex/worktrees/<root-sha>/<name>/`, keyed on the repository's root commit the way the history and transcript stores already are, and writes nothing beside your checkout. Bare `abcd ahoy worktree` lists what the store holds for this repository — name, branch, path, whether the tree is clean, whether its branch has merged — and names any worktree of the repository that sits outside the store; `abcd ahoy worktree list --all` walks every lane on the machine. `abcd ahoy worktree prune` reclaims a worktree whose branch has merged and whose tree is clean, moves the worktree's private notes into a dated archive rather than deleting them, reports each removal by name, and leaves everything else where it is, with the reason; a worktree that has gone quiet without merging is pointed out with a summary and goes only when you name it. The `/abcd` status board carries one line: how many worktrees the store holds for this repository and how many are reclaimable. Your project directory holds what you put there.
>
> "I came back from a run to find twenty-two new folders beside my projects, none of which I had made, and no way to tell which were still in use," said Maya, autonomous-development practitioner. "The isolation was right — parallel agents need separate checkouts. The location was not. Now the checkouts live where abcd keeps the rest of its machine state, one command lists them, one command clears the ones whose work has merged, and my folder is mine again."

## Why This Matters

Parallel agents need separate checkouts: `AGENTS.md`'s concurrent-sessions convention makes the checkout the unit of isolation, and the record's gates read the whole tree, so two sessions in one checkout fail each other's gates in both directions. Nothing says where a checkout goes, so each agent picks, and what an agent picks is git's default, a sibling directory. On 2026-09-01 twenty-one spent worktrees (1.4 GB) were removed by hand from the product thinker's project directory ([iss-2609020721142452](../../../work/issues/resolved/iss-2609020721142452-worktrees-for-parallel-lanes-are-created.md)); on 2026-09-06 one session created twenty-two more, beside four unrelated projects, and `git worktree list` on that checkout named twenty-seven. The product thinker's objection, verbatim: "I don't want a user to be surprised that a folder is all of a sudden full of stuff."

That objection is the rule [adr-2609091248200336](../../decisions/adrs/2609091248200336-a-tool-never-creates-directories-in-user.md) records: a tool never creates directories in space that belongs to the user, and agent and session scratch is machine-scoped. This intent is the capability that makes the rule followable — an agent that has somewhere to put a worktree, and a user who can see and reclaim what was put there.

The shape is not new. `~/.abcd.noindex/history/<root-sha>/`, `~/.abcd.noindex/transcripts/<root-sha>/` and `~/.abcd.noindex/voyage/<root-sha>/` are each keyed on the repository's root-commit SHA, and `internal/core/history/location.go` states why: a checkout moves, is renamed, and is cloned twice on one machine, while its root commit changes under none of that. `index.json` is the sole user-scope registry of every repository abcd knows on the machine, so a cross-repository listing is a walk that already exists. There is a de facto precedent on disk as well: `~/.abcd.noindex/worktrees/488a0aa9/phase-9` and `.../park-gates`, laid by hand by an autonomous run under abcd's own root commit before the verb exists ([dcn-2609260000000007](../../../work/decisions/dcn-2609260000000007-the-build-loop-s-worktree-store-is-keyed.md)). That is corroboration that the location is the natural one, not authority for the shape — the precedent abbreviates the key, and this store uses the full SHA the sibling stores use.

Two things separate the store from the same pile somewhere less visible, and the intent owns both.

**Discoverability.** A worktree under `~/.abcd.noindex/` is invisible to `ls` where the user works, which is the point, and so it has to be visible somewhere else. Git already knows every worktree of a checkout wherever it sits (`git worktree list --porcelain`), so the store adds no registry of its own. What stands in for one is a proof git can give: a worktree belongs to the store when git lists it as a worktree of this repository, its real path lies inside this repository's lane, and its own common directory is this checkout's. It adds a verb that reads git's answer and says which entries are in the store, which are outside it, and what state each is in; and a line on the `/abcd` status board, so a user who never types the verb still learns the count.

**Lifecycle.** A worktree whose branch has merged is garbage, and nothing today notices. Reclaiming is `abcd ahoy worktree prune`: it removes a worktree only when its branch is merged into the default branch and its tree is clean, moves the worktree's local tier into the notes archive first, and names what it declined and why. It runs when the user runs it; the status board's reclaimable count is what prompts them. Reclaiming automatically on merge is [itd-118](../superseded/itd-118-merged-work-leaves-no-residue-abcd.md)'s post-merge tidy; when that ships it calls this verb rather than inventing a second reclaim.

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

- [adr-2609091248200336](../../decisions/adrs/2609091248200336-a-tool-never-creates-directories-in-user.md) — the rule: a tool never creates directories in user-owned project space; agent and session scratch is machine-scoped. This intent is its enforcement on the write side.
- [adr-2609091248201071](../../decisions/adrs/2609091248201071-the-transcript-corpus-is-a-sibling-store.md) and `internal/core/history/location.go` — the sibling store this copies: user-level, root-SHA-keyed as a directory, self-creating through one seam, never through a symlink. The reasoning for the key is written there and is not repeated here.
- [adr-35](../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md) — the voyage store moved to `~/.abcd.noindex/` on the same ground, and embark's destination gate "never overwrites a directory abcd did not produce": the same stance, at the destination, that prune takes in the lane.
- [iss-2609020721142452](../../../work/issues/resolved/iss-2609020721142452-worktrees-for-parallel-lanes-are-created.md) — the captured defect and its three options; this intent is its option 2, with the list and prune verbs and the merged-branch rule the issue asked for.
- [itd-118](../superseded/itd-118-merged-work-leaves-no-residue-abcd.md) — the post-merge tidy, which names the worktree among the residue a merge leaves; this intent supplies the store it tidies and the verb it calls.
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
10. **The store's root is spelled under the renamed home** (2026-10-04, recorded with spc-2610031309233367's step 4, no interview): `~/.abcd.noindex/worktrees/<root-sha>/<name>/` and `~/.abcd.noindex/notes/<root-sha>/`, the location [adr-2610031751065746](../../decisions/adrs/2610031751065746-the-worktree-store-lives-under-the.md) sets when it supersedes adr-2609091248200336's `~/.abcd/` spelling. The trust rule and everything else in this record are unchanged.

## SOTA

Surveyed on 2026-10-23. The host-side findings are shared with the
[security and isolation SOTA note](../../research/notes/2026-10-23-security-isolation-sota.md)
(its itd-148 section) and the [autonomous loops SOTA note](../../research/notes/2026-10-23-autonomous-loops-sota.md).
The store-manager comparison is new here. The internal lineage stays under
`## Prior Art`.

- **The first host now makes worktrees itself, inside the checkout.** A
  sub-agent's `isolation: worktree` gives it a temporary worktree, held under
  `git worktree lock` while it runs, at `.claude/worktrees/` under the
  repository by default. Headless runs leave their worktrees and locks behind
  for a later sweep. The host refuses a worktree whose git identity resolves
  into the main checkout (Claude Code worktrees documentation, cited in both
  notes). The host's default lands exactly where
  adr-2609091248200336 forbids, which makes "never a host's default worktree
  location" a live rule rather than a hypothetical.
- **The host offers the seam to relocate.** A `WorktreeCreate` hook "replaces
  the default git behavior", so a hook calling `abcd ahoy worktree add` can put
  the host's own isolation worktrees in the store (Claude Code hooks
  documentation, cited in the autonomous loops note). That turns the AGENTS.md
  convention into a mechanism on that host. The seam is a pair of hooks, not
  one: the host's periodic sweep keeps every worktree without its own marker,
  a hook-made one included, and removes a hook-made worktree only through a
  paired `WorktreeRemove` hook (Claude Code worktrees documentation).
- **Third-party managers split on where worktrees live.** gwq keeps a central
  base directory, `~/worktrees` by default, laid out by remote URL
  (`{{.Host}}/{{.Owner}}/{{.Repository}}/{{.Branch}}`). It lists every
  repository's worktrees with `-g` and deletes an unmerged branch only with
  `--force-delete-branch` ([gwq](https://github.com/d-kuro/gwq)). Worktrunk
  computes paths from a template, and its quick start shows the sibling layout
  `~/repo.feature-auth`, the beside-the-project placement the ADR rules out
  ([Worktrunk README](https://cdn.jsdelivr.net/gh/max-sixty/worktrunk@main/README.md)).
  Worktrunk's `wt merge` removes worktree and branch together after a local
  merge. The record keeps branch deletion out of scope (itd-118).
- **A URL key and a root-commit key fail differently.** A URL key (gwq)
  breaks on a rename, a fork and a repository with no remote. The root-commit
  key survives all three, at the cost of putting every clone of one project on
  the machine in one lane. The prune proof already ties a worktree to its
  checkout. A name collision between two clones' `add` is the remaining case.
- **Merged-and-clean is the field's prune rule too.** gwq's force flag for
  unmerged branches and the host's lock-while-running match the record's
  refusals: dirty, unmerged, locked, outside the store.

**Path 2**: native, with git's own worktree and prune as the primitives and
the root-commit key the sibling stores use. The host's `WorktreeCreate` hook
is named as the seam that routes a host-made worktree into the store. No
dependency. The record was ruled at its 2026-09-30 planning interview; this pass raises
two questions below.

Questions this pass raised for the fit-challenge before the build. They are kept here, not under Open Questions, so they do not hold a build the record already rules ready:

- Two clones of one project on one machine share a lane, because they share a root commit. What does `add` do when the other clone already holds a worktree of that name: refuse naming the other checkout, or qualify the name?
- Should abcd ship the host's `WorktreeCreate` hook, so a host-made isolation worktree lands in the store, or leave the AGENTS.md convention as the only route on that host? If it ships it, does it ship the paired `WorktreeRemove` hook too, since the host's sweep never removes a hook-made worktree and a sub-agent's throwaway lane would otherwise sit in the store until prune names it?

## Open Questions

_None open._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-01d317e17525 -->
Fidelity review — receipt rcp-01d317e17525 (verifier abcd:intent-auditor claude-opus-5-5).

Provenance: abcd:intent-auditor@claude-opus-5-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:c4c21d8e0e38a05344c95e8f83e4cea2c02cb4ff3434487e9e1a202f463e418f
Input attestations: commit:1009cdd4a3127b7adcc75d4bdeb8089d367a73b8 (step 1, on origin/main)@-; commit:5ce7c0e882e8ce7019eca085d6cb1c9fe629f607 (steps 2 to 6, on origin/main)@-; diff:origin/main...b70b459b798f0e7d2b5f80f12e69c9813889b5f9 (step 7 and the close, branch worktree-surfaces-1115)@sha256:6028324478fae2d4a1623c193016f24ba1abd54433acbfe1ccc7c4f300745721; commit:01e9fbf41e4ef6061ef6db4977432af072b53ed3 (merge of PR #1577, the registry-label fix for ac-7, on origin/main)@-; tree:4276675814bf85427deedc9b4f5dff01e501ee52 (HEAD of the re-audit worktree, origin/main merged in)@-;

Acceptance rollup: MET 18 · MET_WITH_CONCERNS 3 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: Add makes the worktree at the lane path on a new branch and the door prints that path alone; the CLI test checks the path, the branch, and that the checkout's parent gains no entry.
  evidence: internal/core/ahoy/worktree/worktree.go:352 — "args = []string{"worktree", "add", "-b", branch, "--", path, opts.Base}"
  evidence: internal/surface/cli/ahoy_worktree.go:174 — "return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) { fmt.Fprintln(w, out.Path) })"
  evidence: internal/surface/cli/ahoy_worktree_test.go:679 — "func TestAhoyWorktreeAddPrintsThePath(t *testing.T) {"
  evidence: internal/surface/cli/ahoy_worktree_test.go:705 — "t.Fatalf("nothing is made beside the checkout: %d -> %d entries (%v)""
- ac-2 — MET: levels() makes each level one at a time with EnsureRealDir, then Lstat-checks it before going deeper. Add reads the chain (present) before Ensure, so a symlinked level refuses before git runs. The test plants a symlink at each of the three levels.
  evidence: internal/core/ahoy/worktree/worktree.go:163 — "if err := fsutil.EnsureRealDir(dir, dirPerm); err != nil {"
  evidence: internal/core/ahoy/worktree/worktree.go:175 — "is not real directories: %s is a symlink or a file"
  evidence: internal/core/ahoy/worktree/worktree.go:324 — "if _, err := lane.present(); err != nil {"
  evidence: internal/core/ahoy/worktree/worktree_test.go:125 — "func TestEnsureRefusesASymlinkedLevel(t *testing.T) {"
  evidence: internal/core/ahoy/worktree/worktree_test.go:99 — "func TestEnsureMakesEachLevelTheCallersAlone(t *testing.T) {"
- ac-3 — MET: CallersAlone holds every level to a single owner and no group or other write bit. Tests cover modes 0770, 0722 and 0777 and a substituted foreign owner at each level, and nothing is made beneath a refused level.
  evidence: internal/core/ahoy/worktree/worktree.go:179 — "case errors.Is(err, fsutil.ErrDeclarationWritable):"
  evidence: internal/core/ahoy/worktree/worktree.go:183 — "is owned by another account, or its owner could not be read"
  evidence: internal/core/ahoy/worktree/worktree_test.go:163 — "func TestEnsureRefusesALevelAnyoneElseCanWrite(t *testing.T) {"
  evidence: internal/core/ahoy/worktree/worktree_test.go:202 — "func TestEnsureRefusesALevelAnotherAccountOwns(t *testing.T) {"
- ac-4 — MET: ValidName, the occupied-path check and the root-commit check each refuse with a named reason before any level is made. --branch checks out an existing branch with no new ref. The CLI and core tests cover ../x, -x, a taken name, no commits and --branch.
  evidence: internal/core/ahoy/worktree/worktree.go:194 — "func ValidName(name string) bool {"
  evidence: internal/core/ahoy/worktree/worktree.go:330 — "%s is occupied: git lists a worktree there already"
  evidence: internal/core/ahoy/worktree/worktree.go:101 — "the repository has no root commit to key the worktree store on"
  evidence: internal/surface/cli/ahoy_worktree_test.go:741 — "func TestAhoyWorktreeAddRefusesAndWritesNothing(t *testing.T) {"
  evidence: internal/surface/cli/ahoy_worktree_test.go:720 — "func TestAhoyWorktreeAddChecksOutAnExistingBranch(t *testing.T) {"
  evidence: internal/core/ahoy/worktree/worktree_test.go:424 — "func TestAddRefusesBeforeWritingAnything(t *testing.T) {"
- ac-5 — MET: The loop's lane step calls worktree.LaneForContext, Ensure, Adopt and Add (through the addWorktree seam). The test watches that seam and finds exactly one Add call at the store's path, and none on adoption.
  evidence: internal/core/implement/loop/lane.go:42 — "var addWorktree = worktree.Add"
  evidence: internal/core/implement/loop/lane.go:381 — "adopted, err := worktree.Adopt(c.RepoRoot, lw.lane, name, lw.Branch)"
  evidence: internal/core/implement/loop/lane_store_test.go:15 — "func TestTheLaneGoesThroughTheStorePackage(t *testing.T) {"
- ac-6 — MET: scan() makes one row for each worktree git lists apart from the main checkout. A row is marked store only when it is a direct child of the real lane, and outside otherwise. The test checks three rows, each with branch, clean/dirty and merged/unmerged.
  evidence: internal/core/ahoy/worktree/list.go:240 — "case filepath.Dir(real) == realLane:"
  evidence: internal/core/ahoy/worktree/list_test.go:72 — "func TestListRendersTheLaneAndASibling(t *testing.T) {"
  evidence: internal/core/ahoy/worktree/list_test.go:98 — ""sibling": {Branch: "sib", Where: "outside", State: "clean", Merged: "unmerged"},"
- ac-7 — MET: The door now reads the registry: registryLabels() calls ahoy.RegistryLabels, which maps each root commit in ~/.abcd.noindex/history/index.json (read under the machine home, through the scrubbing loader) to its name, and passes the map to ListAll, which heads a lane with its registry name and leaves an unregistered one under its SHA. The CLI test registers one of two lanes with a path that does not exist and checks the registered lane's name and the other's SHA in both forms. The core test covers two registered lanes and one unregistered, a stale registry path that leaves the rows byte-identical, a moved checkout as 'checkout not found' with the repair hint, and an abbreviated directory as 'not a lane' naming the full-key lane and git worktree remove. The brief and the plugin page now describe the labelling.
  evidence: internal/surface/cli/ahoy_worktree.go:82 — "labels, note := registryLabels()"
  evidence: internal/surface/cli/ahoy_worktree.go:83 — "got, err := worktree.ListAll(worktree.AllOptions{Labels: labels, Here: hereOrNone(), Forge: worktreeForge})"
  evidence: internal/surface/cli/ahoy_worktree.go:433 — "reg, err := ahoy.RegistryLabels()"
  evidence: internal/core/ahoy/registry_labels.go:36 — "func RegistryLabels() (map[string]RegistryLabel, error) {"
  evidence: internal/core/ahoy/registry_labels.go:42 — "idx, err := loadHistoryIndexUnder(home)"
  evidence: internal/core/ahoy/worktree/list.go:469 — "if l, ok := opts.Labels[n]; ok && l.Name != "" {"
  evidence: internal/surface/cli/ahoy_worktree_test.go:231 — "func TestAhoyWorktreeListAllLabelsRegisteredLanes(t *testing.T) {"
  evidence: internal/surface/cli/ahoy_worktree_test.go:233 — ""name": "alpha-repo", "path": "/no/such/checkout""
  evidence: internal/core/ahoy/worktree/list_test.go:235 — "func TestListAllLabelsEveryLane(t *testing.T) {"
  evidence: internal/core/ahoy/worktree/list_test.go:280 — "if moved.Reason != "checkout not found; run `git worktree repair` from the checkout""
  evidence: internal/core/ahoy/worktree/list_test.go:298 — "t.Fatalf("a stale registry path changed the rows:\n%s\n%s", a, b)"
  evidence: .abcd/development/brief/04-surfaces/01-ahoy.md:351 — "under its repository's name where its root commit is registered in"
  evidence: commands/ahoy.md:1792 — "from any folder. Each lane is shown under its repository's name where its root"
- ac-8 — MET: reclaim runs `git worktree remove` without --force and never deletes a branch. The test checks that the directory is gone and unlisted, that refs/heads/done still verifies, and that the row reads reclaimed.
  evidence: internal/core/ahoy/worktree/prune.go:302 — "_, err := gitutil.Run(repoRoot, "worktree", "remove", "--", path)"
  evidence: internal/core/ahoy/worktree/prune_test.go:98 — "func TestPruneReclaimsAMergedCleanWorktree(t *testing.T) {"
  evidence: internal/core/ahoy/worktree/prune_test.go:110 — "repo.Git("rev-parse", "--verify", "refs/heads/done")"
- ac-9 — MET: judgeTip falls through to rebased or squashed (patch-id) when the forge is silent. The test squash-merges two commits with the forge unreachable and gets a reclaimed row plus the no-forge note.
  evidence: internal/core/ahoy/worktree/merged.go:174 — "if rebased(dir, def, tip) || squashed(dir, def, tip) {"
  evidence: internal/core/ahoy/worktree/prune_test.go:121 — "func TestPruneReclaimsASquashMergedWorktree(t *testing.T) {"
- ac-10 — MET_WITH_CONCERNS: The handover note and log are archived redacted, at 0600, beside a manifest under notes/< root-sha>/< ts>-< name>/, and the row names the entry. A failed archive keeps the worktree. Concern 1: the tier's built binary (abcdrepo.CheckoutBuildRel and its < binary>.< pid> file) is left out of the archive and deleted with the worktree, so 'nothing from the tier is deleted' holds for notes only. Concern 2: the keep-on-failure half is tested only with a file the redaction pass refuses, never with an archive store that cannot be written, through prune.
  evidence: internal/core/ahoy/worktree/prune.go:325 — "entryDir, err = writeNotes(repoRoot, lane, e, notes, opts.Now())"
  evidence: internal/core/ahoy/worktree/prune_test.go:144 — "func TestPruneArchivesTheTierRedacted(t *testing.T) {"
  evidence: internal/core/ahoy/worktree/prune_test.go:230 — "func TestPruneKeepsAWorktreeWhoseNotesCannotBeArchived(t *testing.T) {"
  evidence: internal/core/ahoy/worktree/notes_test.go:19 — "func TestPruneLeavesTheTiersBuiltBinaryOutOfTheNotes(t *testing.T) {"
- ac-11 — MET: decide() lists a quiet, unnamed worktree with its dossier (Dirty is the first field). --yes reclaims only that candidate, through the same archive and removal path. A dirty named candidate is kept as 'dirty: uncommitted changes'. Core and CLI tests run 15 days on.
  evidence: internal/core/ahoy/worktree/prune.go:260 — "case q.quiet && !named:"
  evidence: internal/core/ahoy/worktree/quiet_test.go:68 — "func TestPruneListsAQuietWorktreeWithItsDossier(t *testing.T) {"
  evidence: internal/core/ahoy/worktree/quiet_test.go:144 — "func TestPruneYesRemovesTheNamedQuietCandidateOnly(t *testing.T) {"
  evidence: internal/core/ahoy/worktree/quiet_test.go:193 — "func TestPruneKeepsADirtyNamedQuietCandidate(t *testing.T) {"
  evidence: internal/surface/cli/ahoy_worktree_test.go:593 — "func TestAhoyWorktreePruneListsAndNamesQuietCandidates(t *testing.T) {"
- ac-12 — MET_WITH_CONCERNS: scan() runs lane.present() before it judges anything, and the door maps any refusal to exit 2. Tests cover a group-writable lane and a symlinked store with nothing removed. Concern: 'another account's directory' is never driven through prune; it is proved only by Ensure's substituted-owner test of the shared levels() function. Exit 2 at the door is tested only for a 0777 lane.
  evidence: internal/core/ahoy/worktree/list.go:208 — "inLane, err := lane.present()"
  evidence: internal/surface/cli/ahoy_worktree.go:66 — "return &exitError{Code: 2, Msg: "abcd ahoy worktree " + verb + ": ""
  evidence: internal/core/ahoy/worktree/prune_test.go:259 — "func TestPruneRefusesAStoreItCannotTrust(t *testing.T) {"
  evidence: internal/surface/cli/ahoy_worktree_test.go:513 — "func TestAhoyWorktreePruneRefusesAStoreItCannotTrust(t *testing.T) {"
- ac-13 — MET: Outside, dirty and unmerged rows are kept, each with its reason. The door exits 1 when Reclaimed() is 0 and 0 otherwise. The core test checks the three reasons and the counts; the CLI test checks exit 1 for an unmerged lane and exit 0 once a merged one is reclaimed.
  evidence: internal/core/ahoy/worktree/prune.go:283 — "return "outside the store; never touched""
  evidence: internal/surface/cli/ahoy_worktree.go:244 — "if got.Reclaimed() == 0 {"
  evidence: internal/core/ahoy/worktree/prune_test.go:297 — "func TestPruneKeepsWhatItCannotReclaim(t *testing.T) {"
  evidence: internal/surface/cli/ahoy_worktree_test.go:355 — "func TestAhoyWorktreePruneExitCodes(t *testing.T) {"
- ac-14 — MET: A lane directory git does not list gets reasonNotListed. A listed one whose common directory differs gets reasonOtherCommon. refusedFor keeps any row that has a reason. The test plants both, and both stay untouched and reported.
  evidence: internal/core/ahoy/worktree/list.go:260 — "e := entry{path: lane.Path(n), name: n, where: WhereStore, reason: reasonNotListed}"
  evidence: internal/core/ahoy/worktree/list.go:316 — "case fsutil.RealExistingPath(common) != own:"
  evidence: internal/core/ahoy/worktree/prune_test.go:331 — "func TestPruneLeavesWhatFailsTheProof(t *testing.T) {"
- ac-15 — MET: refusedFor keeps a locked worktree (with git's reason), the cwd's own worktree, and one the build loop holds, before any judgement. Removal never passes --force. The test checks all of them stand with their reasons.
  evidence: internal/core/ahoy/worktree/prune.go:286 — "case e.wt.Locked && e.wt.LockReason != "":"
  evidence: internal/core/ahoy/worktree/prune.go:291 — "return "prune's own working directory""
  evidence: internal/core/ahoy/worktree/prune.go:294 — "return "held by the build loop's run " + run"
  evidence: internal/core/ahoy/worktree/prune_test.go:366 — "func TestPruneKeepsLockedOwnCwdAndHeld(t *testing.T) {"
- ac-16 — MET: On a dry run, reclaim stops before the archive and the removal and reports 'would reclaim'. The core test compares dry-run rows to real-run rows field by field. The CLI test checks the --dry-run --json rows and the exit code.
  evidence: internal/core/ahoy/worktree/prune.go:316 — "if opts.DryRun {"
  evidence: internal/core/ahoy/worktree/prune_test.go:433 — "func TestPruneDryRunReportsTheRowsARealRunWould(t *testing.T) {"
  evidence: internal/surface/cli/ahoy_worktree_test.go:367 — "js, code := runWorktree(t, true, "prune", "--dry-run")"
- ac-17 — MET_WITH_CONCERNS: boardScan runs one peers.Scan and feeds the worktrees member. The line is absent on an empty lane and is not part of statusblock.Block. The test checks held 2 and reclaimable 1, the absence on an empty lane, and the status block. Concern 1: 'reclaimable' is the peers scan's 'spent' estimate, not prune's judgement. It misses a squash-merged branch and counts a locked or freshly cut worktree; the product thinker signed this off as 'can be cleared' in the spec's Open Point. Concern 2: 'starts no per-lane status read' is held by the code's shape alone; no test asserts it.
  evidence: internal/surface/cli/peers.go:413 — "rep, err := peers.Scan(root)"
  evidence: internal/surface/cli/board_worktrees.go:35 — "held, spent := rep.HeldUnder(lane.Dir)"
  evidence: internal/core/peers/read.go:135 — "func (r Report) HeldUnder(dir string) (held, spent int) {"
  evidence: internal/surface/cli/board_worktrees_test.go:24 — "func TestTheBoardCountsTheWorktreesTheLaneHolds(t *testing.T) {"
- ac-18 — MET: The lane comes from gitutil.RootCommit, not the path. The test renames the checkout and gets the same lane, and the stale worktree reads 'checkout not found' with the repair hint.
  evidence: internal/core/ahoy/worktree/worktree.go:82 — "sha, err := gitutil.RootCommit(repoRoot)"
  evidence: internal/core/ahoy/worktree/list_test.go:146 — "func TestListFromAMovedCheckoutFindsTheSameLane(t *testing.T) {"
- ac-19 — MET: finish() narrows the directory git made to the owner. Both the core and CLI add tests check Perm() == 0o700. On a filesystem that cannot change modes, the worktree keeps git's mode by design.
  evidence: internal/core/ahoy/worktree/worktree.go:441 — "_, _ = fsutil.NarrowToOwner(path)"
  evidence: internal/surface/cli/ahoy_worktree_test.go:700 — "if fi, err := os.Lstat(l.Path("feature")); err != nil || fi.Mode().Perm() != 0o700 {"
  evidence: internal/core/ahoy/worktree/worktree_test.go:292 — "if fi, err := os.Lstat(got.Path); err != nil || fi.Mode().Perm() != 0o700 {"
- ac-20 — MET: AGENTS.md's concurrent-sessions convention at HEAD names `scripts/abcd.sh ahoy worktree add <name>` and the store path, and names no `git worktree add` or `../`. The test checks both.
  evidence: AGENTS.md:308 — "Make it with `scripts/abcd.sh ahoy worktree add <name>` and work in the"
  evidence: internal/surface/cli/ahoy_worktree_test.go:1207 — "func TestAgentsNamesTheWorktreeVerbAndNoSiblingForm(t *testing.T) {"
- ac-21 — MET: commands/ahoy.md carries the host-run step: an agent needing an isolated checkout runs add and works in the printed path, never the host's default place. The test checks those phrases.
  evidence: commands/ahoy.md:1746 — "**The host-run step.** When you need an isolated checkout, for a lane, a"
  evidence: commands/ahoy.md:1748 — "path it prints. Do not use the place your host would put a worktree by default,"
  evidence: internal/surface/cli/ahoy_worktree_test.go:1183 — "func TestAhoyPageCarriesTheWorktreeHostRunStep(t *testing.T) {"

Gap audit:
- honoured:
  - add creates the worktree in ~/.abcd.noindex/worktrees/< root-sha>/< name>/ through a level-by-level, symlink- and owner-checked maker, at 0700, and writes nothing beside the checkout
    evidence: internal/core/ahoy/worktree/worktree.go:154 — "func levels(home, rel, what string, create bool) (bool, error) {"
    evidence: internal/surface/cli/ahoy_worktree_test.go:679 — "func TestAhoyWorktreeAddPrintsThePath(t *testing.T) {"
  - one store primitive: the build loop's lane goes through worktree.Add, Adopt, Ensure and LaneFor
    evidence: internal/core/implement/loop/lane_store_test.go:15 — "func TestTheLaneGoesThroughTheStorePackage(t *testing.T) {"
  - prune removes only what passes the proof of belonging and is merged and clean, never forces, keeps the branch, and names every kept worktree with its reason
    evidence: internal/core/ahoy/worktree/prune.go:280 — "func refusedFor(e entry, held Held, cwd string) string {"
    evidence: internal/core/ahoy/worktree/prune_test.go:331 — "func TestPruneLeavesWhatFailsTheProof(t *testing.T) {"
  - the local tier is archived redacted under ~/.abcd.noindex/notes/< root-sha>/< timestamp>-< name>/ with a manifest before removal
    evidence: internal/core/ahoy/worktree/prune_test.go:144 — "func TestPruneArchivesTheTierRedacted(t *testing.T) {"
  - quiet worktrees are listed with a dossier, dirty state first, and go only when named with --yes
    evidence: internal/core/ahoy/worktree/quiet_test.go:144 — "func TestPruneYesRemovesTheNamedQuietCandidateOnly(t *testing.T) {"
  - exit codes 0 reclaimed, 1 nothing reclaimed, 2 refusal or fault, with a dry run answering the real run's code
    evidence: internal/surface/cli/ahoy_worktree_test.go:355 — "func TestAhoyWorktreePruneExitCodes(t *testing.T) {"
  - AGENTS.md and the ahoy plugin page point an agent at the add verb and the printed path
    evidence: AGENTS.md:308 — "Make it with `scripts/abcd.sh ahoy worktree add <name>` and work in the"
    evidence: commands/ahoy.md:1746 — "**The host-run step.** When you need an isolated checkout, for a lane, a"
  - list --all names each registered lane from ~/.abcd.noindex/history/index.json and leaves an unregistered one under its root commit; the registry's path label is never read, and an unreadable or refused registry keeps every lane under its SHA with one path-free note
    evidence: internal/surface/cli/ahoy_worktree.go:433 — "reg, err := ahoy.RegistryLabels()"
    evidence: internal/surface/cli/ahoy_worktree_test.go:260 — "func TestAhoyWorktreeListAllWithoutARegistry(t *testing.T) {"
    evidence: internal/core/ahoy/worktree/list_test.go:235 — "func TestListAllLabelsEveryLane(t *testing.T) {"
- diverged:
  - The board's reclaimable count was promised as the count prune would reclaim. The delivered count is the peers scan's 'spent' estimate (merged by ancestry, record folders clean). It can miss a squash merge and can count a locked or freshly cut worktree. The product thinker signed this off as 'can be cleared'.
    evidence: internal/core/peers/read.go:135 — "func (r Report) HeldUnder(dir string) (held, spent int) {"
    evidence: .abcd/development/specs/closed/spc-2609301811532881-session-and-agent-worktrees-live-in-a.md:460 — "The board's "can be cleared" count is the peers scan's judgement, not prune's"
  - The promise was that the tier is moved and never deleted. The tier's built binary and its < binary>.< pid> build file are left out of the archive and deleted with the worktree (iss-2611022247131866).
    evidence: internal/core/ahoy/worktree/notes_test.go:19 — "func TestPruneLeavesTheTiersBuiltBinaryOutOfTheNotes(t *testing.T) {"
  - 'Merged' was promised as the forge state or patch-equivalence. The delivered ladder adds an ancestry rung that counts only a branch with a commit of its own. A forge answer counts only when the pull request's head is the local tip, so the forge rung is narrower than promised.
    evidence: internal/core/ahoy/worktree/merged.go:157 — "if pr.State == "MERGED" && !pr.IsCrossRepository && pr.HeadRefOid == tip && pr.BaseRefName == branchName(def) {"
    evidence: internal/core/ahoy/worktree/merged.go:170 — "if ownCommit(dir, ref, tip, start) {"
- missing: (none)

Scope-condition dispositions:
- cond-2609301811538239 — narrowed: A worktree the user made by hand outside the lane is listed as outside and never touched. But 'store' versus 'outside' is decided by location alone, with no record of who made a worktree. A worktree placed by hand inside the lane passes the proof of belonging and is reclaimed once merged and clean.
  narrowing: Holds for hand-placed worktrees outside this repository's lane. A worktree placed by hand directly in the lane, of this checkout, is treated as the store's and can be reclaimed.
  evidence: internal/core/ahoy/worktree/list.go:240 — "case filepath.Dir(real) == realLane:"
  evidence: internal/core/ahoy/worktree/prune_test.go:309 — ""beside": "outside the store; never touched""
- cond-2609301811538333 — survived: The store lives under the caller's home (abcdhome). A home that is not a real directory, a level that cannot be made, or a symlinked level refuses before anything is made beneath it, and Add reads the chain before Ensure.
  evidence: internal/core/ahoy/worktree/worktree.go:155 — "if !fsutil.IsRealDir(home) || !fsutil.ValidRelPath(rel) {"
  evidence: internal/core/ahoy/worktree/worktree_test.go:125 — "func TestEnsureRefusesASymlinkedLevel(t *testing.T) {"
- cond-2609301811536785 — survived: The lane key is the full root-commit SHA and nothing else, so a new root commit gives a new lane. A moved checkout keeps its lane.
  evidence: internal/core/ahoy/worktree/worktree.go:82 — "sha, err := gitutil.RootCommit(repoRoot)"
  evidence: internal/core/ahoy/worktree/worktree_test.go:65 — "func TestLaneForKeysOnTheFullRootCommit(t *testing.T) {"
  evidence: internal/core/ahoy/worktree/list_test.go:146 — "func TestListFromAMovedCheckoutFindsTheSameLane(t *testing.T) {"
- cond-2609301811539227 — survived: The judgement is against DefaultRef. It uses the forge (only at the local tip), then ancestry with the branch's own commits, then rebase or squash patch-equivalence. With the forge unreachable, only git decides. A forge answer whose head is not the tip, and unmerged content, both stay unmerged.
  evidence: internal/core/ahoy/worktree/merged.go:153 — "func judgeTip(dir, def, ref, branch, tip, start string, prs []pullRequest) judgement {"
  evidence: internal/core/ahoy/worktree/merged_test.go:238 — "t.Run("forge head is not the tip", func(t *testing.T) {"
  evidence: internal/core/ahoy/worktree/merged_test.go:195 — "t.Run("squash merge", func(t *testing.T) {"
- cond-2609301811533865 — survived: The store and the notes archive sit under the machine home, outside every repository. Every path the verbs print, and the manifest, are home-redacted or in display form, and tests assert that no absolute path reaches the output. Nothing in the delivery syncs the store; that half is held by its absence, not exercised.
  evidence: internal/core/ahoy/worktree/worktree.go:37 — "var StoreRel = abcdhome.Rel("worktrees")"
  evidence: internal/surface/cli/ahoy_worktree_test.go:120 — "func TestAhoyWorktreeListNeverPrintsTheAbsolutePath(t *testing.T) {"
  evidence: internal/core/ahoy/worktree/prune_test.go:222 — "if strings.Contains(string(b), home) || strings.Contains(string(b), pat) {"
<!-- abcd-review-end receipt=rcp-01d317e17525 -->

Audit owed flag cleared 2026-11-15 (receipt rcp-01d317e17525): a re-run of the audit judged no criterion NOT_MET or INCONCLUSIVE, and iss-2611151551240397 is resolved.

## Grounds

- pursued: working copies keep piling up with nothing to list or clear them; wrong if, after it ships, copies still accumulate outside the store.
