---
schema_version: 1
id: "iss-2610100602595553"
slug: "a-rate-limit-gets-no-wait-once-the-pause"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "pacing's rate-limit checkpoint, itd-2609201925079472 criterion 8"
remedy: "Give the rate-limit checkpoint its own wait, independent of pace.pause_minutes: the reset time the runner reports when it reports one, else a configured or bundled rate-limit wait"
---

A rate limit gets no wait once the pause is zero: pacing's rate-limit checkpoint (itd-2609201925079472) ends the run's window early and sets next_eligible_at to now plus the run's pause, and with the bundled pause now 0 (iss-2610100546338578) the next step hands the same work out again at once, so a rate-limited run can hit the same limit again and again. The product thinker ruled on 2026-10-10 that a rate limit waits on its own, independent of the pause, as work after pacing and the drain (itd-82).
