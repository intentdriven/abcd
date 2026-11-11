---
schema_version: 1
id: "iss-2611110638266241"
slug: "a-walled-drive-s-stage-cannot-widen-its"
severity: "critical"
category: "bug"
source: "agent-observation"
found_during: "autonomous run, live drive of 11 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/widen.go"
remedy: "Take the widening out of the stage: after the implement stage and before validation the loop itself, which holds the credential, reserves the paths the lane's commits change beyond the held footprint (the remedy iss-2611101316537446 names), the brief stops telling a walled stage to run widen, and a widening that backs off another holder's live hold is a wait by design, never three failed launches and a stuck tombstone. A test drives a lane that commits a file outside its found_at with no credential in the stage and finds the path reserved before validation."
blocked_by: [iss-2611101316537446]
resolution: "The loop, which holds the credential, now reserves every path a lane's branch changes beyond its hold: when it reads a stage's receipt, before each validator step, and at the push and arming fences. A path another copy holds is a wait bounded by the lane's minutes limit, not a refusal. The stage brief no longer tells a stage to run the widen command. A drive started from a build that holds this change no longer hands such lanes back as stuck; a drive already running on an older build keeps the fault until it is restarted."
impact: fix
resolved_by:
  commit: "c41bee2c8707af89b9a321a0a86fa81134416583"
---

A walled drive's stage cannot widen its lane's hold, so every lane whose fix needs a path beyond the issue's found_at is lost. The implementer's brief tells it to run abcd implement widen before changing a path outside the hold; widen reserves by pushing to origin, and a drive's stage runs behind the wall with no credential for origin by design, so the push fails (could not read Username, terminal prompts disabled) and no reservation is made. The implementer then stops with no receipt, as the brief says, the loop reads that as a failed launch, starts it twice more, and hands the issue back as stuck with a tombstone that keeps every later drain off it. On 11 Nov 2026 a live drive lost two lanes in a row this way within an hour of landing its first, at three implementer launches each, one of them over thirty minutes long. Only a lane whose whole fix sits in its found_at path gets through, which is the exception.
