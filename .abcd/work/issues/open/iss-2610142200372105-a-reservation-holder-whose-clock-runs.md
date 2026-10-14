---
schema_version: 1
id: "iss-2610142200372105"
slug: "a-reservation-holder-whose-clock-runs"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "the final review of spc-2610031753359849 steps 8 to 11 and 14"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/claim.go"
remedy: "Take the effective expiry as max(the trailer expiry, the reader's first fetch of the tip plus the lease), or have the writer refuse to push when its clock disagrees with the remote's Date header by more than the margin."
---

A reservation holder whose clock runs behind by more than half its lease plus the five-minute take-over margin can be taken over between its own renewals, so two drains on two accounts can work one issue; the bound is documented on the reserve page and in spc-2610031753359849 but not closed, and it is the one remaining route to that failure.
