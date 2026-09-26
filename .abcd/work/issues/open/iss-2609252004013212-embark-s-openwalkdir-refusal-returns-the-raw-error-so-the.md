---
schema_version: 1
id: "iss-2609252004013212"
slug: "embark-s-openwalkdir-refusal-returns-the-raw-error-so-the"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lifeboat/embark.go"
---

embark's openWalkDir refusal returns the raw error, so the fatal line reads 'openat <name>/.: not a directory' and leaks the '/.' suffix into a user-facing refusal naming a path that does not exist (internal/core/lifeboat/embark.go:622; review-lifeboat 2). Wrap it with the entry name.
