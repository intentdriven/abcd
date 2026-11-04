---
schema_version: 1
id: "iss-2611040337254654"
slug: "the-go-build-cache-reached-about-323-gb"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "the v0.17.0 cut and the overnight lanes, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/load_wait.go"
remedy: "Have the load gate read free disk at the start of a run, and trim the build cache by age past a declared floor, saying what it removed; a test sets a low floor with a fixture cache."
refines: [iss-2610290717018523]
wontfix_reason: "Duplicate: the open record already holds this cache event."
duplicates: [iss-2610290717018523]
---

The Go build cache reached about 323 GB on 2026-11-03 with every entry used within two days, because many worktrees and shards each build their own variants; nothing bounds it and the disk reached 5 GiB free on 2026-11-02, which an unattended run would not survive.

## Grounds

- declined: Duplicate: the open record already holds this cache event.
