---
schema_version: 1
id: "iss-2611060835285748"
slug: "a-race-stage-that-had-to-wait-for-the"
severity: "minor"
category: "tech-debt"
source: "agent-observation"
found_during: "v0.16.2 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/testreuse/run.go"
remedy: "Keep the gate's admission until the stage holds the lock again, or say in the code why the gap is accepted."
---

A race stage that had to wait for the check directory's lock passes the race gate again holding no lock (`internal/testreuse/run.go:224`), so another queued race stage can start in the gap before the first retakes it. Two race stages then overlap and load the machine, though no result is affected.
