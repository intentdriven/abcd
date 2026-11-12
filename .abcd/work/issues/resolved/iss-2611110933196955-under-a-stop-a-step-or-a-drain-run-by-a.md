---
schema_version: 1
id: "iss-2611110933196955"
slug: "under-a-stop-a-step-or-a-drain-run-by-a"
severity: "major"
category: "bug"
source: "agent-observation"
found_during: "review of the landing record's schema raise"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/control.go"
remedy: "Record in the run state which checkout and account drives the run, apart from the driver's process, and have every save that takes the whole state, by a session that is not that driver, judge the version the file states first, as the renewal does: the step's stop check, a drain's move over its own lanes, and finishMerged's landing step, resyncParked, watchParked and handBackClosed; keep green the upgrade-resume test and a new drain finishing an earlier drain's merged lanes, and add an on-disk test per route."
resolution: "Every whole save of a run by a session that may not be its driver is judged first by the version its file states: a file at another version is saved only by the live session the run's control record names as its driver, and is otherwise left as it stands with a line naming the file and both versions. The step under a stop, a drain's stop and its own lanes, the four parked-run saves, the lane-in-progress saves, the renewal and the transcript record all go through that one judge; a drain claims its own lanes first, so a drain resumed after an upgrade takes them up."
impact: fix
resolved_by:
  commit: "840bc2eb930c853d5db4a8e0d6d88a8a43268b93"
---

A step or a drain run by a newer abcd in a checkout where an older build drives runs can save those runs at the newer schema version, and the older driver then refuses its own run files. The renewal, the parked renewal, a drain's stop over a run not among its lanes and a drain's take-up of such a parked run's stop mark now leave such a run alone. Two routes stay open. Under a stop, the step's own stop check (ClaimDriver claims nothing under a stop and the check saves the whole state) and a drain's move over its own lanes settle the run at the newer version. With no stop at all, a drain's move over a parked run with no stop mark, among its lanes or not, saves the whole state at the newer version in four places: finishMerged's landing step on a merged lane, resyncParked, watchParked and handBackClosed. Nothing a run holds tells its own driver's new session from a second session, a newer build stepping an older file is also how a driver resumes after an upgrade, and a new drain, which starts with no lanes, finishing an earlier drain's merged lanes is the ordinary path after one.

## Grounds

- pursued: an older driver keeps reading its own run while a newer session or drain runs beside it; shown wrong if any route still rewrites an older run's file while another live session drives it
