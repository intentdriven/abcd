---
schema_version: 1
id: "iss-2610161756384596"
slug: "abcd-s-load-gate-admits-a-fixed-number"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "product thinker, 2026-10-16 evening"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/machineload"
remedy: "Derive the admission counts from measured resources: online cores, free memory and the one-minute load, together with the measured cost of recent runs (each preflight's peak load and memory, and each race stage's, recorded by the gate when it closes). Admit another heavy stage only while the projected load stays under the wait limit, with a person's user-layer setting able to lower the derived number but never raise it past what memory allows. Measure first on this machine with the two-preflight trial, and research the state of the art (CI runner autoscaling, make -l load-limited jobs, Bazel's local resource estimation) before choosing the model. Relates to the sub-agent budget intent itd-2610100704270254 and to iss-2610161639025702."
---

abcd's load gate admits a fixed number of preflights, one, whatever the machine can carry. The product thinker asked on 2026-10-16 that abcd know how many concurrent preflights, race runs and sub-agents are feasible from the resources it actually has, instead of a number a person sets and re-sets by hand (the machine-wide cap moved six times in two weeks: six, eight, ten, twelve, ten, fourteen).
