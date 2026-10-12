---
schema_version: 1
id: "iss-2610110613372021"
slug: "stale-merge-group-runs-hold-runners"
severity: "minor"
category: "tech-debt"
source: "agent-observation"
found_during: "abcd-60 drain run 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/ci.yml"
remedy: "ci.yml keys merge_group runs on github.run_id with cancel-in-progress off, and each group's ref (gh-readonly-queue/main/pr-N-<sha>) is unique, so neither the run id nor github.ref can cancel a superseded run. Key merge_group runs on the pull request number parsed from the queue ref (pr-N), with cancel-in-progress on for merge_group, so a rebuilt group cancels the older run for the same PR; do the same for the attribution and external-review workflows; check the zizmor audit accepts it."
resolution: "queue-sweep.yml (#933) cancels, then force-cancels, the merge-group runs of ci, attribution and external-review whose gh-readonly-queue ref is gone, and never a live entry's; its first four live runs on 2026-10-11 left every live run alone"
impact: fix
resolved_by:
  commit: "0e47fb78337f"
---

When the merge queue rebuilds its groups (an entry dropped, re-queued or re-armed), the CI runs of the superseded groups keep running and holding runners, though their gh-readonly-queue refs are deleted and nothing reads their results. On 2026-10-11 nothing merged from 03:36 to past 07:00 BST. Seven stale merge-group ci runs (PRs 903, 906, 908, 911 and 915 at older shas) held the runners while every live group's run sat queued, until a session cancelled them by hand.
