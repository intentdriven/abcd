---
schema_version: 1
id: "iss-2608282038283692"
slug: "a-stale-remote-branch-carrying-a-secret-literal-fails-the-gi"
severity: "major"
category: "process"
source: "agent-finding"
found_during: "intent-implementation-run"
found_at: ".github/workflows"
resolution: "ci.yml's gitleaks step passes --log-opts: base..HEAD on a pull request, HEAD on push and merge_group, HEAD with a warning when a pull request's base is unusable. A stale remote branch is no longer walked. TestGitleaksScansOnlyTheHistoryThisRunWouldLand runs the step's script per event; checked by hand against the real gitleaks on a scratch repository."
impact: internal
resolved_by:
  commit: "4c13635f4e35a9c16578b573a2401fb26df2ec85"
---

A stale remote branch carrying a secret literal fails the gitleaks gate on every open pull request, and deleting the branch is the only remedy: the CI job checks out with fetch-depth 0, which fetches every remote ref, so 'gitleaks git' scans all of them rather than the pull request's own history. A literal committed on a branch whose pull request was closed and superseded therefore keeps failing unrelated pull requests, and the failure names neither the branch nor the file when the run log truncates the findings, so the natural diagnosis (a leak in this pull request) is wrong. Observed 2026-08-28: commit 9ffbbfbc on the superseded branch of a closed pull request blocked all seven open pull requests until the branch was deleted. Worth either scoping the scan to the pull request's own commits, or adding a cleanup gate that refuses to leave a leaking branch on the remote after its pull request closes.

## Grounds

- pursued: a secret on a branch that is not an ancestor of the checked-out commit no longer fails the scan, while one in the landing history still does; shown wrong by a CI gitleaks failure whose finding sits only on another ref
