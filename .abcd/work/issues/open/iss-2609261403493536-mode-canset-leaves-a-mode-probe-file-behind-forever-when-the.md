---
schema_version: 1
id: "iss-2609261403493536"
slug: "mode-canset-leaves-a-mode-probe-file-behind-forever-when-the"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/mode/question.go"
---

mode.CanSet leaves a .mode-probe-* file behind forever when the local tier turns unwritable between the probe's create and its remove (a chmod during the hook's window, or a kill between the two): CanSet reports 'removing the probe' once, the file stays, and nothing ever sweeps it. Review2 of the badge lane reproduced it: 40 probes against a chmod-flapping tier left 7 probe files. Remedy: sweep stale probe files at the top of CanSet.
