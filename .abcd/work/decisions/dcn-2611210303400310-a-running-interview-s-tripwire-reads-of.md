---
id: dcn-2611210303400310
slug: a-running-interview-s-tripwire-reads-of
date: 2026-11-21
---
2026-11-21 — A running interview's tripwire reads, of the shared refs, only the branch its own checkout's HEAD names, by the value that branch holds (its loose ref, else its packed-refs line), so a peer's commit, record branch or ref packing in another worktree no longer stops it (iss-2611012144422634). A ref or tag a role plants beside that branch is no longer caught: it moves neither the next commit nor a default push, and catching it would mean watching every peer's refs again. Submodule refs, the main worktree's HEAD and every sibling worktree's entry and push receipts stay watched as before. A change to the own branch stops the interview with the branch, its old and new commit named in plain words; offering to continue past it needs the terminal front door and is left to a follow-up.
