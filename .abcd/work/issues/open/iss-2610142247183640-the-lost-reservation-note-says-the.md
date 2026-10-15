---
schema_version: 1
id: "iss-2610142247183640"
slug: "the-lost-reservation-note-says-the"
severity: "nitpick"
category: "ux"
source: "impl-review"
found_during: "the last review of spc-2610031753359849 steps 8 to 11 and 14"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/reservation.go"
remedy: "Word the expired case 'was taken by X from another copy, lapsed at T'."
---

The lost-reservation note says the reservation 'is held by X from another copy, until T' for another copy's tip that has already lapsed, which reads as live.
