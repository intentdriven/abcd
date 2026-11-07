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

A failure to write a stop's day page is kept on the run's control and leaves a failure line on its record. The next page written clears it, including a later stop's, and adds the line `the day's page was written` (`stop.go:889`, `stop.go:915`), so the earlier stop's page is never written and the record's latest line reads as though it was. A person looking for what that stop halted finds no page, and a failure line followed by one that seems to answer it.
