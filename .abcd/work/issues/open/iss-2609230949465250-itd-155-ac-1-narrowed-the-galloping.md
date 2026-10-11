---
schema_version: 1
id: "iss-2609230949465250"
slug: "itd-155-ac-1-narrowed-the-galloping"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "autonomous run 2026-09-23 fidelity audit"
origin: researcher-authored
production_mode: hand-written
remedy: "Waits on the owed ruling: if the promise is narrowed, add an Audit Notes line to shipped itd-155 stating that ac-1's no-truncation guarantee holds within gallopBudget's per-line 4*len+4096 bytes, with an issue naming the change (the H10 form); if the promise stands, extend a match cut at the window edge once the budget is spent to the end of its non-space run, so the redaction over-covers instead of truncating. Either way pin a test on a line engineered to exhaust the budget that asserts the ruled match end."
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (renewed by run A 2026-09-29 after the v0.10.0 grant lapsed at the v0.11.0 anchor): Accept that itd-155's no-truncation guarantee holds within a per-line budget (4*len+4096 bytes), beyond which the fixed 512-byte window returns, and amend ac-1 to say so?"
---

itd-155 ac-1 narrowed: the galloping adjacency probe (internal/adapter/scanner/scanner.go gallopingFind) grows under a per-line budget of 4*len(line)+4096 bytes and, once that budget is spent, keeps the fixed 512-byte window it already has — so on a line engineered to grow the window at many junctions a match end can again be a truncation artefact of the window edge, the shape iss-189/iss-190 were about. A single long token always fits the budget. The trade-off (a measured quadratic cliff without the cap) is recorded in code only; the intent promises 'never a truncation artifact' unconditionally, so the promise or the record should say where the guarantee ends

## Remedy grounds (2026-09-29)

- Why: the two answers the owed question admits; the ruling is unanswered and none is picked. The second keeps the measured quadratic cliff closed, because extending to the end of a run is one linear scan.
- Rejected: removing the budget, which restores the quadratic cliff the cap exists to close.
