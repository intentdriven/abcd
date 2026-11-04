---
schema_version: 1
id: "iss-2611040337082034"
slug: "a-review-receipt-binds-one-exact-head-so"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "the v0.17.0 cut and the overnight lanes, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/land.go"
remedy: "Let a review receipt name the head it read and the earlier receipted head its delta starts from, so land accepts a chain; hold a preflight until the last review round where a lane expects changes; a test records a chain and refuses a gap in it."
---

A review receipt binds one exact head, so a one-line fix after a review means the whole review pair again, and a preflight already running is thrown away: on 2026-11-03 and 04 the CI split went through four rounds and its preflight was stopped three times. Reviewing a small delta costs little, but the receipt has no way to say that a delta was what was read.
