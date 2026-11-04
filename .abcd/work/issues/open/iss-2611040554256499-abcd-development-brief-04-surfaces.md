---
schema_version: 1
id: "iss-2611040554256499"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/30-inbox.md"
remedy: "Limit the sentence to report-shaped names, or list every other file as unreadable too."
---

.abcd/development/brief/04-surfaces/30-inbox.md:30-32 says a waiting file this abcd cannot read is listed as unreadable, never dropped, and that a file that is not a report says so. The list reads only names matching fileNameRe (internal/core/report/inbox.go:89, 373). A file named notes.md in the inbox is absent from `abcd inbox`, its tally and its unreadable rows; only a report-shaped name is listed as unreadable.
