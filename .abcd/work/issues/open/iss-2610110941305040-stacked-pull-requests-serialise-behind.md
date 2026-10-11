---
schema_version: 1
id: "iss-2610110941305040"
slug: "stacked-pull-requests-serialise-behind"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "merge-pipeline evidence from abcd-0b, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/work/rulesets/main-protection.json"
remedy: "measure how long a stack's last step waits behind its lower steps, and weigh landing a stack as one pull request, or queueing a step only once the step below it has merged, against the review cost of one large diff"
---

Stacked pull requests serialise behind the merge queue: verb-split steps 2 to 7 are #913, #918, #919, #920, #921 and #922, each carrying the previous step's commits, so none can merge before the one below it and every queue removal of a lower step delays the whole stack. On 2026-10-11 #913, the stack's base, was itself removed from the queue at the macOS timeout.
