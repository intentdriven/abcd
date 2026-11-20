---
schema_version: 1
id: "iss-2611201352116252"
slug: "the-drive-s-queue-watch-names-a-dropped"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "diagnosing a parked drive lane"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/queue.go"
remedy: "Have failedGroupChecks ask gh for only the fields it reads, with --jq projecting each workflow run to head_branch, head_sha, name, conclusion and created_at (and a smaller page if that still does not fit), so the listing stays far under the cap; add a test that the projected listing of a hundred runs parses into groupRun"
---

The drive's queue watch names a dropped lane's failed merge-group check by listing the repository's latest 100 merge-queue workflow runs through gh api, and that listing now exceeds the forge client's 1 MiB output cap, so every dropped lane reads 'the merge group's checks could not be read (... wrote more than 1048576 bytes)' and the failed check is never named; pull request #1622 has carried that clause since 18 Nov. The loop reads only five fields of each run.
