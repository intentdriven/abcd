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
remedy: "Judge the whole save after the take-over's claim and before it reserves the issue again or clears the stop's mark, so a take-over that judge refuses pushes and writes nothing"
---

A take-over of a run its own drive's signal halted reserves the issue again and clears the stop's mark before the whole save of the run is judged, so a take-over that judge then refuses, the run's file at another schema version the session may not save, has pushed a reservation and cleared the mark though nothing of the take-over is saved; a drive then passes the lane over holding that reservation, which nothing renews or releases until it lapses with its lease
