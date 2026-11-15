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

- Just before the preflight, the drive reads its held settings again, in each environment its git runs in. The held settings now also include the settings that change what a checkout writes: core.attributesFile, core.autocrlf, core.eol, core.symlinks, core.fileMode, core.ignoreCase, core.precomposeUnicode, core.checkRoundtripEncoding, core.sparseCheckout, core.sparseCheckoutCone and submodule.recurse. The common git directory's info/attributes was already held. Any change holds the landing and names the setting.
- The drive adds a detached linked worktree of the lane's head, read as stored (no replace refs or grafts), in a temporary directory it makes. The add goes through the same runner as the lane's own worktree add. The repository's filters and post-checkout hook run as they would on the person's own checkout, and in a drive they run under the stage's wall.
- Before the preflight and again after it, the drive compares the checkout with the head's stored tree by content. It does this through an index it builds itself from the head. It also lists every untracked file, ignored ones included, and allows only the paths the preflight is known to write, which is the checkout's local tier. Any difference holds the landing and marks nothing.
- The preflight runs in that checkout with GOWORK=off and GOENV=off. GOFLAGS is set explicitly, to empty, when the environment carries none.
- When the preflight passes, the drive records the head in its own memory as one its preflight passed on. It then copies the receipt the preflight minted into the main checkout's local tier, where the pre-push hook and the landing look for one. The drive removes its checkout once the preflight has ended, whether it passed or not.
- Nothing is copied from the stage's worktree, including the test reuse reports under `.abcd/.work.local/preflight-reuse`.

## Alternatives Considered

- **Compare the stage's worktree with the head, then run the preflight there.** Rejected. The comparison has to read the worktree through its index and its ignore rules, and both belong to the stage. A stage can defeat it.
- **Make a checkout with its own git directory** (a clone of the head). Rejected. A clone loses the repository's local configuration, including its filters and large-file setup, as well as the shared refs that the preflight's gates read, such as origin's default branch.
- **A linked worktree of the stored head, made by the drive (chosen).** It shares the repository's configuration, which the drive holds and compares, and its objects and refs. None of its files come from the stage.

## Consequences

- Every lane's preflight in a drive is a full run with no reuse carried over from the stage's worktree. It needs the disk and the time of a fresh checkout and a full test run, once per lane head.
- A repository whose filters change content and are defined only in a global configuration that the drive's comparison does not read may hold every lane. The hold names the file that differs.
- The receipt for a drive's lane now sits in the main checkout's local tier, not in the lane's worktree.
- A stage can no longer make the preflight read files the head does not hold. What the preflight shares with the stages outside the checkout is outside this decision.
