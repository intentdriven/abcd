---
schema_version: 1
id: "iss-2611060835328376"
slug: "a-launch-failure-with-nothing-spent-that"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "v0.16.2 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/runner/runner.go"
remedy: "Count free launch failures per lane and hand the lane back at a limit."
---

A launch failure with nothing spent that names a 5xx or 403, or runs out of retries, is graded BlameNone (`internal/core/runner/runner.go:331`), so no probe asks. If such a failure ever belongs to one lane alone, the queue parks on it until the bound, moving nothing; no observed input does this yet.
