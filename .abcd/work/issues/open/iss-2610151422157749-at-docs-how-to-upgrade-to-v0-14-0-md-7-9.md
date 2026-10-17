---
schema_version: 1
id: "iss-2610151422157749"
slug: "at-docs-how-to-upgrade-to-v0-14-0-md-7-9"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "v0.14.2 release review, docs-currency pair, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/how-to/upgrade-to-v0.14.0.md"
remedy: "Correct docs/how-to/upgrade-to-v0.14.0.md:7-9,32-35 so it states what the shipped binary does, as the finding describes."
---

At docs/how-to/upgrade-to-v0.14.0.md:7-9,32-35: The guide says a script that runs an old command name 'gets a refusal that names what replaced it'. The new paragraph covers bare `abcd memory` now running, but on this binary `abcd memory ingest`, `ask` and `lint` refuse as a plain cobra unknown sub-verb (`abcd: unknown command "ingest" for "abcd memory"`, exit 2) with no pointer to `library`. Verified: `abcd source` still names its successor. The paragraph also says only 'a release after v0.14.0' and does not name v0.14.2. / The note says 'A release after v0.14.0' reuses the name memory without naming the release, while the table above still lists `abcd memory` as retired. Verified true of this binary (`abcd memory` answers with a count, `abcd memory ingest` is still unknown command, exit 2). Naming v0.14.2 would be exact. (v0.14.2 release docs-currency review, raised by opus+sonnet.)
