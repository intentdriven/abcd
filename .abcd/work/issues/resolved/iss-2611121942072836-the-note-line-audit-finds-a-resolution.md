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
resolution: "The note-line audit now follows layer notes from layered.RootsFor and the package functions that return them, whatever the variable is called, flags a notes function taken as a value or the layered package imported under another name, and pins the functions that resolve layers. Its three tests pass against the package's current source."
impact: fix
resolved_by:
  commit: "08dbd1236d1e9f39f951ae91c6f456e553fcb9a3"
---

The note-line audit finds a resolution note's print by the name of the variable a loop ranges over (a name ending in notes), so a loop over layer notes under another name is not seen, and nothing pins which verbs are handed layer notes at all.
