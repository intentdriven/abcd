---
schema_version: 1
id: "iss-2610252224170402"
slug: "abcd-work-rulesets-readme-md-10-says"
severity: "nitpick"
category: "drift"
source: "agent-finding"
found_during: "state-of-the-art passes of the hand-back triage, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/work/rulesets/README.md"
remedy: "Correct the count at README.md:10 to ten, or replace the number with a pointer to the list in main-protection.json so the two cannot drift again."
refines: [iss-296]
resolution: "the README points at the mirror's required_status_checks list instead of stating a count"
impact: internal
resolved_by:
  commit: "2bb0530796bef1833b1064b9af2e04477fb6249c"
---

.abcd/work/rulesets/README.md:10 says main-protection.json carries the nine required status checks, but main-protection.json lists ten.
