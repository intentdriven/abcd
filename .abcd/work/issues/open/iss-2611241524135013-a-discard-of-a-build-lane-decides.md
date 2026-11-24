---
schema_version: 1
id: "iss-2611241524135013"
slug: "a-discard-of-a-build-lane-decides"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/discard_unheld.go"
remedy: "Record the commit sha that each reservation, renewal and resume claim returns in the run's state, and match the discard's anchor on that sha, keeping the time match as the fallback for states written before the field existed."
---

A discard of a build lane decides whether origin's reservation is the run's own by matching the second a reservation commit was claimed in against the times the run logged. The match is sound only while every logged act pushed exactly one commit, and two fixes in November had to restore that. Any later site that logs an act without pushing breaks it again with no error.
