---
schema_version: 1
id: "iss-2610110918418459"
slug: "landing-never-marks-step-landed"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "fidelity audit of itd-2609212103565953, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "On each lane's landing, before the push, add the '- landed: <pull request or commit>' line under the lane's spec step in the open spec on the lane's branch (the format spec/steps.go parses), committed with the landing's records; test that a three-step run whose lane 2 is handed back leaves steps 1 marked and a --remainder close carries only steps 2 and 3."
resolution: "The landing marks the lane's spec step landed in its records commit, naming the judged head, before any close moves the spec."
impact: fix
resolved_by:
  commit: "850af4cf2107292bc4ff3299be76cf35c9e245c6"
---

The implement loop's landing never writes a spec step's '- landed:' line, so a spec the loop landed part of still reads those steps as unlanded: a later 'spec close --remainder' carries steps that already merged into the remainder, and a fresh 'abcd build' of the same spec opens a lane for them again. Found by the fidelity audit of itd-2609212103565953 (receipt rcp-2257cc3a38d9, criterion ac-3 MET_WITH_CONCERNS); spc-2609202134338445's Progress already says the landing does not mark a non-closing lane's step.

## Grounds

- pursued: a spec the loop lands part of reads its landed steps as landed, so a later build opens its first lane on the first unlanded step; shown wrong if a build of a part-landed spec opens a lane for a step already on the default branch

## Further cases, 2026-10-23

While declaring needs lines on the open specs (2026-10-23), these open specs were found with steps whose code is on the default branch but whose `landed:` line is absent, so a fresh build of the spec would open a lane for each again:

- spc-2609212015054359 (drain ledger triage), steps 1 to 3: 80fa3da48, 378348a14, 91765d9c5, which the spec's own Progress names.
- spc-2610021456200324 (the consistency pass), steps 1 to 3: 5887d3fdb, ec7156cfc, ec768a4d1.
- spc-2610031844142274 (the board at the terminal's width), steps 1 to 4: 92722ff92, ca821c7fa, 317f6bf70, 7995f892b.
- spc-2610150728396006 (sessions announce), steps 1 to 4, through pull request 1065.
- spc-2610040741034208 (the dashboard), step 1: 77ca9b27a.
- spc-2610111627213138 (memory recall), steps 1 and 2: c8d1ab0df, 59b69f398.

The remedy stands for lanes still to land. These existing cases also need a one-off backfill of their `landed:` lines, each checked against its commit, before any of these specs is built again.
