---
schema_version: 1
id: "iss-2610160608430668"
slug: "the-row-at-abcd-development-brief"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.0 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md"
remedy: "Rewrite the row at 02-verification-matrix.md:104 to the one-day fence and cite TestTheFenceHoldsADayWhenThePullRequestIsLeftForAPerson."
---

The row at .abcd/development/brief/06-delivery/02-verification-matrix.md:104 says the land fence 'renews for a day under a ruleset mirror and the default two hours without one', citing loop.TestTheFenceRenewsWithTheDefaultLeaseWithoutAMirror. No test of that name exists. The fence holds a day with or without a mirror (fenceLease = lease.Max, internal/core/implement/loop/reservation.go:179), and the no-mirror case is held by loop.TestTheFenceHoldsADayWhenThePullRequestIsLeftForAPerson (internal/core/implement/loop/parked_reserve_test.go:188).
