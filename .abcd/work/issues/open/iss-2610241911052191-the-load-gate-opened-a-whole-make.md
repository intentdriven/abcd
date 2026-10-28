---
schema_version: 1
id: "iss-2610241911052191"
slug: "the-load-gate-opened-a-whole-make"
severity: "major"
category: "observation"
source: "user-observation"
found_during: "autonomous run 2026-10-24, gated preflight of chore/capture-holdups-3fee"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
remedy: "Gate admission on headroom the run will need for its length, not on the instant load, or scale each test stage's timeout with the load the way its -p already is."
---

The load gate opened a whole make preflight at load 17.8 on ten cores, under the 20 threshold, yet three packages (internal/core/claim, internal/core/implement/loop, internal/surface/cli) then hit the twenty-minute test timeout, because the other sessions' ungated stages kept the load near the threshold for the whole run
