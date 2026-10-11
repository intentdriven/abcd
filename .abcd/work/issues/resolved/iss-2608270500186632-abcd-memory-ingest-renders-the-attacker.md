---
schema_version: 1
id: "iss-2608270500186632"
slug: "abcd-memory-ingest-renders-the-attacker"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "github-ledger-dedup-2026-08-27"
found_at: "internal/core/memory/ingest.go"
resolution: "memory ingest sanitises the licence field before rendering (#262)"
impact: fix
---

abcd memory ingest renders the attacker-derived IngestResult.Licence field to the terminal without sanitisation. No sibling record covers the memory-ingest licence render site. GitHub mirror: #262