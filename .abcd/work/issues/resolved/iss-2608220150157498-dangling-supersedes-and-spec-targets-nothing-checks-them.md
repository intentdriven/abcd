---
schema_version: 1
id: "iss-2608220150157498"
slug: "dangling-supersedes-and-spec-targets-nothing-checks-them"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "abcdev-site-plan investigation 2026-08-21"
found_at: ".abcd/development/decisions/adrs"
related_intents: [itd-160]
resolution: "the dangling-reference detector is armed and proven: both field kinds reach the gate, the eight-entry backlog is baselined, and the four ratchet behaviours are pinned by tests"
impact: internal
resolved_by:
  intent: "itd-160"
  spec: "spc-52"
---

Eight typed cross-references point at targets absent from the tree — adr-22 supersedes adr-14, adr-15 and adr-17; adr-25 supersedes adr-8; adr-27 supersedes adr-16; adr-28 supersedes adr-18; adr-35 supersedes adr-4 (all retired under retire-the-name); itd-3 names spec_id spc-1 which has no file — and nothing checks supersedes targets today. The 2026-08-21 site investigation counted six; the in-session grep found eight. The planned site build arms the detector via the .abcd/site-baseline.json ratchet, seeded with what the build finds, and the tombstones-or-stubs question (itd-136/itd-137) decides how they render