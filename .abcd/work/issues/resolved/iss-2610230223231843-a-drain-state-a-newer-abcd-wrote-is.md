---
schema_version: 1
id: "iss-2610230223231843"
slug: "a-drain-state-a-newer-abcd-wrote-is"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "review pair of the run-state schema-15 bump, 2026-10-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/drain.go"
remedy: "Read the drain state's schema_version loosely first and refuse a version above this abcd's as a newer abcd's file, telling the person to run the newer abcd and not to remove the drain state, before any strict decode, as decodeState does for the run state"
resolution: "readDrain refuses a drain state of a later schema version as a newer abcd's before any strict decode, telling the person to run the newer abcd and not to remove the file, as the run state does"
impact: fix
resolved_by:
  commit: "1cba486c1"
---

A drain state written by a newer abcd is refused as a file that does not parse, with a remedy that throws the drain away. readDrain (internal/core/implement/loop/drain.go) decodes the drain state strictly and refuses any schema_version other than its own, so a newer abcd's drain state (a later drain schema, or a field a newer build adds to a stored route) reads as "does not parse as the drain's state" with the remedy "restore it or remove it", which deletes the drain record of the newer run. The run state already refuses a newer writer's file before any strict decode, naming it a newer abcd's and saying not to remove it (decodeState); the drain state has no such check.
