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
remedy: "Give the CI workflow's merge_group runs a concurrency group keyed on the queue entry's pull request (github.event.merge_group head ref, minus its sha) with cancel-in-progress: true, so a rebuilt group cancels its superseded run; check the zizmor audit accepts it."
---

When the merge queue rebuilds its groups (an entry dropped, re-queued or re-armed), the CI runs of the superseded groups keep running and holding runners, though their gh-readonly-queue refs are deleted and nothing reads their results. On 2026-10-11 nothing merged from 03:36 to past 07:00 BST. Seven stale merge-group ci runs (PRs 903, 906, 908, 911 and 915 at older shas) held the runners while every live group's run sat queued, until a session cancelled them by hand.
