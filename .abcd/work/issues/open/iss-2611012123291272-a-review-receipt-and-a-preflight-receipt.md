---
schema_version: 1
id: "iss-2611012123291272"
slug: "a-review-receipt-and-a-preflight-receipt"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "landing a records branch through repeated amendments, 2026-11-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing/facilitator.go"
remedy: "Let a review receipt carry forward to a later head when every commit after the reviewed head touches records alone: run a delta review over the reviewed head's diff to the new head, and record the new receipt as resting on the earlier one, so a one-sentence records fix costs one delta review rather than a full pair."
---

A review receipt and a preflight receipt each name one head, so every amendment to a records branch, however small, sends both reviewers and the preflight round again in full: on 2026-11-01 one records branch took six full review rounds in an evening, most for a one-sentence fix, each round re-reading every record already approved (internal/core/landing/facilitator.go reads the receipt by head).
