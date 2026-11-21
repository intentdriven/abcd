---
schema_version: 1
id: "iss-2611190439574731"
slug: "make-preflight-prints-no-wall-time-for"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "autonomous run of 15 Nov 2026, measuring step 3 of spc-2611012047503720"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
remedy: "Have make preflight print each stage's elapsed time and the whole run's on its last lines, beside the reuse counts, with a test of the report format; the measurement is then completed in the research note from ten consecutive landings on the person's machine, against a baseline time taken from one full check of the same tree with reuse off."
resolution: "make preflight now prints each stage's time and the whole run's checked time after its receipt line, with the waits at the load gate, the race gate and the check directory's lock kept apart; the run's clock lives under the local tier's preflight-clock/"
impact: internal
---

make preflight prints no wall time, for its stages or the whole run, so the saving criterion of itd-2611011325421304 (ten landings' reuse share and check time against the 2026-11-01 baseline) can record the reuse share but not the check time: the first ten reports after step 2 show 1010 of 1424 package runs reused, and no log says how long any check took.
