---
schema_version: 1
id: "iss-2610241524383006"
slug: "make-preflight-s-smoke-lane-runs-before"
severity: "major"
category: "observation"
source: "user-observation"
found_during: "autonomous run 2026-10-24, preflight of chore/capture-holdups-3fee"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
remedy: "Run the smoke and cold-reading eval lanes behind the same load gate as the test stages, or give them an explicit timeout scaled like the test stages' -p, so a loaded machine delays them instead of failing them."
refines: [iss-2610161639025702]
resolution: "The same defect as iss-2610241228178135, fixed on this branch: make smoke and make evals-cold-reading run go test with -timeout $(EVAL_TIMEOUT), 20m by default, each CI and release eval step sets a budget that fits its job after the steps before it, and evallanebudget_test.go holds both"
impact: internal
---

make preflight's smoke lane runs before the load gate under go test's default ten-minute timeout, and at a one-minute load of 18 on ten cores the whole evals package ran out of time (TestEveryCommittedEntryFitsItsDeclaredWindow among the tests still running), failing a records-only branch's preflight after about twenty minutes
