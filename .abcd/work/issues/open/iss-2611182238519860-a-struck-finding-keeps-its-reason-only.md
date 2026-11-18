---
schema_version: 1
id: "iss-2611182238519860"
slug: "a-struck-finding-keeps-its-reason-only"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "autonomous run of 15 Nov 2026, review of the review set's step 5"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/findings.go"
remedy: "Keep each struck location and its reason on the round's judge record, which the round keeps after its findings are dropped, and print them in the plain-text run record beside the judge's line; a test strikes a finding, runs a fix round, and finds the location and reason in both the JSON and plain record."
---

A struck finding keeps its reason only while the judged round is the latest: once the fix round opens the next round, the older round's findings are dropped, so a finished run's record shows the judge ruled and how many it struck but not which findings or why, and the plain-text run record never shows a strike's reason.
