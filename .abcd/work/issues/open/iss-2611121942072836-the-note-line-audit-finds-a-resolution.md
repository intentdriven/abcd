---
schema_version: 1
id: "iss-2611121942072836"
slug: "the-note-line-audit-finds-a-resolution"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "review of the landings verb"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/note_line_test.go"
remedy: "Find the prints by the type of the value printed, or by the function that returns layer notes, and pin the list of verbs that resolve layers."
---

The note-line audit finds a resolution note's print by the name of the variable a loop ranges over (a name ending in notes), so a loop over layer notes under another name is not seen, and nothing pins which verbs are handed layer notes at all.
