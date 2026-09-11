---
schema_version: 1
id: "iss-2609091258115663"
slug: "the-range-gates-self-skip-in-the-merge-queue"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "adversarial-review"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/ci.yml"
resolution: "The three range-scoped steps in ci.yml now derive BASE_SHA from a three-rung chain — the pull request's base, then merge_group.base_sha, then the push's before — so a merge-queue entry resolves the base the entry was formed against and runs the checks that were skipping themselves there. github.event.before stays last because it is the only rung that can be present and wrong, and each step keeps its own guard against an all-zeroes or unreachable base. A new semantic detector in internal/core/lint evaluates every BASE_SHA expression ci.yml declares, in simulated merge_group, pull_request and push contexts, and derives the guarded set from the file so a fourth range gate is covered by existing."
impact: internal
---

Three range-scoped gates derive their base from the pull-request event or the push event and fall back to nothing, so on a merge-group event they skip themselves and say so. The expression is the pull request base sha or the push before sha, and the workflow's own comment beside it acknowledges that a merge-group event carries neither; the log of a real queue run shows the consequence in the gates' own words, that there is no usable base ref for this event so the range checks are skipped. The gates that go quiet are the issue-resolution range check and the decisions-append check, and the run that skips them is the one that actually gates the merge. That inverts the protection: the queue entry is the only run that sees the final base after a competing pull request merged ahead of it, which is exactly the race a range check exists to catch, and it is the run where the check does not happen. A pull request whose base was clean when it was armed can therefore merge a resolution that a competitor made terminal while it waited, and every earlier run reported green on a base that no longer existed. The event carries the field the expression needs: merge_group.base_sha names the queue entry's base, so extending the fallback chain to it closes the hole without loosening anything. Detector: a merge-group run must resolve a base ref and run the range checks, and the checks must fail a queue entry whose competitor made its record terminal first, while a pull-request run keeps resolving the base it resolves today.

## Grounds

- pursued: the queue entry is the only run that sees the final base after a competing pull request merged ahead of it, so arming the range checks there is what closes the race; a merge-queue run that reds where the pull-request run passed for a reason other than a genuine collision would show it wrong.
