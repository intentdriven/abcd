---
schema_version: 1
id: "iss-2611190011141331"
slug: "claim-snapshot-contention-the-overlap"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "autonomous run of 15 Nov 2026, review of the build next overlap fix"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/crowd.go"
remedy: "Make Contention skip a reservation from this copy only when its holder is the reader's own, the same test crowding makes, with a test that re-mints the holder id and finds the candidate excluded at overlap and the start refused alike."
resolution: "Contention now exempts a reservation from this copy only when its holder is the reader's own (Reservation.OwnHere, the predicate crowding shares), so build next, the board head, the drain's pass-over and the widening's peer-hold read all agree with the reservation act after a holder id is re-minted"
impact: fix
---

claim.Snapshot.Contention, the overlap rule build next and the drain judge against, exempts every live reservation made from this copy, while the reservation act's own crowding exempts one only when this copy made it under a holder id the reader owns. After a holder id is re-minted (home emptied), a reservation from this copy under the old id holds a path; build next keeps the intent whose footprint shares it and the start backs off with exit 3, and the next pick names the same intent again because the view has not changed.
