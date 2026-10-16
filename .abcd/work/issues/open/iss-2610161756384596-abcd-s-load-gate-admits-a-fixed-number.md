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

## Refinement 2026-10-16 (product thinker): staggered admission sized to the machine

The product thinker proposed a staggered approach. Two runs execute while two more wait, and the number running at once (1, 2, 3 …) follows the machine: more on a fast machine, one at a time on a slow one. That fits this record's model as a pipeline with one capacity per stage:

- **Cheap stages run freely.** Format, lint, records and build use a few cores and mostly wait on files.
- **Unit test stages are admitted up to N.** N is derived from cores, memory and the measured cost of recent stages.
- **Race stages are admitted up to M.** M is 1 on most machines. On a large one it is derived the same way.
- **Live load limits both N and M**, so a machine busy with other work admits fewer, and a quiet one admits more.

A caveat from today's evidence: much of the gate wait came from load above the limit caused by the operating system (Spotlight re-indexing and the VPN extension), not from preflights. More concurrency cannot shorten that part. Admission has to tell its own load from the machine's other load, and the measurement must report the two separately.
