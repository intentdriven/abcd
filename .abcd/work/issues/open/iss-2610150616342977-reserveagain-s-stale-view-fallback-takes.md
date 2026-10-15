---
schema_version: 1
id: "iss-2610150616342977"
slug: "reserveagain-s-stale-view-fallback-takes"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "final fix round of spc-2610031753359849 steps 12, 13 and 15 (feat/reserve-build)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/reservation.go"
remedy: "Pass the run's renewal term on reserveAgain's plain-reserve fallback too, so a renewal through it keeps the fenced term."
---

reserveAgain's stale-view fallback takes the plain reserve's renewed row with the default lease, not the preserved term (renewalTerm): in the narrow case where the view as last fetched hides this copy's own live reservation, a resume renews it through the plain reserve and can cut the extended day the land fence gave a pull request waiting on a person.
