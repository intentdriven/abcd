---
schema_version: 1
id: "iss-2610111007107579"
slug: "window-clock-and-series-state-twice"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "build next lane review, run-2610110908001704, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/drain.go"
remedy: "Once itd-82 ships, lift one window-clock type (start, elapsed, next eligible, pause) and one locked state-file read/write helper into loop/, port drain.go and nextseries.go onto them, and keep each file's JSON shape unchanged; abcd-a5 reviews the drain side."
---

The pace window clock and the locked JSON state file beside the runs exist twice in the implement loop: loop/drain.go keeps drain.json under drain.lock with window_started_at and next_eligible_at, and build next's series (loop/nextseries.go, landing with itd-2609211116005482) keeps next.json under next.lock with the same fields and the same window logic, copied because drain.go is mid-change under itd-82. Two copies is the point the one-canonical-primitive principle files it.
