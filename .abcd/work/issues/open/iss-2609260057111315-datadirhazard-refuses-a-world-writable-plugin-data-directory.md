---
schema_version: 1
id: "iss-2609260057111315"
slug: "datadirhazard-refuses-a-world-writable-plugin-data-directory"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/data_dir.go"
---

dataDirHazard refuses a world-writable plugin data directory or cache subdirectory, but not a group-writable one or one owned by another uid, so on a shared host a group member or another account can still supply the cache artefact and its recorded hash the hazard check claims to rule out. It should apply the caller-alone test the home-scoped declarations use (not writable by group or other, owned by this uid).
