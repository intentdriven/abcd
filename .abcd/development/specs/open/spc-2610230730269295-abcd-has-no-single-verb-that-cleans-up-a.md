---
id: spc-2610230730269295
slug: abcd-has-no-single-verb-that-cleans-up-a
intent: itd-2610222214476887
origin: researcher-authored
production_mode: hand-written
---
# `abcd sweep`: one dry run over every leftover, and `--apply` for what is provably spent

## Summary

spc-2610230730269295 delivers itd-2610222214476887. `abcd sweep` is one verb over five kinds of leftover:
- worktrees;
- local branches;
- merged pull-request branches on origin;
- reservation branches;
- waiting inbox reports.

A bare run changes nothing: it lists every item with what `--apply` would do, or why it would leave the item. `--apply` acts only on what is provably spent.

The verb adds two things of its own, a liveness probe and patch salvage. Everything else is primitives it calls:
- the worktree store's list, merged judgement and prune (itd-2609091014076309, which this spec waits on);
- the reservation sweep's judgement and its workflow (iss-2610142200389875);
- the inbox's promote (internal/core/report);
- the claim package's reservation view;
- the embargo store's path test.

## Approach

### Rows

One `Row` per item, carrying:
- kind (worktree, branch, remote-branch, reservation, inbox, stray or embargo-store);
- name;
- path, home-redacted;
- the state the sweep read;
- action (remove, delete, dispatch, promote, save-patch or keep);
- reason;
- for each piece of evidence, whether it is proven or a hint.

A dry run renders the rows. `--apply` renders the same rows, each with its outcome, and `--json` emits them. Exit codes:
- 0 when anything was acted on;
- 1 when the run completed and acted on nothing;
- 2 on a refusal or a fault.

### Worktrees

**Rows.** They come from the store's list (git's `worktree list --porcelain`, so worktrees outside the store appear too) and the store's merged judgement. That judgement reads ancestry, a rebase merge, a squash merge by patch-equivalence, and the forge where it is reachable.

**Removal.** It is the store's prune core, called per worktree. Prune archives the notes, never forces, leaves the branch standing, and refuses locked, own and loop-held worktrees. The sweep adds no reclaim of its own.

**Keep reasons the sweep adds on top of prune's:**
- the embargo store, named by `storepath.IsStorePath`, which is never read, saved or removed;
- a live session (below);
- an external lock, with git's lock reason shown and a JSON reason's `owner` named.

### Liveness

A worktree is held when any of these proofs names it:
- a live reservation's footprint on origin, read through `claim.View` after the sweep's fetch;
- the build loop's run state (`loop.Runs`) naming it as a lane, for an unfinished run within the lease;
- a running process whose working directory is inside it.

The process proof reads `/proc/<pid>/cwd` on Linux. On macOS it runs `/usr/sbin/lsof -a -d cwd -Fn` by absolute path, never resolved through PATH, with a 5-second deadline, a 4 MiB bounded read and defensive parsing of the `-F` fields. It uses no cgo, because the released binaries are cross-compiled.

A last write to the tree or its index within 24 hours (configurable) is a hint, not a proof. The hint also keeps the worktree.

The probe fails closed. A worktree the probe could not clear counts as possibly live whenever any of these happens:
- lsof is missing, times out, exits non-zero, or prints what the parser cannot read;
- `/proc` cannot be read.

Such a worktree is left in place and named "process check unavailable: <why>", and the dry run says so once at the top.

### Patch salvage

Patch salvage applies to a dirty worktree that no proof holds:
- Its `git diff --binary` and `git diff --cached --binary`, and a tar of its untracked, non-ignored files, are written to `~/.abcd.noindex/notes/<root-sha>/<UTC>-<name>-salvage/`.
- The write goes through the store's directory primitive and redaction.
- A manifest names the worktree, its branch, HEAD, the home-redacted source path and the file list.
- The worktree is kept.
- A second run that finds an identical diff (by sha256) writes nothing new and points at the earlier salvage.

The embargo store and any worktree outside the store are never salvaged.

### Local branches

A local branch with no worktree is judged by the same merged judgement. It is deleted with `git branch -d`, never `-D`, so git re-checks the merge.

These are kept, with the reason:
- the default branch;
- the checked-out branch;
- a branch with commits not on its upstream;
- reservation refs.

### Merged pull-request branches on origin

Under `--apply`, a branch on origin is deleted when the forge records its pull request as merged, or when it is patch-equivalent to the default branch, once nothing still needs it:
- no open pull request heads it;
- no reservation names it;
- no running session works on it, judged by the liveness proof against any local worktree on that branch.

