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
---

Three range-scoped gates derive their base from the pull-request event or the push event and fall back to nothing, so on a merge-group event they skip themselves and say so. The expression is the pull request base sha or the push before sha, and the workflow's own comment beside it acknowledges that a merge-group event carries neither; the log of a real queue run shows the consequence in the gates' own words, that there is no usable base ref for this event so the range checks are skipped. The gates that go quiet are the issue-resolution range check and the decisions-append check, and the run that skips them is the one that actually gates the merge. That inverts the protection: the queue entry is the only run that sees the final base after a competing pull request merged ahead of it, which is exactly the race a range check exists to catch, and it is the run where the check does not happen. A pull request whose base was clean when it was armed can therefore merge a resolution that a competitor made terminal while it waited, and every earlier run reported green on a base that no longer existed. The event carries the field the expression needs: merge_group.base_sha names the queue entry's base, so extending the fallback chain to it closes the hole without loosening anything. Detector: a merge-group run must resolve a base ref and run the range checks, and the checks must fail a queue entry whose competitor made its record terminal first, while a pull-request run keeps resolving the base it resolves today.
