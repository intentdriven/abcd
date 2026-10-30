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
resolution: "keepAside now sweeps from the review's directory, before it writes, every regular file named .<route>-return-<16 hex>.tmp last written an hour ago or more, so a temporary file a killed process left is removed the next time an answer is kept aside there; a younger one, which may be a concurrent writer's, stands."
impact: internal
resolved_by:
  commit: "1b43847829cdce3e903dd036e533b73f319856de"
---

A process killed between keepAside writing the answer's temporary file and linking it to its name leaves a hidden .rp-return-<hex>.tmp in the review's directory, which nothing removes later.

## Grounds

- pursued: a stale .<route>-return-<hex>.tmp in a review's directory disappears on the next keepAside there while a fresh one, a look-alike name and a directory stand; TestKeepingAsideSweepsATemporaryFileAKilledProcessLeft would show it wrong by failing, as would any report of a live writer's temporary file vanishing before its link (which keepAside also degrades through createAside).
