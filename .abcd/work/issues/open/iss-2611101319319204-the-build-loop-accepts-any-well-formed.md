---
schema_version: 1
id: "iss-2611101319319204"
slug: "the-build-loop-accepts-any-well-formed"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/validate.go"
remedy: "Waits on a ruling first: then have a validator return state the files it read, and have the loop refuse a return naming fewer of the diff's files than the chosen floor."
---

The build loop accepts any well-formed validator return whatever it read, so a shallow review passes as a full one. On 7 Nov at about 20:40, on a lane of about 800 lines across 23 files, both Sonnet validators returned clean after 7 tool calls in about 30 seconds each, one of them skimming the tests, while the Opus pair took more than 16 calls each. A person must choose whether a return must state what it read, and what floor the loop holds it to.
