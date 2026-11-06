---
schema_version: 1
id: "iss-2611060810119096"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/18-ideate.md"
remedy: "Add `implies_building` to the list in the chapter and the command page, or exempt the field if a gap may name a record yet to exist."
---

`.abcd/development/brief/04-surfaces/18-ideate.md:163-166` lists where an unresolved record id refuses the verdict: a claim or its primary source, a grill note, a kill attempt, a reviewer's label or sources, the ruling, a rejected alternative, an origin source or the reframing. A reviewer's `implies_building` entries are checked too (`citationPlaces`, `internal/core/ideate/record.go:1151`), and `commands/ideate.md` does not name them either. The field is meant to name gaps, so a composer reading the list would not expect the refusal.
