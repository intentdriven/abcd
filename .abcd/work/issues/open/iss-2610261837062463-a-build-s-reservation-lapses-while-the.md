---
schema_version: 1
id: "iss-2610261837062463"
slug: "a-build-s-reservation-lapses-while-the"
severity: "major"
category: "bug"
source: "agent-observation"
found_during: "build itd-82 lane-1, 2026-10-26"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/reservation.go"
remedy: "Renew a run's reservation while an agent holds a stage, not only when it hands its receipt back (receipt already renews in loop.go): the gated waiter an agent's go test and make preflight run under (implement load --wait), and implement status, renew any reservation of a run in this checkout that is due, as step does; and judge a lapsed revival's overlap against the lane's changed files, as the landing's narrowing does, not the spec's whole footprint, so the same person's other copy holding an unrelated file stops nothing."
---

A build's reservation lapses while the host waits on an agent, because only implement step renews it (renewReservation in internal/core/implement/loop/reservation.go): build itd-82's run-2610260500113538 reserved itd-82 until 07:00Z, its implementer then held the implement stage for about thirteen hours under the shared machine's load gate, and no step ran to renew it. Revived after the lapse, the reservation is judged like a new one (crowd.go), and a build's automated act backs off on any overlap with the same person's other copy, so the renewal was refused for loop.go, in the footprint of iss-2610142134353345, live from the main checkout. The run goes on to validation holding no reservation, and the land fence will stop it, though nothing of the other record's work touches the lane's files.
