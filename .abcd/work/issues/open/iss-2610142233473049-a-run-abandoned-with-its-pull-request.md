---
schema_version: 1
id: "iss-2610142233473049"
slug: "a-run-abandoned-with-its-pull-request"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "the last review of spc-2610031753359849 steps 8 to 11 and 14"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/reservation.go"
remedy: "Tombstone the run's reservation when the run is ended or its pull request closes unmerged (the landing's closed-without-merging refusal, and a run removed by hand)."
---

A run abandoned with its pull request closed unmerged, and the run left in place, keeps its issue or intent reserved for up to a day, since the fence now renews for a day under a ruleset mirror and nothing tombstones a reservation when a run is abandoned.
