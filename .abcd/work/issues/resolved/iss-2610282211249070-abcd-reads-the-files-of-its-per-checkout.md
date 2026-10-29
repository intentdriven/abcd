---
schema_version: 1
id: "iss-2610282211249070"
slug: "abcd-reads-the-files-of-its-per-checkout"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "the local tier's state readers"
remedy: "Refuse a local-tier state read of a path git tracks (failing closed when git cannot answer), through one helper every such reader calls, and add a lint rule refusing any tracked file under .abcd/.work.local/."
resolution: "abcd now reads a checkout's local-tier state only when git tracks none of it, and record-lint refuses a change that tracks a file there."
impact: fix
---

abcd reads the files of its per-checkout local tier (.abcd/.work.local/) without checking that git does not track them. The tier is this checkout's alone and is ignored by design, so a copy of one of its files that a commit carries is not this checkout's state, yet it is read as if it were.
