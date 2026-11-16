---
id: adr-2611151105306366
slug: the-drive-s-own-preflight-runs-in-a
status: proposed
date: 2026-11-15
supersedes: null
superseded_by: null
related_intents: []
related_rfcs: []
related_adrs: []
---

# ADR-2611151105306366: The drive's own preflight runs in a fresh checkout of the lane's stored head

## Context

Decided by the run for owner review.

`abcd drain --drive` lands a lane only on a preflight receipt that its own preflight minted. Until now the drive ran that preflight in the lane's worktree, the checkout its stages worked in. Everything in that worktree was written by what ran there, including its index and its ignored files. The stage's worktree is not the drive's to trust, so a preflight that passed there does not show that the lane's stored head passes.

The landing already reads what a lane changed from stored commits, at the repository root. After this decision, the stage's worktree matters to the landing only for the lane's branch.

## Decision

We will run the drive's own preflight in a fresh checkout that the drive makes, not in the stage's worktree. The steps are:

- Just before the preflight, the drive reads its held settings again, in each environment its git runs in. The held settings now also include the settings that change what a checkout writes: core.attributesFile, core.autocrlf, core.eol, core.symlinks, core.fileMode, core.ignoreCase, core.precomposeUnicode, core.checkRoundtripEncoding, core.sparseCheckout, core.sparseCheckoutCone and submodule.recurse. The common git directory's info/attributes was already held. The sparse-checkout patterns file that git names at the checkout (`git rev-parse --git-path info/sparse-checkout`) is now held beside it, because `git worktree add` copies it into the checkout the drive makes. A file that is absent and one that is empty read as different. Any change holds the landing and names the setting or the file.
- The drive adds a detached linked worktree of the lane's head, read as stored (no replace refs, grafts or commit-graph), in a directory with a random name that it makes, at mode 0700, under `~/.abcd.noindex/preflight-checkouts/`. abcd makes that folder at mode 0700 when it is missing; one that already exists keeps its mode. The add goes through the same runner as the lane's own worktree add. The repository's filters and post-checkout hook run as they would on the person's own checkout, and in a drive they run under the stage's wall.
- Before the preflight and again after it, the drive compares the checkout with the head's stored tree by content. It does this through an index it builds itself from the head, in the same environment the checkout was made in, so a content filter the checkout was written through reads it back. The hooks are pinned off for these reads, and the held settings are checked before each one. Sparse checkout is switched off for these reads (`-c core.sparseCheckout=false`). Otherwise a file that the copied patterns leave out would be missing from the checkout, and git 2.34 and later would leave it out of the comparison, so a preflight on a partial tree would be taken as the head's. With sparse checkout off, a missing file is a difference. Before the preflight, the hold then says that the checkout is missing files the head holds, and that sparse checkout is the likely reason. The drive does not use `git add --sparse` for this, because that option arrived in git 2.34 and the drive otherwise runs on git from 2.31.
- Before the preflight, every untracked file outside the checkout's local tier is a difference, ignored ones included.
- After the preflight, an untracked file is a difference unless the head's own `.gitignore` files ignore it or it is in the checkout's local tier. No other ignore source is read. An untracked `.gitignore` or `.gitattributes` is a difference, ignored or not, because either one changes how the checkout is read. The exception is one below a directory that the head's `.gitignore` files ignore as a whole, which holds no tracked file; there it changes how nothing but that directory's own untracked files are read. Any difference holds the landing and marks nothing.
- The preflight runs in that checkout with GOENV=off. When the lane's head tracks a regular `go.work` at its root, read from the head as stored, GOWORK is left unset, so Go's own search, for a Go command started inside the checkout, finds the nearest `go.work` the head tracks and stops at the checkout's root. Otherwise GOWORK is off. GOFLAGS is set explicitly, to empty, when the environment carries none.
- When the preflight exits 0 and the landing is still held after it, the hold stands for that head, and the drive does not run the preflight on it again.
- When the preflight passes, the drive records the head in its own memory as one its preflight passed on. It then copies the receipt the preflight minted into the main checkout's local tier, where the pre-push hook and the landing look for one. The drive removes its checkout once the preflight has ended, whether it passed or not.
- Nothing is copied from the stage's worktree, including the test reuse reports under `.abcd/.work.local/preflight-reuse`.

## Alternatives Considered

- **Compare the stage's worktree with the head, then run the preflight there.** Rejected. Whatever the comparison read there would be read from the stage's worktree, which is not the drive's to trust.
- **Make a checkout with its own git directory** (a clone of the head). Rejected. A clone loses the repository's local configuration, including its filters and large-file setup, as well as the shared refs that the preflight's gates read, such as origin's default branch.
- **A linked worktree of the stored head, made by the drive (chosen).** It shares the repository's configuration, which the drive holds and compares, and its objects and refs. None of its files come from the stage.

## Consequences

- Every lane's preflight in a drive is a full run with no reuse carried over from the stage's worktree. It needs the disk and the time of a fresh checkout and a full test run, once per lane head.
- A preflight may leave build output that the head's `.gitignore` files ignore, including a tool's cache directory that holds its own `.gitignore`. A file ignored only by `.git/info/exclude` or by a global ignore file holds the landing, and the hold names it.
- The receipt for a drive's lane now sits in the main checkout's local tier, not in the lane's worktree.
- A stage can no longer make the preflight read files the head does not hold. What the preflight shares with the stages outside the checkout is outside this decision.
