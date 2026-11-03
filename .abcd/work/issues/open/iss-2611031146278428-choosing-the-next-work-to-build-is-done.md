---
schema_version: 1
id: "iss-2611031146278428"
slug: "choosing-the-next-work-to-build-is-done"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "coordinating the build lanes, 2026-11-03"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/board"
remedy: "Add a board view (or build --next) listing what an unattended driver can start now, and for each ready item it skips the reason: held, a live or person-owned step, a major, a footprint another copy holds, or already building."
---

Choosing the next work to build is done by hand: the board lists 32 ready intents, but on 2026-11-03 the first three could not be built unattended (one held on a product ruling in prose, one whose first step is a live proof needing the person's yes, one mid-build), majors are refused by the build, and a build whose footprint overlaps another copy's reservation is refused, so the coordinator read each record and chose a copy to build from by hand.
