---
schema_version: 1
id: "iss-2609232048579579"
slug: "testpngchunkcountbombisbounded-internal-adapter-scanner"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
---

TestPNGChunkCountBombIsBounded (internal/adapter/scanner/container_test.go) still fails on machine load although iss-2609091215552981 was resolved: the resolution moved the proof to the refusal but kept a plain-run wall-clock runaway guard of 10s against a run of about 1.5s, and at a 1-minute load of about 20 to 40 the plain run took 10.63s and failed while its own result carried the chunk bound's refusal (png: more than 1024 compressed chunks). The resolved record's grounds name exactly this as what would show it wrong: a red on a slow machine where the bound held. Observed in make preflight on the merged tip 7469213f of feat/guard-per-call-workdir. Fix: size the guard as a hang detector far above any plausible loaded run (or route it through the hang-budget helper the wall-clock sweep introduces), so only a runaway reds it; detector: with the chunk bound removed the test must still fail on the missing refusal.
