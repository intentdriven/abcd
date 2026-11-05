---
schema_version: 1
id: "iss-2611051720178630"
slug: "reserve-admit-can-say-a-pull-request"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "build run, 5 November"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/withdrawact.go"
remedy: "Keep what the first read found and report that instead of the failed second read."
---

`abcd reserve --admit` can say a pull request stays withdrawn when inspect had just found it armed, if the re-read fails between two reads in one act.
