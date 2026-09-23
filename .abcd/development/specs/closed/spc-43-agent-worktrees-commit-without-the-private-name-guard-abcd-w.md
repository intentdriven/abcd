---
id: spc-43
slug: agent-worktrees-commit-without-the-private-name-guard-abcd-w
intent: itd-150
---
# agent-worktrees-commit-without-the-private-name-guard-abcd-w

## Summary

Makes the private name-guard follow the developer across git worktrees. The
committed pre-commit hook that enforces the banlist resolves its store from the
current worktree alone, and `.abcd/.work.local/` is per-worktree and gitignored,
so a linked worktree created by `git worktree add` starts with an empty store
and every commit made there runs with the banlist absent — loudly warned, but
unprotected. This spec teaches the hook to resolve the primary checkout's
`.work.local` store when it runs inside a linked worktree, merge it under the
worktree-local store (local entries win), and keep the existing loud-warning
fallback when no store resolves at all.

## Scope

In:

- `internal/core/ahoy/defaults/pre-commit` — the committed shell hook that reads
  the banlist and blocks the commit. Add primary-worktree resolution, a second
  store root, and a two-root merge of the entry-collection loop.
- `internal/surface/cli/banlist.go` (around lines 433–445) — the CLI verbs'
  store-root resolver, kept in lockstep with the hook so `abcd banlist` reads
  the same effective store the guard enforces.
- `internal/core/banlist/hook_test.go` — a new linked-worktree scenario.

Out:

- No change to the two-layer model itself (public committed list in
  `.abcd/docs-lint.json`; private per-machine list in
  `.abcd/.work.local/private-names.txt`). The public layer is unaffected.
- No change to the Go store primitives in `internal/core/banlist/private.go`
  (`readPrivate`, `AddPrivate`, `RemovePrivate`, …): they remain
  `repoRoot`-parameterised. Only the two callers that *resolve* the root — the
  hook and the CLI resolver — become worktree-aware.
- No cross-worktree write path: entries are still added to the store of
  whichever checkout the verb runs in. This spec is read-side resolution only.

## Approach

**The mechanism — distinguish a linked worktree and locate its primary.** In a
linked worktree, `git rev-parse --git-dir` resolves to
`<primary>/.git/worktrees/<name>` while `git rev-parse --git-common-dir`
resolves to `<primary>/.git`; in the primary checkout the two are equal. The
primary worktree root is the parent of the common dir. The hook currently
resolves only the current root, at `pre-commit` line 151
(`toplevel=$(git rev-parse --show-toplevel)`), then sets
`banlist=".abcd/.work.local/private-names.txt"` relative to it (line 161).

The change adds, after the fail-closed `cd "$toplevel"` (lines 152–157):

1. `common_dir=$(git rev-parse --path-format=absolute --git-common-dir)` and
   `git_dir=$(git rev-parse --path-format=absolute --git-dir)`.
2. When `git_dir != common_dir`, this is a linked worktree; the primary root is
   `dirname "$common_dir"`. Set `primary_banlist="$primary_root/.abcd/.work.local/private-names.txt"`.
3. When they are equal (standalone checkout), `primary_banlist` is unset and the
   hook behaves exactly as today.

**Merge, local-wins.** The entry-collection loop (arrays `keys`, `pats`,
`lines` at `pre-commit` lines 379–464, with format-decl detection at 320–359)
is refactored into a shell function `load_store <path>` that appends parsed
entries into the arrays. The hook calls it for the **primary** store first, then
the **worktree-local** store second, so a duplicate key from the local store
overrides the primary entry — the intent's precedence rule (a worktree-local
entry is never overridden by the fallback). The scan against staged content
(lines 644–674) is unchanged; it reads the merged arrays.

**The store-present decision.** Today `store_present` is set from a single
`[ -f "$banlist" ]` test (lines 300–311). It becomes true if *either* the
worktree-local or the primary store exists. Only when neither resolves does the
hook take the existing loud-warning branch (lines 300–311 / the entryless branch
487–495) and let the commit through — the graceful, non-fail-closed fallback the
intent requires for a standalone checkout with no store. The scratch-dir logic
(lines 521–544) is untouched: it already falls back to `mktemp` when the local
tier is absent.

**CLI parity.** `internal/surface/cli/banlist.go` (lines 433–445) resolves the
store root via `git rev-parse --show-toplevel`; its comment already pins it to
"what the pre-commit guard enforces at". It gains the same common-dir resolution
so that `abcd banlist` in a linked worktree renders the merged view the guard
sees. The Go read path reuses `readPrivate(primaryRoot)` for the fallback layer.

## How it satisfies each acceptance criterion

- *Name in the primary store, absent in the linked worktree, commit blocked* —
  the linked-worktree branch (step 2) resolves `primary_banlist`; `load_store`
  loads it into the scan arrays, so the staged name matches and the commit is
  blocked. Test: `hook_test.go` creates a repo, adds the name to the primary
  `.work.local`, `git worktree add`s a linked tree, stages a commit carrying the
  name there, and asserts the hook exits non-zero naming the entry.
- *Freshly created linked worktree with no store of its own enforces the same
  banlist* — the same branch; `store_present` is true via the primary store even
  though the worktree-local file is absent, with no per-worktree setup. Test:
  same fixture without ever writing a worktree-local store; assert the block.
- *A worktree-local entry is never overridden by the primary fallback* — the
  merge order (primary first, local second) makes a duplicate local key win.
  Test: ban a name locally in the worktree, assert the commit is blocked there
  even if the primary store lacks it; assert a name present in both resolves to
  the local entry's metadata.
- *Standalone checkout with no resolvable primary store keeps today's loud
  warning, not a fail-closed error* — when `git_dir == common_dir` and no local
  store exists, the hook takes the existing warning branch (lines 300–311). Test:
  the existing `TestPreCommitHook_AbsentBanlistWarnsLoudly` and
  `TestPreCommitHook_EntrylessStoreWarnsLoudly` continue to pass unchanged.

## Decisions

The candidate remedies in the intent were (a) seed a pointer to the primary
store at worktree-creation time, or (b) have the hook fall back to reading the
primary store. Remedy (b) is chosen: worktree creation is not a path abcd owns
(a developer runs `git worktree add` directly), so a creation-time seed would be
bypassed exactly when it matters; resolving at enforcement time closes the gap
for every worktree unconditionally. The reach caveat that CI cannot see the
private layer (`PrivateReachNote`, `banlist.go:90–92`) is unchanged — this is a
local-enforcement fix, not a CI one.
