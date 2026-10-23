---
schema_version: 1
id: "iss-2610230651407493"
slug: "while-a-lane-s-landing-is-held-by"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "drain lane run-2610222011116629, 2026-10-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop"
remedy: "On the land fence's release-hold path, renew only when half the lease has passed, with the day-long term the fence documents, so a held lane pushes at most a few renewals over the hold."
---

While a lane's landing is held by another copy's release reservation (rel-vX.Y.Z), every implement step pushes a fresh renewal of the lane's reservation to origin, whatever its age: on 2026-10-23 the branch abcd-reserved/iss-2610120138028442 took five renewal commits between 06:02:43Z and 06:02:57Z and one per later step, each for two hours. commands/build.md says a renewal is pushed only once half the lease has passed, and that the land fence, when it holds, renews for a day; neither holds on the release-hold path, so a lane held for a day writes dozens of commits to the reservation branch.
