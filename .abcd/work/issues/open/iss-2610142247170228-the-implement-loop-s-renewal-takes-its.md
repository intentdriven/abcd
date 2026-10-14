---
schema_version: 1
id: "iss-2610142247170228"
slug: "the-implement-loop-s-renewal-takes-its"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "the last review of spc-2610031753359849 steps 8 to 11 and 14"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/reservation.go"
remedy: "Have decideRenew keep the term of the tip it just judged (a KeepTerm request flag), or update-ref the tracking ref after a successful push."
---

The implement loop's renewal takes its term from the reservation tip as last seen in the remote-tracking ref (claim.View), not the tip claim.Reserve fetches fresh; in a single-branch or depth-limited clone, or one with a narrowed fetch refspec, a push does not update that ref, so a stale two-hour tip can make a drain push a two-hour renewal over the fence's day-long tip and lapse the hold.
