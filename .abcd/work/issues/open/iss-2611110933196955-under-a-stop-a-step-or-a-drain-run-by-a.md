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
remedy: "Record in the run state which checkout and account drives the run, apart from the driver's process, and have every stop-time save by a session that is not that driver judge the version the file states first, as the renewal does; keep the upgrade-resume test green and add an on-disk test per route."
---

Under a stop, a step or a drain run by a newer abcd in a checkout where an older build drives runs saves those runs at the newer schema version, and the older driver then refuses its own run files. The renewal, the parked renewal, a drain's stop and a drain's parked-run note now leave such a run alone; still open are the step's own stop check (ClaimDriver claims nothing under a stop and the check saves the whole state), a drain's move over its own lanes, and finishMerged's landing step on a parked run that is not the drain's lane. Nothing a run holds tells its own driver's new session from a second session, and a newer build stepping an older file is also how a driver resumes after an upgrade.
