---
id: spc-2609301811532881
slug: session-and-agent-worktrees-live-in-a-machine-scoped-store-t
intent: itd-2609091014076309
origin: researcher-authored
production_mode: hand-written
---
# The worktree store: `abcd ahoy worktree add|list|prune`, one lane primitive, and a notes archive beside it

## Summary

This spec delivers
[itd-2609091014076309](../../intents/planned/itd-2609091014076309-session-and-agent-worktrees-live-in-a-machine-scoped-store-t.md):
a session's or an agent's worktree lands in `~/.abcd.noindex/worktrees/<root-sha>/<name>/`,
abcd can say what that lane holds, and abcd reclaims what has merged without
deleting anything it cannot prove belongs to it.

Three verbs land as a sub-verb of `ahoy` (decision 5): **`abcd ahoy worktree
add <name>`** creates a worktree in the lane; bare **`abcd ahoy worktree`** and
**`list`** render every worktree git reports for the repository, marking the ones
outside the store, and `--all` walks every lane on the machine; **`abcd ahoy
worktree prune`** removes a worktree that passes the store's proof of belonging,
whose branch has merged and whose tree is clean, after moving its local tier
into a notes archive. One core package owns both stores, and the build loop's
lane code moves into it, so there is one way a worktree enters the store. The
`/abcd` board gains one line, computed from the scan it already runs.

The rule enacted is
[adr-2609091248200336](../../decisions/adrs/2609091248200336-a-tool-never-creates-directories-in-user-owned-project-space.md);
the store's shape is the transcript store's
([adr-2609091248201071](../../decisions/adrs/2609091248201071-the-transcript-corpus-is-a-sibling-store-that-creates-itself.md)),
and its keying reasons are not repeated here.

## Scope

In: a new package `internal/core/ahoy/worktree` (the lane, `Add`, `List`,
`ListAll`, `Prune`, the notes archive, the merged judgement, the quiet
dossier); `internal/gitutil` (the worktree record carries `locked` and
`prunable`, and the common-directory read moves here from `internal/core/peers`);
`internal/core/peers` (reads the common directory through gitutil, and exposes
the worktrees its scan saw); `internal/core/implement/loop` (the lane step calls
the store; a held-worktree reader for prune); `internal/core/history` (the
sanitise-then-verify pass lifted out of `Capture` so the archive calls the same
function); `internal/core/ahoy` (an exported reader of the registry's labels
and a read-only pull-request query through its existing `gh` runner); the
`ahoy worktree` sub-tree and the board line in `internal/surface/cli`; the plugin
page `commands/ahoy.md`; the concurrent-sessions convention in `AGENTS.md`; the
brief's [`05-internals/03-configuration.md` § The worktree store](../../brief/05-internals/03-configuration.md#the-worktree-store)
and the surfaces register row that points at it; the regenerated surface
snapshot and command reference.

Out, as the intent draws it: deleting branches, tracking refs or remote
branches; moving a worktree that already sits outside the store; worktrees other
tools create; retention of the notes archive; any change to the isolation rule.

## Approach

### One package owns both stores

`internal/core/ahoy/worktree` is the only code that creates, lists or removes a
directory under `~/.abcd.noindex/worktrees/` or `~/.abcd.noindex/notes/`. It lives under `ahoy`
because `ahoy` owns the machine's stores and their registry, and it imports
`ahoy` for the two reads `ahoy` already owns (the scrubbed `index.json` loader,
through a new exported `RegistryLabels`, and the `gh` runner, through a new
exported read-only pull-request query); `ahoy` imports nothing from it, so the
build loop, which already imports `ahoy`, can import it with no cycle. The
package never prints: it returns rows and typed refusals, and the front door
formats them.

The level-by-level store maker is one function serving both stores: each level
of `~/.abcd.noindex/worktrees/<root-sha>/` and of `~/.abcd.noindex/notes/<root-sha>/<entry>/` is
made one at a time with `fsutil.EnsureRealDir` at `0o700`, re-read with
`os.Lstat`, and held to `fsutil.CallersAlone` (owned by this uid, no group or
other write bit) before the next level is made. A symlink or a file anywhere in
the chain, a level owned by another account, or a level writable by group or
others refuses the whole operation, and nothing is made beneath the level that
failed. A level that already exists keeps its mode and is judged as it stands.

