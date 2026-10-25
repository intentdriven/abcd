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
---

make preflight's smoke lane runs before the load gate under go test's default ten-minute timeout, and at a one-minute load of 18 on ten cores the whole evals package ran out of time (TestEveryCommittedEntryFitsItsDeclaredWindow among the tests still running), failing a records-only branch's preflight after about twenty minutes
