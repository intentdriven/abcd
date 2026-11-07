---
schema_version: 1
id: "iss-2611060835423048"
slug: "a-run-marked-by-a-stop-since-lifted-then"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "v0.16.2 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/stop.go"
remedy: "Settle such a run against the stop in force, and compose the page from that."
---

A run marked by a stop since lifted, then settled by a signal to the drive, waits for `--take`, and the day's page it writes says the lifted stop holds (`terminal.go:458`, `stop.go:356`, `stop.go:889`). A person reads that a stop stands when none does, and that the run needs a take-over because of it.
