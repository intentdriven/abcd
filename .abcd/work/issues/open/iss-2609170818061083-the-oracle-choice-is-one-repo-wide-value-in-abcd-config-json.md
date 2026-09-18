---
schema_version: 1
id: "iss-2609170818061083"
slug: "the-oracle-choice-is-one-repo-wide-value-in-abcd-config-json"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "reviewing the oracle seam after the v0.9.0 install, 2026-09-17"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/config.json"
promoted_to: itd-2609170822093401
---

The oracle choice is one repo-wide value in .abcd/config.json (host-delegated | native | cli | api | mcp), so every delegated step reaches a model the same way. The choice should be configurable per task: some steps want a local model (a cheap, private first pass over a transcript), some want the harness to decide, some want the harness told to use a high-end frontier model (a release gate's semantic review, an intent audit) and some a cheaper one (a slug, a summary). Alongside the backend, the same per-task configuration should say whether the harness may use sub-agents for the step at all, and how many, so a fan-out review can be bounded and a single-pass composition kept to one. Today none of this is expressible: the backend is global, and sub-agent use is whatever the command page tells the host in prose. The task classes the agent trust contract already declares (capability_scope.task_classes) are the natural key; the api adapter draft itd-2609081951381895 would be the first backend such a table could route to.

## Grounds

- pursued: flexibility first — each delegated step routed to the model it deserves, with privacy as an option (a local model where the material must not leave the machine) and cost as the essential lever: cheap steps to cheap or local models, the judgement-bearing steps (a release gate's semantic review, an intent audit) to a frontier model, and sub-agent fan-out declared and bounded per step, because unbounded fan-out is where cost and unpredictability come from. Shown wrong if the locally-routed or bounded steps start failing the binary's schema gates or their verdicts drift from the frontier-routed ones, or if bounded runs cost the same and vary as much as unbounded ones.
