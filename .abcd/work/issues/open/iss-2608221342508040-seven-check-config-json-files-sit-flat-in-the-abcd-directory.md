---
schema_version: 1
id: "iss-2608221342508040"
slug: "seven-check-config-json-files-sit-flat-in-the-abcd-directory"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "user-observation"
found_at: ".abcd/site.json"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed I): Adopt a config sub-tree convention for the seven check-config JSON files, with an adr-47 amendment?"
remedy: "Waits on ruling I (adopt the config sub-tree convention): if adopted, move the seven root JSON files other than config.json and rules.json into .abcd/config/, beside the five records already there, with each reader trying the new path first and refusing when both exist, amend adr-47 for the site pair and update the .abcd/README.md index, proven by each gate's tests on the moved path and a test of the both-present refusal; if refused, state in .abcd/README.md that check configuration stays at the root and resolve this record as wontfix."
---

seven check-config JSON files sit flat in the abcd directory; the maintainer suggests a config sub-tree, which needs one convention decision covering all of them plus an adr-47 amendment for the site pair

## Remedy grounds (2026-09-29)

- .abcd/config/ already holds five per-surface records (identity.json, launch-payload.json, version-location.json, artefact.json, reading-presets.json), so the convention exists in part and the ruling is whether to finish it; no outside practice was needed.
- Rejected: a move without a read of the old path, which would break managed repositories already carrying positioning.json at the root.
