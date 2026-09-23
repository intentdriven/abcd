---
name: peers
description: List the records this checkout's sibling worktrees and local branches hold that it does not — before capturing, fixing or filing anything — by invoking the abcd binary. Strictly read-only.
---

# `/abcd:peers`

Show what every peer of this checkout holds that this tree does not, so a
session sees a sibling's capture or fix before it mints a duplicate or re-fixes
a closed record. This command performs **zero writes**: it takes no lock,
fetches nothing and stamps nothing.

A peer is one of two things, both sharing this repository's git common dir:

- a **linked worktree**, read off its disk, so a capture nobody has committed
  yet is seen;
- a **local branch** no worktree has checked out, read from the object store.

Run:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" peers --json
```

The payload carries `live` (the live-peer count), `ids` (the distinct records
across every row), `default_ref` (the branch a peer is judged merged into),
`sources` (the sources read: `worktree` and `branch`), `peers` and `skipped`.
Each peer names its `source`, `branch`, `path` (home-redacted to `~`) and its
`rows`; each row is an `id`, a `kind`, the `folder` that holds it in the peer
and, when the file could be read, a `title`:

- `open-there` — an issue open in the peer and absent from every status folder
  here: the peer captured it. Do not capture it again.
- `terminal-there` — an issue open here and `resolved` or `wontfix` in the
  peer: the peer closed it. Do not fix it again.
- `draft-there` — an intent drafted in the peer and absent here.

A peer with `not_read` set was named and not read, with the reason: git refused
to answer for it, its common dir is another repository's, its ledger holds one
id in two status folders (the reason names the id and the remedy), or it holds
no records at the committed layout. A worktree whose directory is gone or that
git will not read leaves its branch in the object store, so that branch is read
there as a `branch` peer. `skipped` lists the spent peers — a worktree
whose directory is `gone`, or a branch `merged` into the default branch (a
worktree counts as merged only when its record folders are also clean) — which
contribute no rows.

Tell the user the counts, then each peer with rows, grouped by peer. When a
peer holds the record the user is about to capture, resolve or plan, say so
before doing anything, and name the branch and path. With no peers the verb
says so and exits 0. Outside a git checkout it refuses (exit 2).

The same reader answers three not-found paths: `capture resolve`, `abcd
<record-id>` and `intent audit`, asked for a record this checkout does not hold
and a peer does, refuse naming the peer's branch, path and folder instead of
answering not found. The bare `/abcd` board carries one `peers:` line when any
peer holds a record that differs here.

Scope: only the peers that share this checkout's common git dir. A second
clone, another account and another machine are not peers here, and a record in
a stash or an unsaved buffer is invisible.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
