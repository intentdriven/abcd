---
schema_version: 1
id: "iss-2609230949465250"
slug: "itd-155-ac-1-narrowed-the-galloping-adjacency-probe-internal"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "autonomous run 2026-09-23 fidelity audit"
origin: researcher-authored
production_mode: hand-written
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Accept that itd-155's no-truncation guarantee holds within a per-line budget (4*len+4096 bytes), beyond which the fixed 512-byte window returns, and amend ac-1 to say so?"
---

itd-155 ac-1 narrowed: the galloping adjacency probe (internal/adapter/scanner/scanner.go gallopingFind) grows under a per-line budget of 4*len(line)+4096 bytes and, once that budget is spent, keeps the fixed 512-byte window it already has — so on a line engineered to grow the window at many junctions a match end can again be a truncation artefact of the window edge, the shape iss-189/iss-190 were about. A single long token always fits the budget. The trade-off (a measured quadratic cliff without the cap) is recorded in code only; the intent promises 'never a truncation artifact' unconditionally, so the promise or the record should say where the guarantee ends
