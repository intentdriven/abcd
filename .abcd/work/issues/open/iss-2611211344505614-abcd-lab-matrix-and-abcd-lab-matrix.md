---
schema_version: 1
id: "iss-2611211344505614"
slug: "abcd-lab-matrix-and-abcd-lab-matrix"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "build run-2611202324382832 lane-5 (spc-2610161949559050 step 8)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/lab_matrix.go"
remedy: "launch a runner cell's copy from the run (the loop's trial mode per cell), score it on its hidden tests run by abcd rather than the host's report, and wire the local server, the judging session and command-line check judging, as the spec's further steps settle them (dcn-2611211517021229)"
deferred_after: v0.17.0
deferral_reason: "Deferred for one cycle by dcn-2611231314505823: every step of spc-2610161949559050 has landed, and the remaining criteria need design the spec does not settle (dcn-2611211517021229). Owed: a planning pass on the remaining criteria of itd-2610161733316518 before the next cut."
---

abcd lab matrix and abcd lab matrix rerun hand off a consented run's host copies and collect them through lab matrix copy, review, judge and report, but a runner cell's copy is never launched: it waits as awaiting its runner, and the report names it. A host copy's hidden-test score is the host's report, marked as such, and the local server, the judging session and command-line check judging are not called from the command line, so no run has every cell scored by abcd itself
