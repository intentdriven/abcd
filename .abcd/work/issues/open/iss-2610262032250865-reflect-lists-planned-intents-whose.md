---
schema_version: 1
id: "iss-2610262032250865"
slug: "reflect-lists-planned-intents-whose"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 266 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/09-reflect.md:59"
remedy: "Ruled 2026-10-27, the recommended answer accepted (dcn-2610271112330536): After the cut moves every passed target to next, how does reflect still find the planned intents that targeted the release and did not ship? Answer 1: reflect reads the moves the cut recorded (the target_release rewrites in the cut's change, or the changelog's target-move note). Answer 2: drop the unshipped-targets list and the write refusal from reflect, since the cut already lists them as Targeted, not shipped and names the move. Recommend answer 1: it keeps the retrospective's question about missed targets alive, which answer 2 gives up."
finding_ref: cfk-367556ac91e025bc
---

Reflect lists planned intents whose target_release names the release; the cut has already rewritten every such target to next

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/09-reflect.md:59` — "The planned intents whose `target_release` names the release and which **did not ship**"
- End B: `.abcd/development/brief/04-surfaces/04-launch.md:790` — "The ingest moves every target the cut passes (`next`, or a tag at or below the derived version) to `next`"

Reflect's seed lists the planned intents whose target_release names the released tag and which did not ship, warns about them, and its write refuses until they are confirmed. Launch's ingest moves every target the cut passes, a tag at or below the derived version, to next in the same write that rolls the changelog. By the time a tag exists for reflect to take, no planned intent's target names it, so the list is empty by construction and the refusal can never fire.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 266.
