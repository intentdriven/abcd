---
id: dcn-2611210325292707
slug: a-running-interview-s-tripwire-reads-no
date: 2026-11-21
---
2026-11-21 — A running interview's tripwire reads no sibling worktree's HEAD and no push receipt anywhere, so a peer switching branch, rebasing, adding or removing a worktree, or running the preflight in its own worktree no longer stops it (iss-2611012144422634). Of the common directory's worktrees/ it reads its own checkout's entry whole, and of a sibling's entry only commondir, gitdir and config.worktree, compared while the entry stands in both readings: those place or carry the configuration and hooks git runs there, the code-running route iss-2610040739124513 closed, and a peer rarely writes them. In a linked checkout the main worktree's HEAD is a sibling's and is not read. Push receipts are left out in this checkout's local tier and every other: the interview's contract reads none, and scripts/preflight-receipt.sh states a receipt is a local convenience gate CI stands behind, not a security boundary, so a receipt a role plants is no longer caught, which loses no protection CI does not hold. This narrows dcn-2611210303400310, which kept sibling entries and receipts watched.
