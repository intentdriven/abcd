---
schema_version: 1
id: "iss-2611150658164454"
slug: "the-inbox-reads-and-lists-every-waiting"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/report/inbox.go"
remedy: "Have List read the file names first, as Count does, read whole only the newest 200 waiting reports, and end the listing with a line counting the reports it left out."
---

`List` in `internal/core/report/inbox.go` reads every waiting report whole before it drops their bodies, so the cost of one `abcd inbox` listing grows with the number of reports waiting, with no bound on that number. `Count` in the same file already reads file names only.
