---
schema_version: 1
id: "iss-2611101318415625"
slug: "a-lane-s-receipt-cannot-declare-a"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/receipt.go"
remedy: "Have the landing resolve only what the lane's latest receipt declares, add a partial form that lands the change under Refs: and leaves the issue open with the delivered part noted, and let a sync receipt name the issues it resolves with the commit that delivered each; a test withdraws a resolution in a second receipt and finds the landing leaves the issue open."
---

A lane's receipt cannot declare a partial fix, withdraw a resolution an earlier receipt declared, or name issues a sync finds delivered. recordReceipt (internal/core/implement/loop/receipt.go) replaces or appends a resolution and never removes one, and the landing resolves every issue any receipt named. On 8 Nov at 05:44 a fix receipt left out an issue that round 1 had ruled out of the resolutions, the landing still listed it, and both round-2 reviewers blocked on that; on 8 Nov at 06:50 a part-delivered issue had to be resolved whole and its unbuilt half captured again; on 7 Nov at 09:45 an issue lane landed half its issue with no way to say so; on 7 Nov at 23:25 the sync brief's receipt had no resolves field, so an issue an intent's audit found delivered was resolved by hand.
