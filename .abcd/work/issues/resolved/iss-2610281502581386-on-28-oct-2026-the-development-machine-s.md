---
schema_version: 1
id: "iss-2610281502581386"
slug: "on-28-oct-2026-the-development-machine-s"
severity: "major"
category: "bug"
source: "agent-observation"
found_during: "autonomous-run-2026-10-28"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/machineload"
remedy: "Have the load gate and preflight's load-check measure free disk on the volume holding GOCACHE and the worktree store, and refuse to admit a test stage below a floor, naming the cache size and an age-based trim; and have the worktree store's reclaim trim cache entries older than a day when it removes worktrees."
resolution: "The load gate refuses the two stages it gates, make preflight's test stage and its race stage (and any command run through implement load --wait), at either gate and without waiting, while the volume holding the Go build cache or the worktree store has less free space than the floor: 4 GiB for each preflight the gate admits at once (load.preflights), and 8 GiB at least. It reads the disk as the waiter starts and again just before it opens. The refusal names each low volume, its free space and the floor; when the cache's own volume is the low one it also names the cache's path and estimated size, a trim of build entries unused for two days that keeps the fuzz corpora, and go clean -cache as the all-or-nothing alternative. A disk that cannot be read, in whole or in part, is reported and never refused. Preflight's load check prints the same finding as a warning and never fails; the stages preflight runs before its first gate are not checked. Trimming the cache when the worktree store reclaims worktrees is left to iss-2610290717018523."
impact: fix
---

On 28 Oct 2026 the development machine's data volume fell to 658 MiB free of 926 GiB and a peer's verify failed with No space left on device. The Go build cache had grown to 177 GiB: every store worktree, race build and pinned-toolchain build adds entries, and Go trims only entries unused for five days, so days of many-worktree autonomous runs fill the disk, and every preflight on the machine then fails or writes truncated files with no gate naming the cause. Entries unused for two days (79 GiB) were deleted by hand.
