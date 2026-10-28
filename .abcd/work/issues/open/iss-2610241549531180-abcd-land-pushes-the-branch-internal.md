---
schema_version: 1
id: "iss-2610241549531180"
slug: "abcd-land-pushes-the-branch-internal"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "hand-back triage of 2026-10-24, batch b06 reading iss-2609181223354098"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing/land.go"
remedy: "Have abcd land read the pull request's auto-merge and merge-queue state before its push step, and when the entry is armed or queued and the local head differs from the forge head, refuse with the withdraw, push, re-arm sequence instead of pushing; prove it with a forge-stub test in internal/core/landing where a queued entry and a new local commit leave the remote branch unmoved."
refines: [iss-2609012313465609]
---

abcd land pushes the branch (internal/core/landing/land.go:315) before it reads whether the branch's pull request is armed and in the merge queue (enqueue, land.go:1183), so re-running land after a new local commit pushes to a queued entry, the dropped-push hazard iss-2609181223354098 records, and the run then reports the arming as already in place.
