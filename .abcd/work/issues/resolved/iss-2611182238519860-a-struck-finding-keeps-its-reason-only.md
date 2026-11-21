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
resolution: "the round's judge keeps each location it struck with its reason (state schema 29), so a finished run's record shows every strike's reason in its JSON and its text"
impact: fix
---

A struck finding keeps its reason only while the judged round is the latest: once the fix round opens the next round, the older round's findings are dropped, and a strike's reason goes with them. A finished run's record keeps the judge's receipt line naming each struck location, but that line says each reason is on the finding, which the state no longer holds, and the plain-text run record never shows a strike's reason.

## Grounds

- pursued: a strike's reason survives the next round; a record read after a fix round opens another that shows no reason for a struck location would show it wrong
