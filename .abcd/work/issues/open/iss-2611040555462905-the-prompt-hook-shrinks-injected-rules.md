---
schema_version: 1
id: "iss-2611040555462905"
slug: "the-prompt-hook-shrinks-injected-rules"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: "CHANGELOG.md"
remedy: "Have the delivering change carry a record; name this behaviour in the next cut's changelog."
---

The prompt hook shrinks injected rules, notes and pages as a session's context fills, a behaviour change with no line in the v0.16.1 changelog, because no record in the cut covers it.
