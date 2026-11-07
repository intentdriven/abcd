---
schema_version: 1
id: "iss-2611060809000628"
slug: "abcd-development-brief-02-constraints"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md"
remedy: "Add `agents` to the exempt list in 04-naming.md."
refines: [iss-2610151422107680]
---

.abcd/development/brief/02-constraints/04-naming.md:56-61 lists the surfaces exempt from the maritime convention and says they are registered there so the exemption is on the record. `agents` is not on the list and is named nowhere else in the chapter as a verb, though the binary registers `abcd agents` (.abcd/development/release/surface.json) and .abcd/development/brief/04-surfaces/41-agents.md describes it.
