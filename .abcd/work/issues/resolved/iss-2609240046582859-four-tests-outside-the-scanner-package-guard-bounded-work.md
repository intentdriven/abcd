---
schema_version: 1
id: "iss-2609240046582859"
slug: "four-tests-outside-the-scanner-package-guard-bounded-work"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "autonomous run A, gallop round 3 sibling sweep"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/speculate_bound_test.go"
deferred_after: "v0.9.0"
deferral_reason: "The four ceilings sit outside the scanner package this lane converted; each needs its own count seam and regression proof, so they are taken as their own lane next cycle rather than widened into this one."
resolution: "the guard's two ceilings (speculate_bound_test.go, brace_test.go) became counts of work in a29b1cabb; the reading package's two (project_test.go) now bound the process's CPU time from getrusage, which load cannot inflate, pinned by TestTheLinearityClockDoesNotChargeWaiting"
impact: internal
resolved_by:
  commit: "8f450fdb7"
---

Four tests outside the scanner package guard bounded work with a tight wall-clock ceiling, the shape that made the scanner's cost guards fail gates under machine load: internal/core/reading/project_test.go (10s near line 89 and 5s near line 112), internal/core/guard/speculate_bound_test.go (3s near line 108, the tightest) and internal/core/guard/brace_test.go (5s near line 190). Each should assert a load-independent count of work, as the scanner's guards now do, or carry a ceiling generous enough that load cannot trip it. Found by the gallop lane's sibling sweep.

## Grounds

- pursued: a gate run on a loaded machine no longer fails these four guards on linear code, while a quadratic regression still charges its seconds; one of them failing on a busy machine with no code change would show it wrong
