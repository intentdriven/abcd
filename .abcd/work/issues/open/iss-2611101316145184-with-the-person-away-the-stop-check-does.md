---
schema_version: 1
id: "iss-2611101316145184"
slug: "with-the-person-away-the-stop-check-does"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/autonomy/wake.go"
remedy: "Accept a recurring timer whose next run falls before the deadline as the look without adding the lateness allowance past it, count a one-shot wake that fired after its deadline as the look it was set for, and say when a one-shot wake has been overtaken by a later one; a test holds an hourly recurring timer due in ten minutes against a twenty-minute deadline and finds the stop released."
---

With the person away, the stop check does not take a recurring session timer as the scheduled look it asks for, and does not count a one-shot wake that fired late. qualifyingWake (internal/core/autonomy/wake.go) adds up to 30 minutes of assumed host lateness to a recurring task's next run, so a recurring timer whose next run falls before the deadline is refused once that allowance passes it. On 10 Nov at 12:23 a recurring timer with its next run inside the limit was not accepted and a one-off had to be set; the recurring timer had also survived a session resume, so re-arming it made two. On 7 Nov the one-shot wakes set between 20:04 and 21:35 to meet the check's deadlines did not fire while the session was mid-turn, and by 23:32 seven were queued to fire together at the next idle moment and were deleted by hand. Holding wakes queued while a turn runs appears to be the host's behaviour and outside this record's remedy.

Already fixed, and not part of this record: the past deadlines named on 7 Nov at 20:10 and on 8 Nov at 05:27. The clamp in 2fe25c2e3 fixed them under iss-2611011332417169, landed with 05dfe5816; no release tag yet carries it, so those deadlines came from an installed release without the fix.
