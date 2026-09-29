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
resolution: "walkLifeboatFilesBounded wraps an openWalkDir failure with walkOpenRefusal, which names the entry by its lifeboat path and keeps only the cause, so the refusal no longer leaks the descent's /. suffix; the probe's walk, the other caller, discards the error."
impact: fix
resolved_by:
  commit: "e4cf40ecb"
---

embark's openWalkDir refusal returns the raw error, so the fatal line reads 'openat <name>/.: not a directory' and leaks the '/.' suffix into a user-facing refusal naming a path that does not exist (internal/core/lifeboat/embark.go:622; review-lifeboat 2). Wrap it with the entry name.

## Grounds

- pursued: we expect an embark refusal for an unopenable directory to name the lifeboat path and never the /. suffix; a refusal quoting openat and a /. path would show it wrong