The deletion is a push of an empty ref carrying a lease on the tip that was judged (the reservation sweep's pattern), made through the guard, and each branch it deletes is named.

These are never deleted:
- the default branch;
- release and tag refs;
- `abcd-reserved/*`;
- a branch the sweep cannot show is merged.

A branch something still needs is kept, naming what holds it. There is no separate opt-in (the product thinker's ruling of 2026-10-23).

### Reservations

- **Rows.** They come from the reservation sweep's dry-run judgement (`reservesweep.Sweep` with Apply false).
- **Under `--apply`.** When any reservation is spent, the sweep runs `gh workflow run reservation-sweep.yml --ref <default>`, and the workflow deletes them then. The sweep reports the run's id and URL, read back with `gh run list --workflow reservation-sweep.yml --limit 1`.
- **Fallback.** Where `gh` is absent or the dispatch fails, the report gives the next scheduled time, read from the workflow's cron.
- The sweep never pushes a reservation deletion itself.

### Inbox

Under `--apply`, every waiting report is promoted through internal/core/report's promote. Each is filed with the automatic filer's remedy, so the drain never acts on one unread, and the report names each record it filed. A report's text stays framed as untrusted in the rows, and promotion writes a record and runs nothing.

### Strays

An entry in the store lane that is neither a worktree nor a directory failing the store's proof of belonging (a loose file, say) is listed as a stray and never touched.

## How the Acceptance Criteria are met

1. A dry run lists every item and changes nothing: Rows; no writer runs without `--apply`.
2. A patch-equivalent merge is removed: the store's merged judgement and prune.
3. A worktree whose tip is the default branch but which holds a live working directory is kept: Liveness, the process proof.
4. A dirty worktree is saved as a patch and kept: Patch salvage.
5. The embargo store is untouched: the `storepath.IsStorePath` keep reason.
6. A worktree another tool has locked is kept, naming the owner: the lock keep reason.
7. A stray is listed: Strays.
8. A merged pull-request branch is cleared once nothing needs it, or kept naming what holds it: Merged pull-request branches on origin.
9. Reservations are dispatched, never deleted directly: Reservations.
10. Inbox reports are filed: Inbox.
11. The technical list:
    - one reclaim primitive (the store's prune);
    - a fetch first;
    - no force and no process stopped;
    - `--json`;
    - exit codes 0, 1 and 2;
    - writes only under `~/.abcd.noindex/` and the local tier.

## Footprint

- packages: internal/core/sweep/sweep.go, internal/core/sweep/rows.go, internal/core/sweep/liveness.go, internal/core/sweep/salvage.go, internal/core/sweep/branches.go, internal/core/sweep/remote.go, internal/core/sweep/reservations.go, internal/core/sweep/inbox.go, internal/surface/cli/sweep.go, internal/surface/cli/helpgroups.go, internal/surface/cli/barerender.go, internal/surface/cli/cli.go, commands/sweep.md, docs/reference/cli/commands.md, .abcd/development/release/surface.json, .abcd/development/brief/04-surfaces/40-sweep.md
- tests: internal/core/sweep/*_test.go, internal/surface/cli/sweep_test.go

## Steps

1. Rows, liveness and the dry run
   - criteria: 1, 3, 5, 6, 7, 11
   - packages: internal/core/sweep/sweep.go, internal/core/sweep/rows.go, internal/core/sweep/liveness.go
   - tests: the dry run lists every kind and writes nothing; each keep reason (dirty, unmerged, locked naming its owner, live by each proof, the recent hint, the embargo store, a stray, unclassifiable); a worktree whose tip is the default branch with a live working directory kept; the probe missing, timing out or unreadable leaving the worktree as possibly live and saying so; exit codes. Waits on the store's list and merged judgement (itd-2609091014076309 steps 1 and 3).
2. Worktree apply through the store's prune, and patch salvage
   - criteria: 2, 4
   - packages: internal/core/sweep/salvage.go, internal/core/sweep/sweep.go
   - tests: a patch-equivalent merge reclaimed through prune; salvage content, manifest and idempotence; a failed salvage keeping the worktree. Waits on the store's prune (itd-2609091014076309 step 4).
3. Local and merged origin branches
   - criteria: 8
   - packages: internal/core/sweep/branches.go, internal/core/sweep/remote.go
   - tests: `branch -d` honoured and its refusal kept; a merged origin branch deleted under a lease and named; one held by an open pull request, a reservation or a live session kept, naming what holds it; the never-delete list.
4. Reservations and inbox
   - criteria: 9, 10
   - packages: internal/core/sweep/reservations.go, internal/core/sweep/inbox.go
   - tests: the reservation workflow dispatched and its run reported; the fallback to the next scheduled time with no gh; every waiting report promoted with the automatic filer's remedy and named. Waits on iss-2610142200389875's sweep being on main.
5. The surfaces
   - criteria: 1, 11
   - packages: internal/surface/cli/sweep.go, internal/surface/cli/helpgroups.go, internal/surface/cli/barerender.go, internal/surface/cli/cli.go, commands/sweep.md, docs/reference/cli/commands.md, .abcd/development/release/surface.json, .abcd/development/brief/04-surfaces/40-sweep.md
   - tests: internal/surface/cli/sweep_test.go: the verb's rows, `--json` and exit codes through the front door; the reference and surface baseline regenerated.
