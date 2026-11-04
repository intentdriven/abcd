---
schema_version: 1
id: "iss-2611040712378570"
slug: "abcd-implement-load-with-no-site-exits-2"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "the v0.16.1 cut, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/implement_load.go"
remedy: "Word the refusal for a missing flag separately and name the two sites; a test runs the bare command."
---

abcd implement load with no --site exits 2 and says the value given is not repeated here, when no value was given.
