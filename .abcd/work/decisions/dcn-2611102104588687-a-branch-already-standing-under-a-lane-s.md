---
id: dcn-2611102104588687
slug: a-branch-already-standing-under-a-lane-s
date: 2026-11-10
---
2026-11-10 — A branch already standing under a lane's name is checked out by the worktree stage only where origin was asked and the branch head's hooks directory, read as stored, is the hooks directory of the tip origin reported; otherwise the stage refuses before git adds the worktree, so nothing of the branch is checked out and none of its hooks runs. Grounds: git's add runs the post-checkout hook the checked-out commit holds, with the drive's rights, before the lane's hooks can be proved, and a branch under the lane's name can be made by anything that writes refs in the shared repository. A checkout whose hooks are not a working tree's own copy runs none of the branch's files as a hook and is not held. To reverse: drop standingBranchHooks from the worktree stage, and the branch is checked out and proved afterwards as before. Refs: iss-2611071820538335
