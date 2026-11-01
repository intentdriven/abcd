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
remedy: "Carry the runner's own report of why it failed into the failure's detail and the fallback record: for claude, the result line of its stream-json output on stdout (where 'Not logged in' appears; stderr is empty then), else the last lines of stderr; bound it, pass it through the history store's redactor and refuse to carry a span the redactor blocks, update the invariant at internal/core/runner/receipt.go:52-53 that a fallback's detail carries no harness output in the same change, and name the common cause when the runner reports it is not logged in (no API key in its environment)."
---

When a role routed to a runner fails, the loop records the fallback with only the runner's exit status ('2.1.316 exited with status 1', internal/core/runner/proc.go) and leaves out what the runner reported (the stored transcript keeps it, the fallback's detail does not), so the cause is invisible where a person looks: on 2026-11-01 the routed implementer of run-2611012037155265 failed because claude --bare found no API key, and claude reported 'Not logged in' as the result line of its stream-json output on stdout, with stderr empty, which only a hand run of the same command showed. The fallback then parked the lane, since a fallback is final.
