---
schema_version: 1
id: "iss-2611061904457784"
slug: "a-merge-queue-entry-dropped-at-the-job"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/ci.yml"
remedy: "Raise the check job's ceiling from 45 to 60 minutes, under the queue's 90-minute check timeout, moving the watchdog's cap, the warning line and the ceiling the warning names with it."
resolution: "The check job's ceiling is 60 minutes, with the watchdog's cap and the warning's named ceiling, so the watchdog fires at 55 and the warning at 50, all under the queue's 90-minute check timeout. This buys room and takes no time out: the longer lane remains, and internal/core/implement/loop is its largest package at about 1400 to 1500 seconds under race, so the time itself is still owed (iss-2610181757540736 holds that work)."
impact: internal
resolved_by:
  commit: "a084398cc0945e5663322653bd358b158e16aa1d"
---

A merge-queue entry dropped at the job clock reads as a failed check although no test failed. In check (ubuntu-latest) the job clock's watchdog fires at 40 minutes, the 45-minute cap less its 5-minute margin, and ends the tests with nothing failed (run 42269270911, job 128069397267: about 4 minutes before the race stage, over 36 inside it). The cap was tight against queue runs that reach about 40 minutes without failing, while the queue's own check timeout is 90 minutes.
