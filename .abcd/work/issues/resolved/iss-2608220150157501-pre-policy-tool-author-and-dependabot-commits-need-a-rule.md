---
schema_version: 1
id: "iss-2608220150157501"
slug: "pre-policy-tool-author-and-dependabot-commits-need-a-rule"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "abcdev-site-plan investigation 2026-08-21"
found_at: "git history"
resolution: "handled at tip: site contributors derive a separate bots-and-tools row from [bot] names and machine mailboxes (internal/core/site/contributors.go, 766da566)"
impact: internal
resolved_by:
  commit: "766da56642e968b8e12b3ada4373703a2bd3f1ab"
---

One early commit carries the tool name in the git author field (pre-dating the Assisted-by policy) and two commits are dependabot; any contributors rendering from git history needs an explicit rule labelling bot and tool author rows separately from the humans who are authors of record, or the pre-policy commit silently appears as a human contributor

## Grounds

- pursued: the contributors rendering separates tool and bot authors from the humans of record; shown wrong if a pre-policy tool-authored commit renders as a human contributor
