---
schema_version: 1
id: "iss-2610171047088861"
slug: "the-drain-s-renewal-keeps-its-own-copies"
severity: "minor"
category: "tech-debt"
source: "agent-observation"
found_during: "building the load gate's renewal (iss-2610170926295770)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/reservation.go"
remedy: "Once the drain-many-accounts change has landed in the same file, export claim's halfThrough and renewalTerm, have the loop call them and delete its own copies"
---

The drain's renewal keeps its own copies of when a reservation is half through its lease and of the term a renewal pushes (renewalDue and renewalTerm in internal/core/implement/loop/reservation.go), where claim's halfThrough and renewalTerm (unexported until something outside claim calls them) now hold the same two rules for the load gate's renewal: two copies of one rule can drift apart.
