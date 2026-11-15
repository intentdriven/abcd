---
schema_version: 1
id: "iss-2611141812055152"
slug: "the-git-settings-a-drive-holds-across"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/gitenvs.go"
remedy: "judge each such setting: pinned on the command line, held, or shown unread by any command the tree runs, with the table kept beside the held set"
resolution: "A drive holds, beside the settings the run's fingerprint watches, the settings a git command of the drive can read that name or start a program, or change where a networked git goes, whom it trusts or what it sends there (driveHeldConfig in internal/core/implement/loop/remoteid.go), and lists those no git command of the drive reads with the reason (driveUnreadSettings); where git reads a worktree's own settings, as git decides it, it reads them at each lane worktree's own scope too, a lane it makes or adopts as the worktree stage takes it; every fetch through the networked runners goes into no submodule, and the git commands of the drive that compare a working tree with the index start nothing inside one; a read-only merge pins merge.renormalize off; and a log, show, reflog, rev-list, shortlog or diff-tree asks the drive's check when its last format word can show a signature check, or, for log and show, when it has none."
impact: fix
---

The git settings a drive holds across its lanes are not yet judged one by one against git's documentation of the settings that name a program
