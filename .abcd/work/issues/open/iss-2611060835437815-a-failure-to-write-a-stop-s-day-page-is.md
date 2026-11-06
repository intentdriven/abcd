---
schema_version: 1
id: "iss-2611060835437815"
slug: "a-failure-to-write-a-stop-s-day-page-is"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "v0.16.2 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/stop.go"
remedy: "Tie the failure to the stop it belongs to, and clear it only when that stop's page is written."
---

A failure to write a stop's day page is kept on the run's record and is cleared by the next page written, including a later stop's, so the earlier stop's page is never written and nothing says so (`stop.go:889`). A person looking for what that stop halted finds neither a page nor a failure.
