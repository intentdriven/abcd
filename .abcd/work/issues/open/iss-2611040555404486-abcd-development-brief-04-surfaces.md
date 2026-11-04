---
schema_version: 1
id: "iss-2611040555404486"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/29-report.md"
remedy: "Say the row is in the facilitator view and the JSON form."
---

.abcd/development/brief/04-surfaces/29-report.md:118 says the bare `/abcd` board carries the count as its `inbox:` row. Bare `abcd` and `abcd --format markdown` draw no such row with a report waiting; `abcd --view facilitator` does, and --json carries the `inbox` object. commands/abcd.md:135-137 has the same gap.
