---
schema_version: 1
id: "iss-2610150512342464"
slug: "footprints-name-files-not-packages"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "reservations-for-builds review 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim"
remedy: "Let a spec's footprint name files or globs beneath a package (e.g. internal/surface/cli/session.go) as well as whole packages, and judge reservation overlap on those paths, so two builds that add different files to one hub package no longer back off from each other; keep package entries valid, and decide how an unlisted file a build touches is caught (a land-time check of the diff against the footprint)."
---

Reservation footprints are whole packages, so concurrent builds collide on hub packages even when they touch different files. A spec's footprint lists packages, and the reservation overlap check backs off any two holders whose packages intersect, including the same person's two worktrees (product-thinker ruling 2026-10-15: that serialisation stays). Ten of the thirteen open specs that declare a footprint list internal/surface/cli, the package every verb's command lives in, so in practice nearly every pair of concurrent builds runs one at a time even when one adds a new file there and the other edits a different one. Naming files instead of packages would keep the protection while letting non-overlapping builds run side by side; the cost is that every spec must name its files exactly, and a file it forgets is unprotected unless something checks the landed diff against the footprint.
