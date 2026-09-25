---
schema_version: 1
id: "iss-2608311025494209"
slug: "itd-184-adds-four-cold-reading-definitions-to-agents-taking"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "already fixed at tip: 05-internals/01-agents.md says fifteen agent prompts and lists the cold-reading four (53a5a913)"
impact: internal
resolved_by:
  commit: "53a5a913eb22a0ee7dd30d6210eecdc2b9fb174b"
---

itd-184 adds four cold-reading definitions to agents/, taking the shipped prompt count from eleven to fifteen, but 05-internals/01-agents.md still says 'Eleven agent prompt files ship in agents/ today' (line 3) and 'Carried today by all eleven shipped prompts -- the five synthesis/composer prompts, the five reviewer/researcher prompts and the ledger scribe alike' (line 106). The four are also absent from that chapter's 'Shipped agents outside the design roster' section, so the roster no longer lists every shipped prompt. The itd-184 builder's lane did not include 05-internals, so the correction was captured rather than taken.

## Grounds

- pursued: the agents chapter counts and lists every shipped prompt; shown wrong if agents/ holds a prompt the chapter omits
