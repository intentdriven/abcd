---
schema_version: 1
id: "iss-2610310331282356"
slug: "the-v0-14-0-upgrade-guide-states-in"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/how-to/upgrade-to-v0.14.0.md"
remedy: "Make the v0.14.0 upgrade guide's lists say what v0.14.0 shipped, or point at the commands reference for the current set instead of listing it."
refines: [iss-2610251558110452]
resolution: "the user-facing page or help example now states what the binary does"
impact: fix
---

The v0.14.0 upgrade guide states in present tense the menu's commands and the agents-only verbs, and both lists now omit shipped verbs (embargo, land, session, status; announce, checklist, memory, reserve).
