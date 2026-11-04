---
schema_version: 1
id: "iss-2611021751406429"
slug: "a-routed-role-killed-at-its-bound-loses"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "fixing iss-2611012202511008, 2026-11-02"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/runner/runner.go"
remedy: "Pause the runner's clock while its process tree waits at abcd's load gate (the waiter's hold names the waiting pid), or extend the deadline by the time the gate reports waiting; on a kill, read the cost from the run's stored transcript or charge the launch's budget, never drop it; and when a killed role has commits on the lane's branch past its base and a report, keep the lane at its stage for a host or runner to write the receipt instead of counting a failed start."
resolution: "A routed role's clock now stops while its process tree holds a waiter queued at the load gate (credited up to its bound again, the claim outlasting that). A killed launch keeps what its transcript shows it spent, or is charged one agent run, on its fallback receipt and in the run's tally (run state v19, drain state v3). An implementer killed with commits past its base and its report written keeps its lane at its stage for its receipt instead of counting a failed start."
impact: fix
resolved_by:
  commit: "ed40ef462702a8049795fc55f1397d5708b08fba"
---

A routed role killed at its bound loses three things the bound alone cannot save: time it spent waiting at the load gate counts against the bound, so a run queued behind other preflights is killed sooner than its work warrants; a killed launch reports no result and so no total_cost_usd, so its spend is missing from every tally; and a role killed after it committed its work counts as a failed start, falls back and parks the lane, stranding committed, tested work behind a missing receipt. iss-2611012202511008 bounded each role by its own timeout and scoped the implementer's definition of done; these are the parts of its remedy left over.

## Grounds

- pursued: a role queued at the load gate outlives its bound by its wait, a killed launch's fallback receipt carries spent, and a killed implementer with committed work and a report leaves its lane unpaused with no fallback; a kill at the bound while a waiter sits queued under the harness, a fallback receipt with no spent after a kill, or a paused lane after such a kill would show it wrong
