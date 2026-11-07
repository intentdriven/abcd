---
schema_version: 1
id: "iss-2611051719396028"
slug: "a-status-store-that-cannot-be-written"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "build run, 5 November"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/hook_scheduled.go"
remedy: "Say that the stamp failed on the prompt it failed on."
---

A status store that cannot be written, from a full disk or a changed mode, means a due prompt starts no refresh and says nothing, so a kept green can stay silent for up to 48 hours.
