---
schema_version: 1
id: "iss-2608210738363966"
slug: "attribution-gate-has-no-local-mirror"
severity: "minor"
category: "future-work-seed"
source: "review-followup"
found_during: "itd-130 session; #391/#402 attribution repair"
resolution: "make preflight runs check-attribution as a prerequisite: the attribution cases, then the identity and trailer check over origin/main..HEAD with the outbound checker built once, so a commit CI's attribution gate would refuse earns no push receipt. The body half stays in CI, where the body exists. The commit-msg hook (session URL and tool footer) and the pre-commit identity pin already covered parts of this locally."
impact: internal
resolved_by:
  commit: "edbcd12b0"
---

The attribution gate (scripts/check-attribution.sh) is CI-only: no local hook mirrors it, so a commit whose git AUTHOR/COMMITTER is an AI identity (Claude <noreply@anthropic.com>) is only caught in CI, where it presents as an unmergeable-looking PR (a failing required check), not a merge conflict. The pre-commit hook runs only the name-guard; make preflight / the pre-push hook run record-lint + docs-lint + reviews-charter but NOT check-attribution. This cost a full diagnosis on #391 (AI-authored cloud-routine commits) and a reauthor+new-branch repair (#402), because force-push to fix in place is refused by policy. Fix: mirror check-attribution into the pre-push gate (as make preflight already mirrors record-lint), so an AI author identity or a forbidden Generated-by/Co-Authored-By footer fails locally before the round-trip. See memory cloud-routine-git-identity; relates itd-91.

## Grounds

- pursued: an AI author identity, a Co-authored-by trailer or a missing Assisted-by trailer now fails locally before the push; shown wrong if such a commit passes make preflight on a branch
