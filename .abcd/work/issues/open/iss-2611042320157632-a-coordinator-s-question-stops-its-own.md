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
remedy: "An unattended coordinator records the question and carries on, and the question check in abcd guard hook refuses the host's question tool in that mode; a test judges a question from an unattended session and asserts the refusal names where to record it."
---

A coordinator that asks a question through the host's question tool stops all its own progress until the person answers, its wakeups and re-arming included: on 2026-11-04 that left one idle for over an hour while its sub-agents ran on.
