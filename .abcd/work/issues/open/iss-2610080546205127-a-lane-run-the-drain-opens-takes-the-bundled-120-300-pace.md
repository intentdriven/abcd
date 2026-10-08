---
schema_version: 1
id: "iss-2610080546205127"
slug: "a-lane-run-the-drain-opens-takes-the-bundled-120-300-pace"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "overnight drain, 2026-10-08"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/drain.go"
remedy: "When the drain opens a lane's run, pass the drain's own pace (--pace work/pause), sub-agent ceiling and fix rounds into the run's state, recorded with the drain as their origin, so a lane never falls back to the bundled 120/300 pace and 2 sub-agents."
---

A lane run the drain opens takes the bundled 120/300 pace and 2 sub-agents rather than the drain's --pace and --sub-agents, so a continuous drain (--pace 10080/0 --sub-agents 6) pauses its lane after two hours: run-2610080105040584 paused until 08:07Z overnight and was set to the drain's pace by hand in its state.
