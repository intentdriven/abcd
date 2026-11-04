---
schema_version: 1
id: "iss-2611040555455335"
slug: "abcd-build-auto-plan-is-new-since-v0-16"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: "CHANGELOG.md"
remedy: "Have the delivering change carry a record, so a cut's changelog names a new surface that behaves differently unasked."
---

abcd build --auto-plan is new since v0.16.0 and plans a draft with no person present, but the v0.16.1 changelog carries no line for it, because no record in the cut covers it; only a fix to it appears.
