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
remedy: "Stop naming make preflight as an implementer's definition of done, since the loop runs the preflight itself before land, and give the brief a scoped definition of done (format, vet and the changed packages' tests behind the load gate); size a routed role's time bound from the work it is handed rather than one fixed hour, and tell the role in its brief how long it has; and when a role is killed after it has committed, keep the lane at its stage for a host or runner to finish the receipt rather than counting the kill as a failed start."
---

The claude runner kills a routed role at its default one-hour bound (internal/core/runner/runner.go, DefaultTimeout), and an implementer's work in this repository can outlast it: on 2026-11-01 the routed implementer of run-2611012100266247 committed its fix, wrote its report and, running its definition of done over the whole internal tree after the loop package alone took 1,459 seconds, was killed at 1h0m0s before writing its receipt. The kill counted as a failed start, the role fell back to the host and the lane parked, with finished, tested work stranded behind a missing receipt.
