---
schema_version: 1
id: "iss-2611060835357147"
slug: "settlegaveup-probes-for-up-to-three"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "v0.16.2 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/terminal.go"
remedy: "Compare StartedAt and Down under the lock before clearing, and return early on a service it does not know."
---

`settleGaveUp` probes for up to three minutes and then clears the outage record without checking it is still the one it probed (`terminal.go:1062`, `:1081`; `internal/core/implement/outage.go:671`). Another session may have reported a further service meanwhile, and the clear removes that too with an untrue reason, which corrects itself after at most one failed launch.
