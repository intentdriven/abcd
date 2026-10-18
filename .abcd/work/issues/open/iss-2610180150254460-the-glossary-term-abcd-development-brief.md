---
schema_version: 1
id: "iss-2610180150254460"
slug: "the-glossary-term-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "build of the v0.15.0 deferred glossary findings (fix/deferred-glossary-flake)"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/intent.md"
remedy: "Rewrite the sentence to say which parts of a planned intent stay fixed and name the sub-verbs that may still edit it after planning."
---

The glossary term .abcd/development/brief/glossary/core/intent.md says an intent is frozen at promotion time, an immutable input artefact never edited after /abcd:intent plan is run (lines 20-21), but shipped sub-verbs edit planned intents: abcd intent target writes a planned intent's target_release line, intent edge writes its blocked_by and builds_on lists, intent hold writes a held line on a draft or planned intent, intent plan on a planned intent stamps its scope conditions, and intent interview runs on a planned intent and edits its record.
