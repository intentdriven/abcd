---
schema_version: 1
id: "iss-2610142134339172"
slug: "rp-review-sent-twice-after-busy-lock"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "PR-976-review-round-5"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/validate.go"
remedy: "Before DriveReview sends a review to the route, skip the send when the review's await already records an answer (Await.Answered) or its return path already holds one, and name the standing return instead."
---

After a busy-lock refusal leaves the reviewer route's placed return standing, a later implement step can re-tell the open review and DriveReview sends it to rp a second time, a review that can take up to 30 minutes, only to keep the second answer aside because the first return still stands.
