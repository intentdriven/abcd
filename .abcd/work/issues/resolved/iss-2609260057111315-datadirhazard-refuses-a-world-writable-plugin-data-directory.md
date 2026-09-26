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
resolution: "dataDirHazard applies fsutil.CallersAlone (extracted from ReadDeclaration, which now calls it too) to the data directory and its cache/, refusing group-writable and foreign-owned directories as well as world-writable ones."
impact: fix
resolved_by:
  commit: "1aabbfbe"
---

dataDirHazard refuses a world-writable plugin data directory or cache subdirectory, but not a group-writable one or one owned by another uid, so on a shared host a group member or another account can still supply the cache artefact and its recorded hash the hazard check claims to rule out. It should apply the caller-alone test the home-scoped declarations use (not writable by group or other, owned by this uid).

## Grounds

- pursued: only a directory the caller alone can write supplies the owned PATH copy; TestDataDirHazardRefusesAnyoneElsesDirectory and TestInstallIgnoresGroupWritableDataCache would fail if a 0775 or foreign-owned data dir were accepted, and the declaration tests would fail if the extraction changed ReadDeclaration
