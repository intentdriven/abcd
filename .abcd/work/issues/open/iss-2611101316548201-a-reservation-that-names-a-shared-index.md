---
schema_version: 1
id: "iss-2611101316548201"
slug: "a-reservation-that-names-a-shared-index"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/crowd.go"
remedy: "Waits on a ruling first: then let a repository declare shareable files whose overlap does not refuse a reservation, and have the overlap check in internal/core/claim/crowd.go warn for them instead of refusing."
---

A reservation that names a shared index file holds off every other change to that file, so builds that only add a line to it are refused. On 8 Nov at 06:15 build refused an intent because a live reservation named internal/README.md, the package map that most changes touching a new package edit. A person must choose whether some files may be shared between footprints, and how such a file is declared (a list in the repository's configuration, or a mark in the footprint).
