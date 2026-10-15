---
schema_version: 1
id: "iss-2610142247187628"
slug: "logonce-dedupes-a-reservation-entry-on"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "the last review of spc-2610031753359849 steps 8 to 11 and 14"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/reservation.go"
remedy: "Compare a kind token as well as the stage before skipping an entry."
---

logOnce dedupes a reservation entry on its stage alone, not the kind of failure, so a different failure after an unreachable origin (a take-over by another person, say) is dropped from the run record, though the step's answer still names it.
