---
schema_version: 1
id: "iss-56"
slug: "iss-36-lists-abcd-logbook-as-a-retired"
severity: "major"
category: "drift"
source: "agent-finding"
found_during: "autonomous-run"
found_at: ".abcd/record-lint.json"
blocked_by: [iss-36]
resolution: "the shipped binary no longer writes .abcd/logbook anywhere; scanner retired_location_test.go asserts the retirement, and the memory lint destination moved"
impact: internal
---

iss-36 lists .abcd/logbook as a retired location, but the shipped binary writes there: memory lint reports go to .abcd/logbook/memory/ (internal/core/memory/lint.go) and the scanner skip-list references .abcd/logbook/pii-scan/. Adjudicate before any logbook ban is armed; the logbook token was deliberately left OUT of record-lint banned_tokens.