---
schema_version: 1
id: "iss-2610230351350588"
slug: "an-implement-step-refused-at-land-while"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "drain lane run-2610222011116629, 2026-10-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop"
remedy: "Log the reservation_renewed line in the run record whenever a step pushes a renewal, including a step that then refuses at land as contention, so the record matches what origin holds."
---

An implement step refused at land while the lane waits for its full check (exit 3, 'nothing written') still renews the lane's reservation on origin, but the renewal is not logged in the run record as reservation_renewed: on 2026-10-23 abcd peers showed the reservation renewed until 05:51Z while implement record's last reservation line still read 04:33Z, so the record understates how long the run held the issue.
