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
resolution: "Every save of a run's whole state goes through one writer, which replaces a file that states another version only with the right the judge grants the live session the run's control record names as its driver; any other session's save is refused with nothing written, the refusal naming the file, both versions, why and the way on. A claim is how a session becomes that driver: a step, a take-over, a resumed build and a drain over a run it takes up each claim a run whose driver is gone or that names none, so a driver resumed after an upgrade, and a new drain finishing the merged lanes an earlier drain left, save the run at this version, while a run another live session drives is left to it, a drain naming the lane once each move and going on. A claim, a cleared stop mark, the stop's page note and a drive's release add only lines to the run's record and keep the version its file states. The judge tells sessions apart by their host process, so a newer abcd run under the driver's own session counts as that driver. A drain's claim on a parked run that is not one of its lanes lasts for one save and is let go once the save is made."
impact: fix
resolved_by:
  commit: "2d2957b58ec91803802c62b93f295b24ae61dd0e"
---

A step or a drain run by a newer abcd in a checkout where an older build drives runs can save those runs at the newer schema version, and the older driver then refuses its own run files. The renewal, the parked renewal, a drain's stop over a run not among its lanes and a drain's take-up of such a parked run's stop mark now leave such a run alone. Two routes stay open. Under a stop, the step's own stop check (ClaimDriver claims nothing under a stop and the check saves the whole state) and a drain's move over its own lanes settle the run at the newer version. With no stop at all, a drain's move over a parked run with no stop mark, among its lanes or not, saves the whole state at the newer version in four places: finishMerged's landing step on a merged lane, resyncParked, watchParked and handBackClosed. Nothing a run holds tells its own driver's new session from a second session, a newer build stepping an older file is also how a driver resumes after an upgrade, and a new drain, which starts with no lanes, finishing an earlier drain's merged lanes is the ordinary path after one.

## Grounds

- pursued: an older driver keeps reading its own run while a newer session or drain runs beside it; shown wrong if any route still rewrites an older run's file while another live session drives it
