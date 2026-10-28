---
schema_version: 1
id: "iss-2610252215090992"
slug: "glossary-core-technical-facilitator-md-4"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S11"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/technical-facilitator.md"
remedy: "Cite the two ADRs in the glossary entry; correct roles.md to the amended default (a person unless facilitator.mode is set to auto)."
---

glossary/core/technical-facilitator.md:4 and :25 attribute 'the facilitator is a mode, not a person' to draft itd-97; adr-2609151528057260:52 accepted it, and adr-2610150603484802:34-42 set a fresh repository's facilitator to a person. docs/explanation/roles.md:3 and :23 still say the facilitator 'is an AI by default', against that amendment.
