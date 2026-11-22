---
schema_version: 1
id: "iss-2611221058573402"
slug: "after-a-reboot-abcd-drain-s-automatic"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "crash clean-up after the 22 Nov reboot"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/autosweep.go:45"
remedy: "Let the sweep's git reads allow for a loaded machine: budget the worktree listing apart from the fetch (or scale the deadline with load), and when the listing still times out, keep the affected branches with a named reason for that run rather than refusing the local and origin branch halves outright."
---

After a reboot, abcd drain's automatic sweep failed twice with 'git worktree list --porcelain -z: context deadline exceeded' while the machine's load average was about 170. The whole automatic sweep runs under one two-minute budget (autoSweepTimeout), which its fetch of origin and of the reservations spends first; the worktree listing then runs under what is left, inside its own two-minute gitReadTimeout (internal/core/sweep/branches.go:42), from newBranchJudge (branches.go:132) and remoteJudge.readLocal (remote.go:288). Neither bound allows for a system under boot load, and a listing that misses the deadline refuses the whole local-branch half and the whole origin-branch half of the sweep, not just the read.
