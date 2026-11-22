---
schema_version: 1
id: "iss-2611211905250368"
slug: "abcd-build-prints-nothing-in-its-text"
severity: "minor"
category: "ux"
source: "review-followup"
found_during: "the review of the fix for iss-2611101316317556"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/build.go"
remedy: "In the start's text render in internal/surface/cli/build.go (the closure that calls renderStartHead and renderBudget), print one line from the eligible row in res.Checks when the start judged the issue from origin and this checkout is behind it, or could not read origin and judged its own ledger, saying how many commits behind and from which commit, as renderDrainOrigin in internal/surface/cli/drain.go prints the drain's origin line; the --json output and every refusal already carry the clause."
---

abcd build prints nothing in its text output about the checkout being behind origin when a start succeeds
