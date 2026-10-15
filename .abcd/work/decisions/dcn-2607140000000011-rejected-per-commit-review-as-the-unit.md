---
id: dcn-2607140000000011
slug: rejected-per-commit-review-as-the-unit
date: 2026-07-14
---
2026-07-14 (review): REJECTED per-commit review as the unit. No serious tool does it
(all per-PR); findings on WIP commits are false positives by construction; cost is
$75-500 per PR-equivalent at published frontier rates. Unit of review is the BRANCH
DIFF. Also rejected: spec-conformance as a BLOCKING gate — best agent scores 44.4%
on the easier adjacent task (SpecBench) and no precision/recall for
implementation-vs-spec is published anywhere. It ships advisory only, per
verifier-selects-gates-decide. Full evidence:
.abcd/development/research/notes/2026-07-14-cross-model-review.md
