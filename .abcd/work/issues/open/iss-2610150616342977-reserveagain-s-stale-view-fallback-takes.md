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
remedy: "Pass a renewal term on reserveAgain's stale-view fallback too, so a renewal through it keeps the fenced term; the view does not show the reservation there, so the term must come from the fetched tip (a claim.Request that renews with the tip's own term)."
---

reserveAgain's stale-view fallback takes the plain reserve's renewed row with the default lease, not the preserved term (renewalTerm): in the narrow case where the view as last fetched hides this copy's own live reservation, a resume renews it through the plain reserve and can cut the extended day the land fence gave a pull request waiting on a person.

The same holds on the widened-footprint path (849ad17bc): when the spec's footprint has grown past the tip's, a resume takes the plain reserve so the widening is judged for overlap, and that reserve too renews this copy's own reservation with the default lease rather than the preserved term.

The widened-footprint path was fixed in the change that carries this note: that reserve now passes renewalTerm of the reservation the view shows, so it keeps the term (TestAWidenedResumeKeepsTheReservationsTerm). What stays open is the stale-view fallback alone, where the view does not show this copy's reservation and so has no term to pass.
