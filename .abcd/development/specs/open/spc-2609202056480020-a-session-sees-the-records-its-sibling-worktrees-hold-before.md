---
id: spc-2609202056480020
slug: a-session-sees-the-records-its-sibling-worktrees-hold-before
intent: itd-2609091416295622
origin: researcher-authored
production_mode: hand-written
---
# a-session-sees-the-records-its-sibling-worktrees-hold-before

## Summary

The design record for itd-2609091416295622, written from the eight decisions
the product thinker settled on 2026-09-20 (the intent's `## Decisions`).

## Scope

1. **The peer reader** (decision 2), a package under `internal/core/peers`
   with one entry point that returns, for this checkout, the set of live
   peers and each peer's holdings. Two sources behind one interface:
   `worktree` (every linked worktree git lists, confirmed from the
   candidate's own `--git-common-dir` under `gitutil.IsolatedEnv`, read off
   disk) and `branch` (every local branch through the common dir, read with
   `git ls-tree <branch> -- .abcd/work/issues .abcd/development/intents/drafts`).
   A third source, `register`, is an interface slot the coordination intent
   fills; this spec ships it empty and says so. Holdings are ids by filename
   under `open/`, `resolved/`, `wontfix/` and `drafts/`; a title is read
   through `fsutil.ReadGuarded` and the record reader's symlink refusal, and
   a failed title read yields the id alone (criterion 12).
2. **Peer filtering** (decisions 4, 5, 8): a peer is skipped and counted when
   its worktree directory is gone or its branch is an ancestor of the fetched
   default branch (criterion 6); a peer whose ledger holds one id in two
   status folders is returned as a marked peer with no rows, the id and the
   remedy (criterion 7); a peer git refuses to answer for, or whose common
   dir is not this checkout's, is returned as a marked peer with the reason
   (criterion 8).
3. **The diff** (decision 6): per live peer, three rows computed against this
   tree's own holdings read through the canonical ledger reader: open there
   and absent here, open here and terminal there, drafted there and absent
   here (criteria 1 to 4).
4. **The three refusals** (decision 3, criterion 5): `capture resolve`,
   the record dispatcher and `intent audit` consult the reader on their
   not-found path and refuse naming the peer, its branch, its path
   (home-redacted) and the folder; the reader is consulted only after the
   local lookup fails, so a healthy tree pays nothing.
5. **The board line and the command** (decision 3, criteria 9, 10, 11, 13):
   one line on the bare status board when the diff is non-empty; the
   standalone command (`peers` as the working name; the build settles it)
   renders the blocks in text and `--json` with every path through
   `fsutil.RedactHome`; `AGENTS.md`'s concurrent-sessions section names it.

## Out of scope

Any write (the claim intent); the register source's implementation (the
coordination intent); the worktree store's own render (itd-148, which
consumes this reader for its ledger columns).

## Approach

Test-first, in the order above; pieces 1 to 3 are one lane because they
share the fixtures (two worktrees off one temp repository, a third branch
checked out nowhere, a split-ledger peer, a refused peer); pieces 4 and 5
are a second lane. The reader is the one canonical primitive: the refusal
intent's ledger-at-a-ref read is written against the `branch` source
rather than beside it.

## How the criteria are satisfied

1 to 4 by pieces 1 and 3; 5 by piece 4; 6 to 8 by piece 2; 9 to 11 and 13
by piece 5; 12 by piece 1's guarded title read.

