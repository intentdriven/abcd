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
---

A merge-queue entry dropped at the job clock reads as a failed check although no test failed. In check (ubuntu-latest) the job clock's watchdog fires at 40 minutes, the 45-minute cap less its 5-minute margin, and ends the tests with nothing failed (run 42269270911, job 128069397267: about 4 minutes before the race stage, over 36 inside it). The cap was tight against queue runs that reach about 40 minutes without failing, while the queue's own check timeout is 90 minutes.
