---
schema_version: 1
id: "iss-2610252214152975"
slug: "commands-intent-md-232-says-the-grounds"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S02"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/intent.md"
remedy: "Say grounds is the eighth row and open_questions the ninth and last, which gates."
---

commands/intent.md:232 says the grounds check is 'the eighth and last row of the report', but abcd intent ready appends a ninth, gating open_questions row last (internal/core/intent/ready.go:153).
