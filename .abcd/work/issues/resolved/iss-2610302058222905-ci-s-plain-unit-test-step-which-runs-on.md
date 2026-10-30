---
schema_version: 1
id: "iss-2610302058222905"
slug: "ci-s-plain-unit-test-step-which-runs-on"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "merge-queue failure of a records-only pull request, 2026-10-30"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/ci.yml"
remedy: "Give the plain unit step the same declared -timeout make preflight's unit stage uses, and have the step's pinning test hold the two to one budget"
resolution: "CI's plain unit step now runs under -timeout 40m, the base make preflight's unit stage scales from, and TestPlainTestStepRunsTheWholeModule holds the two to one budget. The test failed before the change. The growth in the loop package's time is iss-2610302058287983"
impact: internal
---

CI's plain unit test step, which runs on a records-only or docs-only change, sets no -timeout, so each package inherits go test's ten-minute default. The implement loop package's plain time on the ubuntu runner has grown from 466s to 593s across recent runs and crossed 600s on 2026-10-30 with no test hung, failing a records-only pull request and dropping it from the merge queue. The race steps and make preflight already declare a 40-minute budget.
