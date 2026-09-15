---
schema_version: 1
id: "iss-2608301306587401"
slug: "link-and-plan-do-unlocked-read-modify-writes-on-records-reco"
severity: "nitpick"
category: "tech-debt"
source: "user-observation"
found_during: "itd-179-round-3-security"
found_at: "internal/core/intent/lifecycle.go"
---

Link and Plan do unlocked read-modify-writes on records RecordGrounds now locks

Found by the round-3 adversarial security review of build/itd-179, and recorded
BELOW THE FINDING BAR by the reviewer's own judgement -- worth keeping, not
worth acting on yet.

`internal/core/intent/lifecycle.go:398` (`Link`) and `:207,227` (`Plan`) do
unlocked read-modify-writes on records that `RecordGrounds` also writes
(`planned/`, `drafts/`). So the lock added in ac48db99 serialises RecordGrounds
against itself and against `stampPlanned`, but not against these two peers.

Why it is not a finding: the reviewer could not demonstrate a loss in 400
trials, against a loss within 3 trials for the case the lock actually fixed.
That gap in reproducibility is the whole record -- the theoretical race is real,
the practical window is not demonstrable, and the honest thing is to say so
rather than either fix it blind or drop it.

Pre-existing on main. The branch's new writer is the one that DOES lock.
