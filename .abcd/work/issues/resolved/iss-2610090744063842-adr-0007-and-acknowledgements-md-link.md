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
resolution: "ADR 0007 and itd-27 now credit the to-prd PRD template with a permalink to skills/engineering/to-prd/SKILL.md at c5a4a8c2, the last upstream commit that held it before the 2026-07-08 merge into to-spec and to-tickets, so the attribution resolves again."
impact: internal
resolved_by:
  commit: "18357b67ad69d55e08e0bcad8f89273c687dce48"
---

ADR 0007 and the superseded itd-27 link the PRD template's source at mattpocock/skills skills/engineering/to-prd/SKILL.md, but that skill was merged into to-spec and to-tickets upstream on 2026-07-08, so the link now answers 404 and no longer shows the source it credits.

## Grounds

- pursued: both records' to-prd links answer HTTP 200 and show the credited SKILL.md; a 404 from either permalink, or a record left without a to-prd SKILL.md link, would show it wrong
