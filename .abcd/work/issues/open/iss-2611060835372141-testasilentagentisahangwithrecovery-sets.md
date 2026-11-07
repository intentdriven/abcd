---
schema_version: 1
id: "iss-2611060835372141"
slug: "testasilentagentisahangwithrecovery-sets"
severity: "minor"
category: "tech-debt"
source: "agent-observation"
found_during: "v0.16.2 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/terminal_longrun_test.go"
remedy: "Apply the short limit to the hanging launch alone."
---

TestASilentAgentIsAHangWithRecovery sets a 5-second runner limit that also covers the ordinary launches after the hang (`terminal_longrun_test.go:790`), so a loaded machine could kill one of those and fail the check for exactly one suspect file. It has not been seen failing; a failure would cost a queue entry.
