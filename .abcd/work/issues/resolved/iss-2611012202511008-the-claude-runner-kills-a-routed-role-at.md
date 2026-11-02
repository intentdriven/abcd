---
schema_version: 1
id: "iss-2611012202511008"
slug: "the-claude-runner-kills-a-routed-role-at"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "routing a build's implementer to the claude runner, 2026-11-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/runner/runner.go"
remedy: "Stop naming make preflight as an implementer's definition of done, since the loop runs the preflight itself before land, and give the brief a scoped definition of done (format, vet and the changed packages' tests behind the load gate); bound each routed role by a per-role setting in the machine configuration whose default covers that scoped definition of done, with time waiting at the load gate not counted against it, and tell the role in its brief how long it has; when a bound ends a launch, keep the cost it reported or charge the launch's budget, never lose it; and when a role is killed after it has committed, keep the lane at its stage for a host or runner to finish the receipt rather than counting the kill as a failed start."
resolution: "A routed role is now bounded by the person's roles.<role>.timeout (three hours for the implementer and one for other roles by default), the prompt names the bound, and the brief scopes the implementer's definition of done to what it changed. Not counting time at the load gate, keeping a killed launch's cost, and keeping a lane killed after it committed at its stage are left to iss-2611021751406429."
impact: fix
resolved_by:
  commit: "2730fc890"
---

The claude runner kills a routed role at its default one-hour bound (internal/core/runner/runner.go, DefaultTimeout), and an implementer's work in this repository can outlast it: on 2026-11-01 the routed implementer of run-2611012100266247 committed its fix, wrote its report and, running its definition of done over the whole internal tree after the loop package alone took 1,459 seconds, was killed at 1h0m0s before writing its receipt. The kill counted as a failed start, the role fell back to the host and the lane parked, with finished, tested work stranded behind a missing receipt. The same night two more routed implementers, on itd-39 and itd-2610142145236613, were killed at the hour (22:52 and 22:53 UTC) partway through make preflight, which their briefs named as the definition of done, each with its work committed and its report written, and each fell back to the host. A killed launch reports no result and so no total_cost_usd, so all three launches' spend is missing from every tally. A receipt the host writes after such a kill also loses whatever the implementer meant to resolve beyond the issue it was handed: one killed implementer's report asked its landing to resolve a second issue, the host's receipt left it out, and an accepted receipt cannot gain a resolve, so that issue waits for a separate records change.

## Grounds

- pursued: a routed implementer finishes and writes its receipt within its bound; shown wrong if one is again killed at its bound while running a scoped definition of done
