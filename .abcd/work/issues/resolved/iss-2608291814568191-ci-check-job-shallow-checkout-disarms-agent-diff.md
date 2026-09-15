---
schema_version: 1
id: "iss-2608291814568191"
slug: "ci-check-job-shallow-checkout-disarms-agent-diff"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: ".github/workflows/ci.yml"
resolution: "the check job checks out with fetch-depth: 0 so the base commit and a merge base are present for -agent-diff, and the unarmed fallback prints a ::warning:: naming the missing base"
impact: fix
---

ultra-v0.6.8 C2: the check job in .github/workflows/ci.yml checks out with the default shallow depth (no fetch-depth: 0, unlike the release-gate, record-lint and secret-scan jobs), so BASE_SHA is never a present object, git cat-file -e always fails, and the record-lint step always takes the unarmed fallback. The -agent-diff arm the comment says enforces agent_contract's unbumped-edit check is structurally unreachable, and the step reports green without saying it downgraded. Fix: fetch-depth: 0 on that checkout and a ::warning:: on the fallback so a silent downgrade is visible.
