---
schema_version: 1
id: "iss-2611040555482208"
slug: "validation-rounds-run-two-reviewers-per"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: "CHANGELOG.md"
remedy: "Have the delivering change carry a record; name this default in the next cut's changelog."
---

Validation rounds run two reviewers per code-review role by default, which affects agent spend, with no line in the v0.16.1 changelog, because no record in the cut covers it.
