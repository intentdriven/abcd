---
schema_version: 1
id: "iss-2611190234100152"
slug: "the-start-s-peers-check-for-a-record-s"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run of 15 Nov 2026, fixing iss-2611190011141331"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/check.go"
remedy: "Make reservationHolder name a live tip from this copy whose holder the reader does not own (claim.Reservation.OwnHere) as a holder, leaving HeldBy's board text as it is, with a test that re-mints the holder id and finds the intent excluded at peers and its start refused alike."
---

The start's peers check for a record's own reservation (loop peerSnapshot.reservationHolder, through claim.Snapshot.HeldBy) passes every reservation from this copy, whatever its holder, while the reservation act backs off a live tip from this copy whose holder id the reader no longer owns. After a holder id is re-minted, build next and the board head keep an intent this copy reserved under the old id, and its start backs off with exit 3; Contention's own-tip check, fixed under iss-2611190011141331, already holds it off for the drain.
