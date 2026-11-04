---
schema_version: 1
id: "iss-2611040555499163"
slug: "the-question-limits-and-abcd-guard"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: "CHANGELOG.md"
remedy: "Have the delivering change carry a record; name these in the next cut's changelog."
refines: [iss-2609012313465609]
---

The question.* limits and abcd guard question are new since v0.16.0, and default rows rose from 24 to 32, with no line in the v0.16.1 changelog, because no record in the cut covers them.
