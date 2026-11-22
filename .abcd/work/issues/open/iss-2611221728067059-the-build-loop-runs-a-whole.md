---
schema_version: 1
id: "iss-2611221728067059"
slug: "the-build-loop-runs-a-whole"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop"
remedy: "Decide whether a round is sized to its delta: for a delta holding only records and prose, or the loop's own record commit, run the record checks and fewer reviewers, or keep the whole round and say why the merge head needs it."
---

the build loop runs a whole four-reviewer round on a head whose change since the last passed round holds no code: on 22 November one issue lane had a round for a sync merge that brought 21 record and prose files and a round for the loop's own 16-line record commit, each two research-class and two standard reads
