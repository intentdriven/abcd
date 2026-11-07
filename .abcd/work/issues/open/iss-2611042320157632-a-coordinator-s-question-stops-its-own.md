---
schema_version: 1
id: "iss-2611042320157632"
slug: "a-coordinator-s-question-stops-its-own"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "coordinating the build lanes, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/guard_question.go"
remedy: "A session told to run unattended enters the away state the question check already refuses on, rather than the coordinator having to remember not to ask, so a question is recorded and the work carries on; a test takes a coordinator started for unattended work and asserts its question is refused and names where to record it."
---

A coordinator that asks a question through the host's question tool stops all its own progress until the person answers, its wakeups and re-arming included: on 2026-11-04 that left one idle for over an hour while its sub-agents ran on.
