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
resolution: "The comment above the record-lint job in .github/workflows/ci.yml names the three ledger gates it runs (RD001-RD004, RS001-RS006, DA001-DA004), says the design-record drift gate cmd/record-lint is a step of the check job, and says the id stays because it is the required status check the main-protection ruleset names, so a rename lands only with the live ruleset edit. Resolved by wording: the check keeps its name on the forge, which a rename would change only together with a ruleset edit nobody has asked for. No gate would have caught it."
impact: internal
resolved_by:
  commit: "f3665abbd"
---

The CI job id record-lint in .github/workflows/ci.yml runs scripts/check-reviews-cases.sh and scripts/check-reviews.sh (the reviews-charter gate), while the real record-lint is a step of the check job, so a required status check is named for a gate it does not run. Renaming the job alone breaks the merge gate, because the main-protection ruleset (mirrored in .abcd/work/rulesets/main-protection.json) requires the context record-lint: the rename and the live ruleset edit must land together, which needs the forge ruleset changed by someone holding that permission. Carried over from iss-304 (d-12) when its other two halves were closed.

## Grounds

- pursued: a reader of ci.yml learns from the job's own comment what the record-lint check runs and why it is so named; a gate the job runs that the comment omits, or a claim that the id can be renamed alone, would show it wrong
