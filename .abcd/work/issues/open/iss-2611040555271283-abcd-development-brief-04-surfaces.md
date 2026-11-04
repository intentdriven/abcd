---
schema_version: 1
id: "iss-2611040555271283"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/19-identity.md"
remedy: "Say the target reports the surface as absent, not as drift, and that bare lint stays silent on it."
refines: [iss-2609011423385217]
---

.abcd/development/brief/04-surfaces/19-identity.md:103-105 says a surface whose candidate files are all absent is skipped rather than reported. `abcd lint identity` prints each such surface with the state absent, and its JSON lists it with status `absent`. Only bare `abcd lint` leaves it out, and it counts as no drift.
