---
schema_version: 1
id: "iss-2611032348355667"
slug: "implement-step-hands-out-an-implementer"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "coordinating the build lanes and the walled subscription test, 2026-11-03"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/reservation.go"
remedy: "Refuse with exit 3 when the run holds no live reservation, naming the holder and when its hold ends; a test lapses the reservation, refuses the renewal and asserts no stage is handed out."
---

implement step hands out an implementer (rc=0) on a run whose reservation has lapsed and whose renewal was refused, saying take other work in the same message: on 2026-11-03 run-2611031640393520 would have started an agent unreserved had its coordinating session not stopped.
