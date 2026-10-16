---
schema_version: 1
id: "iss-2610160315304959"
slug: "after-abcd-land-s-own-push-moves-a"
severity: "minor"
category: "ux"
source: "impl-review"
found_during: "review of the abcd land forge-lag fix, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing/land.go"
remedy: "Keep the head the first listing showed; during the post-push wait, stop at once when a listing shows a head that is neither that first head nor the pushed commit (someone else pushed), naming both, instead of waiting out the full 30 s"
---

After abcd land's own push moves a branch, the post-push wait re-reads the pull request until the forge shows the pushed commit, for up to 30 s of pauses. When someone else pushes to the branch straight after, the forge settles on a third head that will never equal the pushed one, and land waits the whole 30 s before it stops with its 'pull the forge's head' remedy. Nothing unsafe follows, since land never arms a head it did not gate, but the stop arrives 30 s late. Found by the Opus reviewer of the forge-lag fix (iss-2610151855227496), 2026-10-16.
