---
schema_version: 1
id: "iss-2611060835444393"
slug: "the-land-fence-s-remedy-for-a-parked"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "v0.16.2 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/reservation.go"
remedy: "Have the remedy set ReservedAt, or have the queue watch renew the reservation."
---

The land fence's remedy for a parked lane with no recorded reservation, an `abcd reserve` by hand, lasts one lease: the run's ReservedAt stays nil, so nothing renews it (`reservation.go:91`, `:416`), and the lapse then says a step renews it where no step does (`:458`). The lane is refused again at its next land, on advice that cannot work.
