---
schema_version: 1
id: "iss-2610230410035051"
slug: "a-release-hold-records-a-merge-queue"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "iss-2610222014550692 round-1 review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/withdrawforge.go"
remedy: "Record both forms for a queued pull request and re-arm auto-merge first, then enqueue, so a pull request that is not yet ready rejoins the queue when it is."
---

A release hold records a merge-queue pull request's withdrawal as the queue form only, though merge-when-ready in a queue repository is auto-merge plus a queue entry; re-arming it is a direct enqueuePullRequest, which the forge refuses when the pull request is not ready (checks stale after the base moved), so it stays withdrawn and the person's wish to merge when ready is lost.
