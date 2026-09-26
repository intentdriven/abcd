---
schema_version: 1
id: "iss-2609251358062952"
slug: "the-ci-job-id-record-lint-in-github-workflows-ci-yml-runs"
severity: "nitpick"
category: "inconsistency"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/ci.yml"
---

The CI job id record-lint in .github/workflows/ci.yml runs scripts/check-reviews-cases.sh and scripts/check-reviews.sh (the reviews-charter gate), while the real record-lint is a step of the check job, so a required status check is named for a gate it does not run. Renaming the job alone breaks the merge gate, because the main-protection ruleset (mirrored in .abcd/work/rulesets/main-protection.json) requires the context record-lint: the rename and the live ruleset edit must land together, which needs the forge ruleset changed by someone holding that permission. Carried over from iss-304 (d-12) when its other two halves were closed.
