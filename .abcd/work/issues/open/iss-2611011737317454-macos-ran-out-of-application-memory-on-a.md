---
schema_version: 1
id: "iss-2611011737317454"
slug: "macos-ran-out-of-application-memory-on-a"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "parallel drain and build sessions, 2026-11-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/machineload/machineload.go"
remedy: "Give machineload's Snapshot a current memory reading (on macOS the memory pressure level and free plus compressible pages, never the cumulative swap counter) and hold the load gate below a memory floor in the shape of the disk floor in internal/core/implement/load_disk.go, keeping the gate's bounded wait; have the gate sample each admitted run's peak resident size so the next peak is attributed before any go test -p is sized from memory. The derived admission counts stay with iss-2610161756384596 and per-stage recording with itd-2610262140595217."
---

macOS ran out of application memory on a 128 GB, 16-core machine on 2026-11-01 at about 17:32, with only this person's sessions active, while several agent sessions drained, built and reviewed in parallel. The Force Quit dialog lists only GUI apps (Ghostty at 8.93 GB, falling to 937 MB afterwards), so the go test processes, race-enabled test binaries and agent sessions running inside the terminal never show. The machine's counters show about 155 GiB swapped out since it booted five days earlier, so this was not the first time it ran past its RAM. The system log needs admin rights to read, so the peak was not attributed to one process; a read shortly after, with 22 test processes live at a load of 57, found the test workload at about 4 GiB and the agent sessions at about 3.8 GiB, so the consumer is unproven.

What abcd can say is that its load gate is blind to memory: machineload's Snapshot holds load averages, cores and the process table but no memory reading, and cmd/test-parallelism derives go test -p from cores and load alone. make preflight gates itself, and an agent's own test run is gated only when it uses the command form (abcd implement load --wait -- go test ...), which the LOAD rules teach but nothing enforces (iss-2610241130157455 settled that it be taught, not enforced); either way no gate checks free memory, as it already checks free disk. iss-2610161756384596 asks for free memory as an input to how many runs are admitted; this record is the first measured outage, and asks for the floor and the attribution that would come before that.
