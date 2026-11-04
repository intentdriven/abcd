---
schema_version: 1
id: "iss-2611040712338963"
slug: "parallel-stages-share-one-session"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "the v0.16.1 cut, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/release-gate/brief-surface-crosscheck.js"
remedy: "Give each stage its own directory in its brief, and have the release gate's hand-run procedure require it."
---

Parallel stages share one session scratchpad with nothing to keep them apart: on 2026-11-04 a release-gate checker reused names another checker was using, committing and installing into that checker's test repository for three minutes, so findings from that window had to be rechecked.
