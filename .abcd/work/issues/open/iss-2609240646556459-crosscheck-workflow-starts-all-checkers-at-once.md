---
schema_version: 1
id: "iss-2609240646556459"
slug: "crosscheck-workflow-starts-all-checkers-at-once"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/release-gate/brief-surface-crosscheck.js"
---

The release gate's brief-surface cross-check workflow (.abcd/development/release-gate/brief-surface-crosscheck.js) starts every checker at once: it maps each brief doc and each surface to an agent and awaits them all in one parallel() call, forty agents at v0.10.0, with no bound the script sets, so the concurrency is whatever the harness's pool gives it. A run under an agent ceiling cannot use it unmodified. The v0.10.0 cut, under a ceiling of four, ran the forty pinned prompts by hand four at a time and re-implemented the script's merge in a local script (the dedup key of `where` plus the first sixty characters of `claim`) to produce the merged findings the receipt records. The receipt's manifest hash pins the prompts and not the concurrency, so the gate's result does not depend on the pool. Wanted: the script takes a concurrency bound from the manifest or the invocation, and the merge is a function a hand-run can call rather than re-type.
