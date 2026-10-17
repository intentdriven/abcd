---
schema_version: 1
id: "iss-2610142134350426"
slug: "kept-aside-temp-left-on-kill"
severity: "nitpick"
category: "tech-debt"
source: "review-followup"
found_during: "PR-976-review-round-5"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/validate.go"
remedy: "Sweep stale .<route>-return-<hex>.tmp files from the review's directory when keepAside next runs there, or when the review's await is verified or re-told."
---

A process killed between keepAside writing the answer's temporary file and linking it to its name leaves a hidden .rp-return-<hex>.tmp in the review's directory, which nothing removes later.
