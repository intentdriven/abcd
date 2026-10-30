---
schema_version: 1
id: "iss-2610241537197997"
slug: "concurrent-preflights-on-one-machine-all"
severity: "major"
category: "bug"
source: "agent-observation"
found_during: "drain session abcd-0d, 2026-10-24 autonomous test run"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
remedy: "Make make preflight take the load gate once, for the whole run, the way the gate's command form does (implement load --wait -- make preflight), so whole preflights run one at a time on a machine, with the wait measured and reported; keep the per-binary -timeout as the stage's safety net only."
resolution: "make preflight now takes the load gate once, for the whole run, through the gate's command form: a queued preflight does no lint, eval, build, vet or test work, whole runs are admitted as many at once as load.preflights allows (one by default), and the gate measures and reports the wait. The per-binary test timeout stays as each stage's safety net."
impact: fix
---

Concurrent preflights on one machine all fail: on 2026-10-24 a records-only branch's preflight reached its test stage while six other preflights and several direct test runs shared a 10-core machine at a load near 20, and six packages (cloudscripts, ahoy, autonomy, claim, embargo, implement/loop) hit their 20-minute per-binary timeouts, so the run failed on nothing its own change touched. The load gate admits each preflight's test stage once the load dips under its limit, and the stages before it run ungated, so every admitted run then competes with the rest and none can finish inside its timeouts. Wrapping the whole run in the gate's command form, which one session found by hand, serialises them.
