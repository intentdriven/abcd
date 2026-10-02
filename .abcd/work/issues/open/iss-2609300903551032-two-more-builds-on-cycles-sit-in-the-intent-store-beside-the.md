---
schema_version: 1
id: "iss-2609300903551032"
slug: "two-more-builds-on-cycles-sit-in-the-intent-store-beside-the"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned"
remedy: "Ruled by CY2 (the technical facilitator, 2026-10-02): both edges of each pair stand as declared mutual pairs (itd-14 with itd-15, itd-2609081951381895 with itd-2609170822093401), so no edge is dropped. The fix now depends on iss-2610020836255174: once the intent record can declare a mutual pair and edge_cycle honours it, declare both pairs there so edge_cycle reports neither; then, with iss-2609300848016421 and the itd-22 and itd-33 edges decided, promote stale_edge and edge_cycle to blocker."
---

Two more builds_on cycles sit in the intent store beside the runner and loop one: the drafts itd-14 and itd-15 each declare builds_on the other, and the planned itd-2609081951381895 (the OpenAI-compatible API oracle adapter) and itd-2609170822093401 (the oracle choice is one repo-wide value) each declare builds_on the other; record-lint's edge_cycle rule reports both
