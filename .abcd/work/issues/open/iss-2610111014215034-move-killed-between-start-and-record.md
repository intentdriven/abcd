---
schema_version: 1
id: "iss-2610111014215034"
slug: "move-killed-between-start-and-record"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "build next lane review, run-2610110908001704, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/drain.go"
remedy: "Write the pick (or drain lane) into the series state as 'starting <run-id>' before the run is created, and on the next move adopt a run whose id the state already names; test by failing the record write after the start and asserting the next move counts the run."
---

A move that starts a run and then records it is two durable writes, so a process killed between them leaves a run its series never records: in build next's series (loop/nextseries.go, startPick then finish) a --max 2 series can start three runs and never report the orphan's outcome, and the drain's drainMove (Start then finish) has the same gap. Found by the build-next lane's ruthless review (run-2610110908001704).