### The lane primitive moves out of the loop, and piece 6 becomes its consumer

`laneWorktree`, `ensureStore`, `adoptWorktree` and `safeSegment` move from
`internal/core/implement/loop/lane.go` into the package as:

```go
func LaneFor(repoRoot string) (Lane, error)                   // key from the root commit, full SHA only
func (l Lane) Ensure() error                                  // the level-by-level maker above
func Adopt(repoRoot string, l Lane, name, branch string) (bool, error)
func Add(repoRoot string, opts AddOptions) (Added, error)     // name, new or existing branch, base
func ValidName(name string) bool                              // one plain path segment
```

A refusal is a typed `*Refusal{Reason, Remedy}`, home-redacted, which the
loop wraps in its own `refuse(StepWorktree, …)` so its receipts read as before.

[spc-2609202134338445](spc-2609202134338445-one-verb-takes-a-single-intent-from-ready-to-delivered-witho.md)
piece 6, the lane, is already landed as a plain `git worktree add` into the
store's path and says it takes the store's verb once this intent ships. It
consumes the package, not the CLI verb: the loop keeps what is its own (the
run-id and lane-id shapes in `laneName`, the `build/` branch prefix, the base
cut from the default branch, the pick commit) and calls `LaneFor`, `Ensure`,
`Adopt` and `Add` for everything that touches the store. `Add` sets the new
worktree's directory to `0o700`, so a lane gets the mode as `add` does. The
loop's existing lane tests stay as the proof that its behaviour is unchanged,
and one new test proves the lane went through the package (criterion 5).

### `add`

`abcd ahoy worktree add <name> [--branch <branch>]` resolves the checkout the
caller stands in, keys the lane on `gitutil.RootCommit` (a repository with no
root commit refuses), validates `name` with `ValidName` (letters, digits, `.`,
`_`, `-`; not led by `-` or `.`; no `..`; not empty) and as a branch name with
`git check-ref-format --branch`, and refuses a name already present in the lane,
whether git lists it or not. Without `--branch` it makes a new branch named
`<name>`, cut from the default branch (`gitutil.DefaultRef`) as the loop's lanes
are; a branch of that name that already exists refuses, naming `--branch <name>`
as the way to check it out. With `--branch` naming an existing local branch it
checks that branch out and makes no new one; a `--branch` that names no local
branch refuses. It runs `git worktree add` through gitutil's isolated
environment, then `chmod 0o700` on the directory git made, and prints the path
home-redacted (text) and as `path` (JSON). The only directories it creates are
the store's levels and the worktree itself, so nothing is written beside the
checkout. Exit 0 on success, 2 on any refusal or fault.

### The proof of belonging

A worktree belongs to the store when all three hold:

1. `git worktree list --porcelain` run from this checkout lists it;
2. its real path (`filepath.EvalSymlinks`) is a direct child of the real path of
   this repository's lane `~/.abcd.noindex/worktrees/<root-sha>/`;
3. `git -C <worktree> rev-parse --git-common-dir`, the worktree's own answer, and
   this checkout's common directory resolve to the same real path.

