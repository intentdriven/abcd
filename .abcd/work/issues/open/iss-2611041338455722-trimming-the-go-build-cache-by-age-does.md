---
schema_version: 1
id: "iss-2611041338455722"
slug: "trimming-the-go-build-cache-by-age-does"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "review of the Go cache trim, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/worktree/cachetrim.go"
remedy: "Give the trim a free-space or size target that removes oldest entries first until it is met, and name how it behaves when nothing old enough remains."
refines: [iss-2610290717018523]
---

Trimming the Go build cache by age does not bound it: on 2026-11-03 the cache reached 323 GB with every entry used within two days, so a prune's trim frees nothing on a machine that busy, and the load gate still stops work on a full disk.
