---
schema_version: 1
id: "iss-301"
slug: "external-review-yml-computes-its-approver-tally-with-gh-api"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: ".github/workflows/external-review.yml"
resolution: "external-review.yml flattens reviews per page with --jq and reduces latest state per reviewer across all pages with sort and awk; per_page=100. TestExternalReviewTallySpansPages runs the committed script against a per-page fake gh over two-page fixtures."
impact: internal
resolved_by:
  commit: "75d498ebcbcd760516170d43f31861d5334f5c2d"
---

external-review.yml computes its approver tally with 'gh api .../reviews --paginate --jq' where --jq runs per API page (not across the full set; --slurp is incompatible with --jq), and no per_page is set so pages are 30 — so the 'latest state per reviewer wins' reduction the comment promises fails once reviews span a page: a stale page-1 APPROVED survives a later page-2 CHANGES_REQUESTED, and a straddling double-approval is counted twice. Fail-open on a required security gate. Not fixed autonomously: required-check workflow that cannot be validated without running Actions
## Evidence

- `.github/workflows/external-review.yml` — `gh api "…/pulls/$PR/reviews" --paginate --jq
  '… | group_by(.user.login) | map(max_by(.submitted_at)) | map(select(.state=="APPROVED"))'`.
  `gh --paginate` applies `--jq` per page (and `--slurp` is incompatible with `--jq`), and no
  `per_page` is set, so pagination is at 30. The "latest state per reviewer wins" comment
  above it does not hold once reviews span a page: a page-1 APPROVED survives a page-2
  CHANGES_REQUESTED, and a straddling double-approval counts twice.
- Required check per `.abcd/work/rulesets/main-protection.json`. Direction is fail-open on a
  security gate. Landed today (iss-281).

## Adversarial review

CONFIRMED (substantive, latent — needs >30 review entries on one external PR) by an
independent refuter. Proposed fix (NOT applied this round): split slurp and jq —
`gh api … --paginate --slurp | jq '[.[][] | …] | group_by(...) | …'`. Left open: modifying a
required-check workflow cannot be validated without a real Actions run.

## Grounds

- pursued: an approval superseded on a later page does not count, and a reviewer counts once however many pages their reviews span; shown wrong by an external PR going green on a withdrawn or double-counted approval
