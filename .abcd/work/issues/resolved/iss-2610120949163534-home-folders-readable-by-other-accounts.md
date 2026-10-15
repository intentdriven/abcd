---
schema_version: 1
id: "iss-2610120949163534"
slug: "home-folders-readable-by-other-accounts"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "security review of the embargo store, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/abcdhome/abcdhome.go"
remedy: "Narrow an existing ~/.abcd.noindex the caller owns, that is a real directory and carries any group or other bit, to 0700 at the CLI front door (never following a symlink, never touching a folder another account owns, never walking the tree), and make a new lane worktree's root 0700 after git creates it; prove with tests, watched failing first, that a fresh home is 0700, an existing 0755 home is narrowed, a symlinked home is left alone, and a new worktree-store entry and runs folder are 0700"
resolution: "abcd narrows an existing home it owns to the account's alone at the front door before any verb runs, never through a symlink or for another account's folder, and narrows each lane worktree git adds; new homes and store folders were already made 0700"
impact: fix
resolved_by:
  commit: "c592d5d44"
---

Folders in abcd's home that an earlier abcd, a hand-run `mkdir`, or `git worktree add` created at the process umask are readable by every other account on the computer whenever the home itself is too: a home made 0755 keeps that mode, because every abcd writer leaves an existing folder's mode as it stands ([dcn-2610040000000007](../../decisions/dcn-2610040000000007-decided-without-a-question-by-abcd-50.md)). On a Mac where `~` is 0750 group staff and every local account is in staff, the other accounts can list and read `~/.abcd.noindex/{worktrees,runs,history,lifeboats,review-copies,scratch,sources,lab}` (each 0755), which hold private checkouts and session material. A lane worktree git adds into the store is 0755 whatever the store's mode.

## Grounds

- pursued: TestTheFrontDoorNarrowsAnOpenHome, TestTheFrontDoorLeavesASymlinkedHome, TestNarrowToOwner* (foreign owner, link, file) and TestTheWorktreeStepMakesTheLaneWorktreeTheCallersAlone; a 0755 home or lane worktree left open, or a symlink target re-moded, would show it wrong
