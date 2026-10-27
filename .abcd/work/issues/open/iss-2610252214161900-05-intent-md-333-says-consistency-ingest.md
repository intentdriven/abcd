---
schema_version: 1
id: "iss-2610252214161900"
slug: "05-intent-md-333-says-consistency-ingest"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S02"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md"
remedy: "State the key, addressed and quote precedence, or point at commands/intent.md § Consistency."
---

05-intent.md:333 says consistency ingest files a finding unless an open record quotes an end; the code checks the finding_ref key first and marks a finding held by a resolved or wontfix record 'addressed', filing nothing (internal/core/intent/consistency.go:891-896, commands/intent.md:1277). 05-internals/01-agents.md:168 has the same gap.
