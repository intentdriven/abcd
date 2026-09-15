---
schema_version: 1
id: "iss-2609021748300398"
slug: "the-entailment-assembly-hands-every-draft-and-planned-intent"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "cold-reading Phase A, orchestrator review of the preset windows"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/scope.go"
resolution: "The drafts and planned include-table rows now narrow by the committed entry's object-set record list exactly as the shipped row does, so a draft or planned intent travels at entailment only when the entry names it. The readings companion's section 6.2 admissibility survives as admit_drafts_and_planned on the entailment entry: default off, refused at load on any other position, and when on it hands every draft and planned intent the position admits. The committed entailment entry names the ten Iteration 2 intents beside the fifteen workstream intents and declares no switch, so the reading is handed the object set the design framework's section 13 fixes and nothing beyond it."
impact: fix
resolved_by:
  intent: "itd-2609020625400445"
  spec: "spc-2609020626048722"
---

The entailment assembly hands every draft and planned intent in the repository — 147 projected intents on this tree — where the design framework's section 13 fixes the object set as Iteration 1's shipped state and claim record: the fifteen workstream intents, which the record extends by the ten Iteration 2 intents. The readings companion's section 6.2 makes drafts and planned intents admissible at the entailment position, and admissible is a permission rather than a scope; the assembler reads it as the scope because a record row is admitted whole when the object set names none of its records, so the drafts and planned rows are never narrowed by the entry's record list the way the shipped row is. The committed entailment entry therefore hands material the object set does not name, and no committed fact says which drafts and planned intents a run is about.

## Grounds

- pursued: the entailment reading is handed the object set alone, so an assembly at that position carries exactly the drafts and planned intents the committed entry names; it would be shown wrong by an assembly handing a draft the entry does not name, by the switch failing to hand all of them, or by the entry needing a record the object set does not contain in order to reach material the reading is about.
