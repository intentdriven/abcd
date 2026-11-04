---
schema_version: 1
id: "iss-2611040554494843"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/22-site.md"
remedy: "Name record.decisions beside the issue-ledger switch and in the export, and have the bare board report it."
---

.abcd/development/brief/04-surfaces/22-site.md:256-261 says one input reaches past the durable record, the issue-ledger switch, and .abcd/development/brief/04-surfaces/22-site.md:340-348 enumerates the record export without decisions. record.decisions (internal/core/site/manifest.go:313, set in .abcd/site.json) publishes .abcd/work/decisions/ as a decision log and a page per record, and record.json carries a `decisions` key. Its only brief mention is .abcd/development/brief/05-internals/10-site.md:105. Bare `abcd site` reports only the issue ledger.
