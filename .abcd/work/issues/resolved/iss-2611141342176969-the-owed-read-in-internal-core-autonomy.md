---
schema_version: 1
id: "iss-2611141342176969"
slug: "the-owed-read-in-internal-core-autonomy"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "review of the renewal rule"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/autonomy/owedread.go"
remedy: "Export from claim a function returning the instant a reservation is half through its term, have HalfThrough and the owed-read's wake time both use it, and extend the loop's and a new autonomy source test to hold both packages to it."
resolution: "claim exports HalfThroughAt, the instant a reservation is half through its term; HalfThrough answers by it and the owed-read's next renewal and wake time take it, and the loop's source test now holds the autonomy sources to the rule, with a new autonomy source test requiring reservationItems to call it."
impact: internal
resolved_by:
  commit: "5c7e55173f57c08c526f399ffe9aebf8fd25c8e0"
---

The owed-read in internal/core/autonomy/owedread.go (reservationItems) works out the instant a reservation is half through its term itself, from its claimed-at and effective expiry, and uses it for the next renewal and the wake time. It agrees with claim's due point (claim.HalfThrough) today, but HalfThrough returns only whether that point has passed, not the instant, so the owed-read cannot call it; a change to claim's due point would leave the wake time at the old half.

## Grounds

- pursued: the owed-read's wake time follows claim's due point wherever claim moves it, with no behaviour change today; the loop's and autonomy's source tests failing on a term subtraction reintroduced in autonomy, or TestTheRenewalRule and the autonomy package's tests failing, would show it wrong.
