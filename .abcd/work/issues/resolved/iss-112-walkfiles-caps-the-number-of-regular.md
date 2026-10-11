---
schema_version: 1
id: "iss-112"
slug: "walkfiles-caps-the-number-of-regular"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "itd-95 P1 review"
found_at: "internal/core/lifeboat/probe.go"
resolution: "Bounded the walk's per-directory read with the shared maxDirEntries guard (readDirBounded), so one directory of millions of entries no longer materialises before the file cap."
impact: fix
---

WalkFiles caps the number of regular files it returns, but fs.WalkDir calls ReadDir(-1) on every directory it enters, so a single directory holding millions of entries balloons memory before the file cap can apply. The maxWalkFiles doc comment claims the walk stays bounded in memory, which is only true of the returned slice. ListDir already bounds this with ReadDir(maxDirEntries); the walk does not share that guard. Found by an independent security review of the itd-95 diff.