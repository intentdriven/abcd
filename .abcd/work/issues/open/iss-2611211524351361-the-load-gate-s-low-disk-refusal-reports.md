---
schema_version: 1
id: "iss-2611211524351361"
slug: "the-load-gate-s-low-disk-refusal-reports"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "autonomous run, disk floor refusal on 21 Nov"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/load_disk.go"
remedy: "On macOS, have the refusal and the load check's disk warning name the figure they read (free now, not counting purgeable space) and say that Finder's available figure includes purgeable space macOS frees only on demand, keeping the floor on free blocks"
---

The load gate's low-disk refusal reports the volume's free blocks (37 GiB here) while macOS Finder shows the same volume with hundreds of GB available, because Finder counts purgeable space (local snapshots, cached cloud copies) that macOS frees only on demand. The refusal never says which figure it reads, so a person comparing it with Finder reads the gate as wrong. The floor counts free blocks on purpose: on 28 Oct the machine reached 658 MiB free and every preflight failed even with purgeable space present.
