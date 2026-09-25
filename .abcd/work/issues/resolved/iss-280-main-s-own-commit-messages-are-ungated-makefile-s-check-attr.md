---
schema_version: 1
id: "iss-280"
slug: "main-s-own-commit-messages-are-ungated-makefile-s-check-attr"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "manual-capture"
found_at: "Makefile"
resolution: "attribution.yml runs the commit half on merge_group (base_sha..head_sha) and on push to main (before..sha), refusing an unusable base; the PR-form steps exempt non-PR events loudly. TestAttributionGatesTheCommitsThatLandOnMain pins it. The local make target keeps origin/main..HEAD, which is correct for a branch; main's own commits are gated by the push run."
impact: internal
resolved_by:
  commit: "7f3b41f1e229f847f93acf64b2fc7362aa0f63f5"
---

main's own commit messages are ungated: Makefile's check-attribution runs origin/main..HEAD, whose base is main itself, so a commit that lands on main (squash, web merge) is never checked by any local or CI run afterwards

## Grounds

- pursued: a commit that lands on main with a machine author or a missing trailer turns the queue or the push run red; shown wrong by such a commit on main with a green attribution push run
