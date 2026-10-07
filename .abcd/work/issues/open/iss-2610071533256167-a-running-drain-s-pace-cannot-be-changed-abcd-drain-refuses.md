---
schema_version: 1
id: "iss-2610071533256167"
slug: "a-running-drain-s-pace-cannot-be-changed-abcd-drain-refuses"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "a /abcd:drain run, 2026-10-07"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/drain.go"
remedy: "Give the drain a verb to change its pace mid-run (abcd drain pace <work>/<pause> [--sub-agents n]), recorded in the drain state with the time and origin of the change and applied from the next move, or at least abcd drain stop, which ends the drain after the lane in progress and records why, so a new drain can start on another pace without hand-editing local state."
---

A running drain's pace cannot be changed: abcd drain refuses --pace, --sub-agents and --fix-rounds while a drain is in progress ('a pace is set when a run starts'), and there is no verb to end a drain early, so the only way to move a drain started on the bundled 120/300 pace (two hours of work, five of pause) to continuous work is to wait for it to end or to discard its local state by hand. Met 2026-10-07 when the product thinker asked a drain started minutes earlier to work throughout.
