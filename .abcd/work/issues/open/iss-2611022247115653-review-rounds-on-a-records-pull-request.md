---
schema_version: 1
id: "iss-2611022247115653"
slug: "review-rounds-on-a-records-pull-request"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "coordinating the build lanes, 2026-11-02"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing"
remedy: "Scope a fix round's review to the delta since the last reviewed head plus whether the prior findings are closed, and keep one full review for the landing commit; say in the review brief which kind of round it is."
---

Review rounds on a records pull request do not converge: #1347 reached its eleventh review round on 2026-11-02, because each round re-reviews the whole diff against main and raises new findings in records the previous fix never touched, so a fix round cannot know when it is done.
