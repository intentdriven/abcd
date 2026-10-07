---
schema_version: 1
id: "iss-2610050348402910"
slug: "an-ai-written-interview-s-role-can-redirect-where-the-change"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25: lane e8term-r3b, sweeping the guard for directories it reads but does not own"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/interview/treewatch.go"
remedy: "Pin the git directory and the common directory the run's first reading resolves (git rev-parse --git-dir --git-common-dir), read those same directories in every later reading of the run, and watch the git directory's own commondir file beside a linked worktree's .git file (already watched), so a redirection is named as a change to the pointer without reading what it points at. Grounds: git honours a commondir file in any git directory (the scratch probe above), readGitDirs re-resolves both directories on every reading (treewatch.go), and the pre-turn reading is the one taken before the role could write."
resolution: "The run's first reading pins git's own directory and common directory and the state of the two files that place them (a linked worktree's .git file, the git directory's commondir); every later reading reads those files before git runs and names one changed as the turn's change, unfollowed, and refuses when git names other directories than the pinned ones (TestARedirectedGitDirectoryIsNamedNotRead)."
impact: fix
resolved_by:
  spec: "spc-2610040847280931"
  commit: "2731f41f1"
---

An AI-written interview's role can redirect where the change guard reads git's own directory, and slow the reading after its turn: a commondir file written into a main checkout's .git (or a linked worktree's commondir or .git file rewritten) makes git resolve the common directory elsewhere, and the next reading then hashes that directory's hooks, info, configuration, refs, modules and worktrees whole, with no budget, before the change is reported. A probe on a scratch repository: .git/commondir naming a directory holding only objects/ and refs/ made git rev-parse --git-common-dir name it, and git status still ran. The guard fails closed (every key of the person's own git directory disappears from the reading, so the turn is stopped), but a refs file there of many GiB holds the turn for minutes first. The followed-link and hooks-directory budget (iss-2610050315146044) does not cover it: git's own directory is read as owned, and its legitimate size (loose refs) is not one the 1,024-entry budget can hold.

## Grounds

- pursued: a commondir or .git file a role rewrites during a turn is named promptly and never read through; a turn that held for git reading the redirected directory, or a redirect that reached the hashing, would show it wrong