The third test is the one `internal/core/peers/read.go` already makes before it
reads a peer, including its refusal of anything but one line of output; the
function `commonDir` moves to `gitutil.CommonDir` and both packages call it.
`gitutil.Worktree` and `ParseWorktreeList` are extended to carry `Locked`,
`LockReason`, `Prunable`, `PrunableReason` and `Detached`, in both the `-z` and
the newline form (the newline form's C-quoted reason is unquoted), because
`list` renders them and `prune` refuses on them.

### `list` and `list --all`

Bare `abcd ahoy worktree` and `list` render one row per worktree git reports for
the repository, the main checkout excepted: name, branch (or `detached`), path,
`clean` or `dirty`, `merged` or `unmerged`, and a `where` column of `store` or
`outside`. Clean is `git -C <worktree> status --porcelain -z
--untracked-files=all` returning nothing, the same judgement `git worktree
remove` makes, so git-ignored files (the local tier) do not make a tree dirty. A
lane row that fails the proof renders its reason in place of the state columns:
a dead `.git` pointer reads "checkout not found; run `git worktree repair` from
the checkout", a directory git lists as prunable reads "directory gone; git
still lists it". A directory in the lane that git does not list renders as
"not a worktree of this repository", and a directory directly under
`~/.abcd.noindex/worktrees/` whose name is a prefix of this repository's root commit but
not the full key renders as "not a lane", naming this repository's full-key
lane and saying its worktrees are retired by hand with `git worktree remove`.

Moving the checkout changes nothing here: the lane is found from the root
commit, and git's own records travel with the common directory, so the moved
checkout lists the same lane; a worktree whose `.git` pointer still names the old
path fails proof 3 and says to run `git worktree repair` (criterion 18).

`--all` walks every directory under `~/.abcd.noindex/worktrees/` without following
symlinks. A directory named by a full root commit is a lane, labelled with the
repository's name from `ahoy.RegistryLabels()` (the loader that already scrubs
`index.json`) where that commit is registered and by the SHA where it is not.
Which checkout a lane belongs to is read from each worktree's own `.git` file
(`gitdir: <common>/worktrees/<id>`), never from the registry's `path` label, so
a label naming a directory that no longer exists changes no row. A lane whose
`.git` pointers are dead renders as one "checkout not found; run `git worktree
repair` from the checkout" row under its SHA with no clean or merged column. A
directory whose name is not a full root commit renders as "not a lane"; when it
is a prefix of the current repository's root commit the row names that
repository's full-key lane, and every such row says its worktrees are retired by
hand. Under `--all` the clean and merged columns are computed from each lane's
own checkout, found through its `.git` pointer.

`--json` emits the same rows as an array under `worktrees` (and `lanes` for
`--all`), every collection an array, never null. Exit 0 on success, 2 on a fault.

### Paths on a machine stream

A store row's path is `fsutil.RedactHome` (`~/.abcd.noindex/worktrees/…`), because the
reader must be able to `cd` into it. An outside row's path is
`fsutil.DisplayPath`: home-relative under HOME, the base name outside it, so it
stays recognisable wherever it lives without printing an absolute local path. A
branch name, a lock reason and a path are another checkout's bytes, and pass
through `termsafe.Sanitize` before they reach a terminal or the JSON envelope.

### Merged

A worktree's branch is merged into the default branch when any of these holds,
checked in order (decision 2):

1. **The forge records it.** Where a forge is reachable, the branch's pull
   request is merged and its recorded head commit is the local branch tip. The
   head-commit test keeps a branch that gained commits after its merge from
   reading as merged. The query is one read-only `gh pr list` per run for the
   lane's branches, through `ahoy`'s existing runner and its timeout, made by the
   caller's own identity.
2. **By ancestry, with commits of its own.** The tip is an ancestor of the
   default branch and is not the point the branch was created from (the oldest
   entry of the branch's reflog). A branch cut and never committed to is not
   merged, however clean; a branch with no reflog to prove its creation point is
   judged by the content tests alone.
3. **By content, commit by commit.** `git cherry <default> <branch>` lists at
   least one commit and every commit is marked equivalent (a rebase merge).
4. **By content, as one change.** The stable patch-id of the branch's whole diff
   from its merge base equals the patch-id of a commit on the default branch
   since that merge base (a squash merge).

"No forge reachable" is any of: `gh` not on PATH, not authenticated, no GitHub
remote, or no answer within the timeout. The run then judges by 2 to 4 alone
and says so once. The merged column carries how it was judged (`forge`,
`ancestry`, `content`), so a reader can tell a forge answer from a content one.
The `list` and `prune` pages document the forge read as part of what the verb
does, which is what keeps it inside
[invariant 7](../../brief/02-constraints/03-invariants.md): a fetch only when
the user invokes a verb whose documented meaning is that fetch. The board never
asks the forge.

### `prune`

`abcd ahoy worktree prune [--dry-run] [--yes <name>]… [--json]` reads the lane,
applies the proof and the merged judgement, and for each worktree decides one
row: `reclaimed`, `quiet` (a candidate, below) or `kept` with a reason. Kept
reasons, each named in the row:

- fails the proof of belonging (not listed by git, outside the lane, another
  checkout's common directory, a dead `.git` pointer): reported, untouched;
- outside the store: listed and never touched;
- locked, with git's lock reason;
- prune's own working directory (the real path of the cwd is the worktree or
  inside it);
- held by the build loop: a lane of a run whose state is not complete names this
  worktree. The loop exports `HeldWorktrees(repoRoot)`, read across the local
  tier of every checkout git lists for the repository, because a run's state
  lives in the checkout that started it; the front door hands it to `Prune` as a
  seam, the shape `statusblock.LaneReader` already takes, so the store never
  imports the loop that imports it. A state file that cannot be read holds every
  worktree named after its run id;
- dirty;
- unmerged (and not a quiet candidate);
- the notes could not be archived (the archive's reason).

A reclaimable worktree is removed by archiving its notes, then running `git
worktree remove <path>` without `--force`. Git re-checks the tree and the lock
at that moment and refuses a tree that changed since it was judged; the refusal
becomes a `kept` row with git's reason, and the archive entry already written
stays and is named, since the tier still stands in the kept worktree and
nothing is lost. `git worktree remove` deletes that worktree's administrative
entry with it, which is git's metadata prune for that one worktree; the
repository-wide `git worktree prune` is never run, because it would also drop
the entry of an outside worktree whose directory is only temporarily absent.
Prune never passes `--force`, never deletes a branch, and deletes nothing git
did not delete. The invariant this enacts is the brief's: the store deletes only
what passes its proof of belonging.

A store refusal (a symlinked level, a level another account owns or others can
write) stops the run before anything is judged: exit 2, nothing removed.
Otherwise the exit code is 0 when at least one worktree was reclaimed, 1 when the
run completed and reclaimed nothing (an empty lane included), and 2 on a
refusal or fault (decision 8). `--dry-run` makes no archive and removes nothing,
and reports the rows a real run would, with `would reclaim` in place of
`reclaimed`; its exit code is the one the real run would return. `--json` emits
the same rows under `rows`, with `archive` naming the notes entry on a reclaimed
row.

### The notes archive

A worktree's `.abcd/.work.local/` is walked without following symlinks. When it
holds anything, `prune` makes `~/.abcd.noindex/notes/<root-sha>/<YYYYMMDDTHHMMSSZ>-<name>/`
through the shared level maker (the entry itself made exclusively, so two runs
in one second cannot share it) and writes into it, inside an `os.Root` opened on
the entry, each regular file at its relative path, `0o600`, and a
`manifest.json`: `schema_version`, `worktree`, `branch`, `source`
(home-redacted), `archived_at` (UTC), `root_sha`, and `files` (path, size and
the SHA-256 of what was written). A symlink is recorded in the manifest as a
link, target home-redacted, and not followed.

Every file and every manifest field is redacted on write through the same pass
the transcript store uses. The body of `history.Capture` that does it (refuse a
degraded scanner; scan with the per-repository scanner and the armed gitleaks
adapter; `scanner.Redact`; the caller-home sweep and its survivor refusal; the
stage-two re-scan that refuses any blocking residual) is lifted into one
exported function in `internal/core/history`, and both `Capture` and the
archive call it, so there is one redaction path, not two. A file that is not
text, is over the per-file cap, or leaves a residual fails the archive, and a
failed archive keeps the worktree, named with the reason and the file's
relative path; nothing from the tier is ever deleted by a failed step. A tier
that is absent or empty writes no entry, and the row says there were no notes.
The report names the archive path, home-redacted.

### Quiet candidates

An unmerged worktree in the lane is quiet when its last activity is older than
the quiet window and no pull request for its branch is open (where a forge is
reachable; without one the window alone decides) (decision 3). Last activity is
the later of the branch tip's committer date and the newest modification time of
the files `git status` reports changed or untracked. The window is the layered
configuration key `worktrees.quiet_days` in `.abcd/config.json`, read through
`internal/core/layered` (the repository file, then `~/.abcd.noindex/config.json`, then
the bundled default of 14), which claims the `worktrees` namespace, so a
misspelt key or a value below 1 is refused naming its file, never replaced by
the default.

Prune lists each quiet candidate with a dossier, in this order: its dirty state
first (counts of modified and untracked files, and the first paths), the
subjects of the branch's commits not on the default branch, a diff summary
(`--shortstat` from the merge base), the record ids the subjects and changed
paths cite, its last activity, and a recommendation ("commit or discard the
changes, then name it", "reclaim with `prune --yes <name>`", or "open a pull
request if the work is wanted"). A candidate is left in place unless named with
`--yes <name>` (repeatable); the core never prompts. A named candidate takes the
same archive step and the same refusals as a merged worktree, a dirty tree
included, so uncommitted work stays until the person commits or discards it.
Every `--yes` name is checked before anything is removed: a name that is not in
the lane refuses the run (exit 2, nothing removed), and a name in the lane that
is not a quiet candidate this run is kept and named with why (merged ones need no
name; an active one is not quiet).

### The board line

The `/abcd` board's `boardPeers` already runs `peers.Scan` for the checkout.
The scan is run once and feeds two members: `peers`, unchanged, and a new
`worktrees` member, `{"held": N, "reclaimable": M}`, rendered as one text line
("worktrees: N in the store, M reclaimable (`abcd ahoy worktree prune`)") and
omitted when N is 0. `peers.Report` gains an accessor listing each linked
worktree its scan saw, with its path and whether the scan judged it spent
(merged by ancestry, record folders clean); the board counts those whose real
path lies inside this repository's lane as held, and the spent ones among them
as reclaimable. The lane path costs one root-commit read (`gitutil.RootCommit`)
for the checkout, not a read per lane, and the line starts no status read of its own.
The member is on `boardOutput` in `internal/surface/cli` and nowhere else: the
Now / Next / Later block the site renders (`statusblock.Block`) does not carry
it.

### The surfaces

`commands/ahoy.md` gains a `worktree` section documenting `add`, `list`
(`--all`), `prune` (`--dry-run`, `--yes`), the forge read, and the host-run step
(decision 6): an agent that needs an isolated checkout (a lane, a subagent, a
peer session) runs `abcd ahoy worktree add <name>` and works in the printed
path, never a host's default worktree location. The prose stays host-agnostic.

`AGENTS.md`'s concurrent-sessions convention replaces "**The store has no verbs
yet.** Aim a plain `git worktree add` at the path and create the lane by hand…"
with the verb: a session gets its own checkout with `go run ./cmd/abcd ahoy
worktree add <name>` (in this source checkout) and works in the printed path;
bare `ahoy worktree` lists the lane and `ahoy worktree prune` reclaims what has
merged. The paragraph keeps its reasons for the location and names no sibling
directory as a place to work.

The brief's worktree-store section carries the store primitive and the
invariant (decision 1), and the surfaces register's row for the store points at
the `ahoy` chapter once the verb ships. The surface snapshot and the generated
command reference are regenerated with the sub-tree.

## How each acceptance criterion is met

1. `Add` makes the worktree at `~/.abcd.noindex/worktrees/<root-sha>/<name>/` on the
   branch, the front door prints the path, and the only directories created are
   the store's levels and the worktree, so the checkout's parent is unchanged.
2. The shared level maker makes and re-verifies each level in turn; a symlink at
   any level refuses before anything beneath it is made.
3. `fsutil.CallersAlone` on every level refuses another account's level or one
   group or others can write, before anything is written.
4. `ValidName`, the lane-occupancy check and the root-commit check refuse with
   the reason; `--branch` on an existing branch checks it out with no new branch.
5. The loop's lane step calls `LaneFor`, `Ensure`, `Adopt` and `Add`; a test
   proves the lane was made through the package.
6. Bare `list` renders every worktree git reports, with the `where` column
   marking the sibling as outside, and branch, clean and merged on each.
7. `--all` labels lanes from `RegistryLabels()` or by SHA, reads ownership from
   each worktree's `.git` file (so a stale registry path changes no row),
   renders a dead pointer as "checkout not found" with the repair hint, and an
   abbreviated directory as "not a lane", naming the full-key lane on a prefix
   match and saying its worktrees are retired by hand.
8. A merged, clean worktree passing the proof is removed by `git worktree
   remove`, which drops git's entry for it; the branch is never deleted; the row
   names it.
9. The squash test (patch-id of the whole branch diff against the default
   branch's commits) judges it merged with no forge, and it is reclaimed.
10. The archive writes the redacted files and the manifest under
    `~/.abcd.noindex/notes/<root-sha>/<timestamp>-<name>/` before removal and the row
    names the path; a failed archive keeps the worktree with the reason.
11. The quiet rule lists the candidate with its dossier (dirty state first) and
    leaves it; `--yes <name>` archives and removes that one only; a dirty named
    candidate is kept because removal is never forced.
12. A store refusal stops the run before judgement: exit 2, nothing removed.
13. Dirty, unmerged and outside rows are kept with reasons and the run exits 1;
    with a merged, clean worktree added it exits 0.
14. The proof of belonging fails for a directory git does not list, or whose
    common directory is another checkout's; it is reported and untouched.
15. The locked, own-cwd and held-by-the-loop refusals keep all three, and no
    removal passes `--force`.
16. `--dry-run` computes the same rows and skips the archive and the removal;
    `--json` emits those rows.
17. The board's `worktrees` member comes from the one `peers.Scan`, is omitted on
    an empty lane, and is not part of `statusblock.Block`.
18. The lane is found from the root commit, so the moved checkout lists the same
    lane.
19. `Add` sets the worktree directory to `0o700` after git creates it.
20. `AGENTS.md`'s convention names the verb and no sibling-directory form.
21. `commands/ahoy.md` carries the host-run step.

## Settled here, not by the intent

These are the facilitator's design calls, each in the direction that keeps a
worktree rather than loses one; none changes a criterion.

- `add` cuts its new branch from the default branch, as the loop's lanes are.
- A branch with no commits of its own is not merged by ancestry (reflog creation
  point), so a freshly added, clean worktree is never reclaimed.
- A forge answer counts only when the pull request's head commit is the local tip.
- `list` asks the forge as `prune` does, documented as part of the verb, so the
  merged column means one thing everywhere; the board never asks.
- The intent's "git's own metadata prune for it" is `git worktree remove`, which
  drops that worktree's entry; the repository-wide `git worktree prune` is never
  run.
- A file the scanner cannot redact (not text, over the cap, a residual) fails
  the archive and keeps the worktree.
- An absent or empty local tier writes no archive entry.
- `--yes` admits only a quiet candidate; a name not in the lane refuses the run.
- `--dry-run` returns the exit code the real run would.
- "Held by the build loop" is a lane of a run whose state is not complete, read
  across every checkout of the repository.
- 2026-10-04: the store's paths are spelled under the renamed home,
  `~/.abcd.noindex/worktrees/` and `~/.abcd.noindex/notes/`, the location
  [adr-2610031751065746](../../decisions/adrs/2610031751065746-the-worktree-store-lives-under-the-renamed-home-abcd-noindex.md)
  sets when it supersedes adr-2609091248200336's `~/.abcd/` spelling
  (spc-2610031309233367, step 4). Nothing else in this record changes.

## Open point

_None open._ The board's "reclaimable" is the peers scan's judgement, not prune's: it can miss a squash-merged branch and can count a locked or freshly cut worktree. The product thinker ruled on 2026-09-30 that the line keeps the words "can be cleared" as an estimate, and that prune's own report is the authority on what was cleared.

## Footprint

- packages: internal/core/ahoy/worktree, internal/core/ahoy, internal/core/implement/loop, internal/core/peers, internal/core/history, internal/gitutil, internal/surface/cli, commands/, AGENTS.md, .abcd/development/brief/05-internals, .abcd/development/brief/04-surfaces
- tests: the level maker over a symlinked, foreign-owned and group-writable level; add's path, mode, branch and name refusals; the loop's lane tests unchanged plus one proving the lane goes through the package; list over a lane of two plus a sibling; list --all over registered, unregistered, moved and abbreviated lanes; the merged judgement over ancestry, a never-committed branch, a rebase merge and a squash merge with no forge; prune's reclaim, keep reasons, exit codes and dry-run; the notes archive's redaction, manifest and failure path; quiet candidates and --yes; the board line from one scan and its absence from the site block; the worktree parser's locked and prunable fields in both forms

## Steps

1. The store package and the lane primitive
   - criteria: 2, 3, 5
   - packages: internal/core/ahoy/worktree, internal/core/implement/loop, internal/core/peers, internal/gitutil
   - tests: the level maker refuses a symlinked, foreign-owned or group-writable level and makes nothing beneath it; `ParseWorktreeList` carries locked and prunable in the `-z` and newline forms; `gitutil.CommonDir` refuses a multi-line answer; the loop's existing lane tests pass unchanged and a new one proves the lane goes through the package
2. `ahoy worktree add`
   - criteria: 1, 4, 19
   - packages: internal/core/ahoy/worktree, internal/surface/cli
   - tests: add makes the worktree in the lane, prints its path, leaves the checkout's parent unchanged and sets `0o700`; `../x`, `-x`, a taken name and a repository with no commits refuse and write nothing; `--branch` checks out an existing branch and makes none
3. `list`, `list --all` and the merged judgement
   - criteria: 6, 7, 18
   - packages: internal/core/ahoy/worktree, internal/core/ahoy, internal/surface/cli
   - tests: three rows for a lane of two plus a sibling, the sibling outside; registered and unregistered lanes under their labels, a stale registry path changing nothing, a moved checkout's lane as "checkout not found", an abbreviated directory as "not a lane" naming the full key; the same lane from a moved checkout; the merged judgement over ancestry, a never-committed branch, a rebase merge and a squash merge with the forge unreachable, and a forge answer whose head is not the tip
4. `prune` for merged worktrees, with the notes archive
   - criteria: 8, 9, 10, 12, 13, 14, 15, 16
   - packages: internal/core/ahoy/worktree, internal/core/history, internal/core/implement/loop, internal/surface/cli
   - tests: a merged clean worktree removed with its branch standing; a squash-merged one reclaimed with no forge; the tier archived redacted beside its manifest and a failed archive keeping the worktree; a store refusal exits 2; dirty, unmerged and outside kept with exit 1 and exit 0 once one is reclaimed; a directory failing the proof untouched; locked, own-cwd and loop-held kept; dry-run and `--json` matching a real run's rows; `history.Capture` still passing its suite through the lifted pass
5. Quiet candidates
   - criteria: 11
   - packages: internal/core/ahoy/worktree, internal/surface/cli
   - tests: a 15-day-quiet unmerged worktree listed with its dossier, dirty state first, and left; `--yes <name>` removes that one only; a dirty named candidate kept; `worktrees.quiet_days` read through the layered resolver and a bad value refused; a `--yes` name not in the lane refusing the run
6. The board line
   - criteria: 17
   - packages: internal/core/peers, internal/surface/cli
   - tests: one line with the held and reclaimable counts from one scan; absent on an empty lane; absent from the site's rendered status block
7. The surfaces
   - criteria: 20, 21
   - brief: when the store ships, `02-constraints/03-invariants.md` gains the invariant "the store deletes only what passes its proof of belonging" (adr-2609091248200336), held by the prune refusal tests; it is not added before the code that holds it exists.
   - packages: commands/, AGENTS.md, .abcd/development/brief/05-internals, .abcd/development/brief/04-surfaces, internal/surface/cli
   - tests: the plugin page carries the host-run step; `AGENTS.md` names the verb and no sibling form; the surface snapshot and command reference regenerated; docs-lint and record-lint clean
