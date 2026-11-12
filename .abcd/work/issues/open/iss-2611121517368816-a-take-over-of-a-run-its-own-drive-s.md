---
schema_version: 1
id: "iss-2611121517368816"
slug: "a-take-over-of-a-run-its-own-drive-s"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/resume.go"
remedy: "Judge the whole save before a take-over clears the stop's mark on every route, so a take-over that judge refuses neither clears the mark nor adds its line to the run's record"
---

A take-over of a run its own drive's signal halted clears the stop's mark, and adds that line to the run's record, before the whole save of the run is judged where the checkout has no origin, where the halt held no reservation, or where the record has already finished on the default branch; a take-over that judge then refuses, the run's file at another schema version the session may not save, has cleared the mark though nothing else of the take-over is saved, and a drive passes the lane over. Where the checkout has an origin and the halt released the issue, the same judge refuses the take-over at its reservation, before anything is pushed and before the mark is cleared, and a reservation pushed in the window before the save is released again when the save is refused, so no route leaves the lane holding a reservation the take-over made
