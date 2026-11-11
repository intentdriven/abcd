---
schema_version: 1
id: "iss-2611032348360959"
slug: "a-spec-footprint-naming-a-whole"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "coordinating the build lanes and the walled subscription test, 2026-11-03"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/reservation.go"
remedy: "Reserve each lane by its own step's paths rather than the whole spec's footprint, so two lanes in one directory that share no file both proceed; a test holds one file and renews a sibling lane."
---

A spec footprint naming a whole directory collides with a peer holding one file in it: run-2611031640393520's footprint has internal/core/lint, a cloud session holds internal/core/lint/prosecitations.go, and the run's renewal is refused although the lane changes none of that session's files.

2026-11-09: seen again on 9 Nov at 08:27 and 12:27 in the autonomous run of 7 Nov: a spec footprint naming the whole internal/core/implement/loop package was refused because another holder had reserved one test file in it, and the step's own files were reserved by hand instead; on 10 Nov at 12:23 the same collision stopped a one-package step. iss-2610230839002949 lists the other sightings.
