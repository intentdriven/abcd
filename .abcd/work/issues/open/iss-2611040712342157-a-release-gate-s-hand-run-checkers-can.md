---
schema_version: 1
id: "iss-2611040712342157"
slug: "a-release-gate-s-hand-run-checkers-can"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "the v0.16.1 cut, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/release-gate/README.md"
remedy: "Give the gate a command that takes the checkers' results and refuses a receipt whose items do not cover the manifest, naming each one missing."
---

A release gate's hand-run checkers can silently leave items unchecked: on 2026-11-04 a brief saying group N was read as 0-based by some and 1-based by others, so two groups checked a neighbour's items and 14 of 50 items had no checker; only a coverage check against the plan showed it. Nothing in the procedure requires that check before a receipt is written.
