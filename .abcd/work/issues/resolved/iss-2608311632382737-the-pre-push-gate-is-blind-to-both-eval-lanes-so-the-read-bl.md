---
schema_version: 1
id: "iss-2608311632382737"
slug: "the-pre-push-gate-is-blind-to-both-eval-lanes-so-the-read-bl"
severity: "major"
category: "process"
source: "impl-review"
found_during: "reviewing the evals audit fix round"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
resolution: "make preflight now runs both eval lanes as prerequisites (smoke, evals-cold-reading), so the tagged eval files the untagged test run cannot compile are executed before a push; TestPreflightRunsBothEvalLanes reads the lanes off the recipe line, and the derived gate-list test propagates the change to every surface that restates it."
impact: internal
---

The pre-push gate is blind to both eval lanes, so the read-block eval is guarded by nothing that blocks anything. make preflight runs the six lint gates, go build, go vet, go test over the untagged packages and the race lane, and neither make smoke nor make evals-cold-reading; the eval files sit behind a build tag, so the untagged test run does not compile them. A defect in the cold-reading evals therefore passes every local gate and surfaces only in CI. That was not hypothetical: a path-elision defect in the amnesia eval's own guard was unsatisfiable wherever the process temp directory is the Linux one, so it would have landed green locally and red in the merge queue, and it was found by an adversarial review rather than by any gate. Pair this with the CI job not being a required status check and the position is that the eval which certifies the firewall runs in no gate that can stop anything: preflight does not execute it, and the job that does execute it cannot block a merge.

## Grounds

- pursued: a defect in the read-block or amnesia eval now fails locally before it reaches CI; it would be shown wrong by a preflight run that is green while make evals-cold-reading or make smoke is red.
