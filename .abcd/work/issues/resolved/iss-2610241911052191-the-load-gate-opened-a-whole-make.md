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
resolution: "Each test stage of make preflight now gives go test a -timeout of its 40-minute base scaled with the one-minute load per online core as the stage starts, between one and two times and rounded up to a whole minute (cmd/test-parallelism -timeout, machineload.GoTestTimeout), the way its -p is already derived; on a CI runner, or where the machine cannot be read, it stays the base CI's steps declare. Admission on projected headroom, the remedy's other route, stays with draft itd-2610262141001890 (dcn-2610291132223552)."
impact: fix
---

The load gate opened a whole make preflight at load 17.8 on ten cores, under the 20 threshold, yet three packages (internal/core/claim, internal/core/implement/loop, internal/surface/cli) then hit the twenty-minute test timeout, because the other sessions' ungated stages kept the load near the threshold for the whole run

## Grounds

- pursued: a package that ran long only because the machine was loaded now finishes inside its budget; a package killed at a scaled timeout with no test hung, at a load under twice the cores, would show the scaling too weak.
