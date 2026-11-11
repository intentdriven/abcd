---
schema_version: 1
id: "iss-2611071820538335"
slug: "a-hook-change-on-main-stops-a-running"
severity: "critical"
category: "bug"
source: "user-observation"
found_during: "first live abcd drain --drive"
origin: researcher-authored
production_mode: hand-written
found_at: "the drive's configuration fingerprint (config-fingerprint.json)"
remedy: "Pin a lane's hooks when the lane opens, from the reviewed default branch it opens on, and refuse only a hook change made inside the lane; a hook change between lanes is the default branch's, which review and the queue already passed."
resolution: "A lane's hooks are pinned as it opens from the commit it opens on, which origin itself reports as its default branch's tip, after its hooks on disk are proved to be that commit's; a hook change merged on the default branch is no change in later lanes, and a hook changed in a lane is still refused."
impact: fix
resolved_by:
  commit: "64a5e44b38ecc60d7f96985a6887c7a837b8d96f"
---

A running abcd drain --drive takes no git step in any lane once a hook file on the default branch changes. It pins the hashes of the repository's hook files when it starts and compares each lane's worktree, opened from the moving default branch, against them, so a reviewed hook change merged during the run stops every later lane: the first live drive waited in fifteen-minute steps from its second lane on, after a change to the commit hooks merged.
