---
schema_version: 1
id: "iss-239"
slug: "dangling-pre-rebuild-spec-ids-in-the-draft-corpus-itd-44-spc"
severity: "major"
category: "drift"
source: "agent-finding"
found_during: "intent-planning-prep"
found_at: ".abcd/development/intents/README.md"
resolution: "The draft intents itd-30, itd-44, itd-51, itd-55, itd-57, itd-59 and itd-62 qualify every predecessor-store spec id they cite; itd-44, itd-34 and intents/README.md no longer claim the decision verdict was delivered and say no binary code implements it; the specs charter names the live ordinal ceiling spc-70 and the colliding ids. spc-75 had already left itd-61. The planned-intent sibling is iss-2609261536147903."
impact: internal
resolved_by:
  commit: "934a4934"
---

Dangling pre-rebuild spec ids in the draft corpus: itd-44 (spc-56), itd-51 (spc-33/37), itd-57 (spc-48/60/62), itd-61 (spc-75) cite specs from the numbering adr-21/adr-26 reset (live store is spc-1..spc-22). Worse, intents/README.md and itd-34 assert the itd-44 lineage 'landed as itd-44 (spc-56 thin adoption)' — no such spec or decision-verdict code exists, so the record claims a delivery the tree cannot back.

## Grounds

- pursued: every spc id a draft intent cites either names the live spec it means or carries the predecessor-store qualifier, and no record states the decision verdict ships; a draft whose bare spc-N names an unrelated live spec, or a sentence saying the classifier emits decision, would show it wrong
