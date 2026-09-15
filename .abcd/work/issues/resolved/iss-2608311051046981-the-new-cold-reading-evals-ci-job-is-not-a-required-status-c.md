---
schema_version: 1
id: "iss-2608311051046981"
slug: "the-new-cold-reading-evals-ci-job-is-not-a-required-status-c"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "the cold-reading-evals context is added to the committed ruleset mirror .abcd/work/rulesets/main-protection.json (nine required checks now), the workflow comment records that the job is required and why requiring it is safe, and TestColdReadingEvalsIsARequiredStatusCheck holds the three parts together: the job exists, it stands down on no event, and the mirror requires its context. The LIVE ruleset edit is the maintainer's, after this merges."
impact: internal
---

The new cold-reading-evals CI job is not a required status check: .abcd/work/rulesets/main-protection.json lists attribution, check (macos-latest), check (ubuntu-latest), external-review, gitleaks, record-lint, smoke and zizmor, and the job spc-64 lands reports a conclusion without gating the merge. The whole point of the always-run lane is that a record-only pull request cannot reach main with warm content in included material, and an unrequired check does not stop one. Add cold-reading-evals to the ruleset and to its committed mirror.

## Grounds

- pursued: a record-only pull request carrying warm content into included material is stopped at the merge rather than reported on; it would be shown wrong by a merge that lands with the cold-reading-evals check red or absent.
