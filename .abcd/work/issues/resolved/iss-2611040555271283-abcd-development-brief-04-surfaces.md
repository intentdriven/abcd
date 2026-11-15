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
resolution: "The identity chapter now says a surface whose candidate files are all absent is reported as absent, not as drift: abcd lint identity prints it with the state absent and its JSON lists it with that status, and bare abcd lint stays silent on it. TestBriefIdentityChapterSaysAbsentSurfaceIsReportedAbsent holds the chapter to StatusAbsent."
impact: internal
resolved_by:
  commit: "2bbd4e916a3d29913cd58785607d4d7ec54616ad"
---

.abcd/development/brief/04-surfaces/19-identity.md:103-105 says a surface whose candidate files are all absent is skipped rather than reported. `abcd lint identity` prints each such surface with the state absent, and its JSON lists it with status `absent`. Only bare `abcd lint` leaves it out, and it counts as no drift.

## Grounds

- pursued: the chapter matches positioning.Check (StatusAbsent), the lint identity render and the identity-positioning rule; a chapter that says the surface is skipped, or a render that stops listing absent surfaces, would show it wrong, and the detector fails on the first
