---
schema_version: 1
id: "iss-2610160924043423"
slug: "when-a-plugin-manifest-sweeps-a"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "v0.15.0 deferred findings, PR1 confirmation review, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/gates.go"
remedy: "Judge a manifest-swept hook entry's mode against the executed-reference union too, or keep the hook-command entry when deduplicating; test a swept directory plus a direct hooks.json run of a 0644 script."
---

When a plugin manifest sweeps a directory of hooks ("hooks": "./hooks/"), each script there becomes a manifest entry that wins dedupeEntries, and launch's hook-compliance gate skips every entry that is not a hook-command entry, so a 0644 script that hooks.json executes directly is never judged. The same at 2347eb6c3 and after the iss-2610160608456302 fix; pre-existing.
