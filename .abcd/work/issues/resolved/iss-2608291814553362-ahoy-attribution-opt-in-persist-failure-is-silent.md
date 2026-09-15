---
schema_version: 1
id: "iss-2608291814553362"
slug: "ahoy-attribution-opt-in-persist-failure-is-silent"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: "internal/core/ahoy/attribution_hook.go"
resolution: "recordAttributionOptIn appends a change-note on the read and write error paths, as registerRepo does; a test drives a malformed config.json through install --attribution and asserts the note"
impact: fix
---

ultra-v0.6.8 A1: recordAttributionOptIn in internal/core/ahoy/attribution_hook.go returns silently when readConfig or writeConfig fails, so ahoy install --attribution writes the prepare-commit-msg hook to disk but attribution.hook: true never lands in config.json and the receipt still reports success. attributionOptedIn then returns false, no gap is raised, and every later plain install is a no-op for the hook — the silent degradation the opt-in exists to remove. Fix: append a change-note on each error path exactly as registerRepo does for its history-lock failure.
