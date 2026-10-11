---
schema_version: 1
id: "iss-101"
slug: "history-index-lost-update-concurrent"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "multi-agent-bughunt"
resolution: "History-index registration now serializes on an inter-process lock in ~/.abcd/history with a re-load inside the lock, and bootstrap seeds index.json via an atomic exclusive hard link; the re-founding prompt is asked before the lock and its state re-checked under it."
impact: fix
---

registerRepo does an unlocked load-modify-write of ~/.abcd/history/index.json (loadHistoryIndex -> mutate -> writeHistoryIndex, apply.go), and bootstrapHistory is check-then-act. WriteFileAtomic prevents torn files but not lost updates (last rename wins), so two concurrent 'abcd ahoy install' runs from different worktrees can drop a repo registration or clobber the registry with an empty Repos list, erasing lineage/supersedes links set on explicit user confirmation. Deferred from the multi-agent bug hunt (was B30): a correct fix needs an inter-process O_EXCL/flock lock in ~/.abcd/history held across the load-mutate-write, with a re-load inside the lock and bootstrapHistory creating index.json with O_EXCL. The registerRepo sequence spans an interactive prompter.Confirm (re-founding lineage), so the lock must NOT be held across that prompt (risk of blocking other processes on user input). Low severity, self-heals on next install. Needs a dedicated change.