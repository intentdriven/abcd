---
schema_version: 1
id: "iss-2610090744063842"
slug: "adr-0007-and-acknowledgements-md-link"
severity: "nitpick"
category: "drift"
source: "agent-finding"
found_during: "2026-10-09 ideate research leg"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/decisions/adrs/0007-grill-skill-and-glossary.md"
remedy: "Repoint the credit to the commit that last held to-prd (a permalink to that SKILL.md at its last commit) so the attribution still resolves, in the ADR and in itd-27 alike."
---

ADR 0007 and the superseded itd-27 link the PRD template's source at mattpocock/skills skills/engineering/to-prd/SKILL.md, but that skill was merged into to-spec and to-tickets upstream on 2026-07-08, so the link now answers 404 and no longer shows the source it credits.
