---
schema_version: 1
id: "iss-2610160608394736"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.0 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md"
remedy: "Add the (convention) mark to the surface_history bullet at 05-intent.md:504."
resolution: "05-intent.md marks the surface_history bullet (convention), says no shipped check reads the field, and places IL012 in the later-phase IL0xx design"
impact: internal
---

The brief at .abcd/development/brief/04-surfaces/05-intent.md:504 states 'surface_history entries are well-formed' with lint code IL012 (warn) as an unmarked invariant, in a list whose unenforced bullets carry (convention). No shipped check reads surface_history and no IL012 code exists, and line 493 says the IL0xx codes are later-phase design.
