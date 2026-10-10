---
schema_version: 1
id: "iss-2610100546338578"
slug: "a-paced-run-pauses-300-minutes-after-every-120-minute-window"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/pace.go"
remedy: "Set BundledPauseMinutes to 0 so the bundled pace is 120/0; a repository or machine that wants a pause sets pace.pause_minutes"
resolution: "the bundled pause is 0 (BundledPauseMinutes), so a run with no pace configured never pauses; a repository or machine sets pace.pause_minutes for one"
impact: fix
---

A paced run pauses 300 minutes after every 120-minute window by default: the bundled pace is 120/300, so an implement run that outlives its first window stops every stage, landing moves included, for five hours unless a flag or a configuration layer sets the pause. The product thinker ruled on 2026-10-10 that no pause is needed and that the default must be zero.
