---
schema_version: 1
id: "iss-2611012049508108"
slug: "when-a-role-routed-to-a-runner-fails-the"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "routing a build's implementer to the claude runner, 2026-11-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/runner/proc.go"
remedy: "Carry the last lines of the runner's stderr, bounded and redacted, into the failure's detail and the fallback record, and name the common cause when the runner says it is not logged in (no API key in its environment), so the person sees why the role fell back to the host."
---

When a role routed to a runner fails, the loop records the fallback with only the runner's exit status ('2.1.316 exited with status 1', internal/core/runner/proc.go) and drops what the runner wrote on stderr, so the cause is invisible: on 2026-11-01 the routed implementer of run-2611012037155265 failed because claude --bare found no API key and answered 'Not logged in', which only a hand run of the same command showed. The fallback then parked the lane, since a fallback is final.
