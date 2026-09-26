---
schema_version: 1
id: "iss-2609261004261615"
slug: "lab-preflight-isolation-hooks-reads-core-hookspath-raw-with"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-lab"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lab/preflight.go"
---

lab preflight isolation.hooks reads core.hooksPath raw with git config --get and judges the unexpanded text: a %(prefix)/... value (git expands it to its install prefix) is treated as snapshot-relative and passes as resolving inside the lab, a false PASS on the operator-state route; the ~user branch is a hand-rolled special case of the same gap. Judge the path git itself expands (git config --type=path --get) and refuse any expanded path outside the lab.
