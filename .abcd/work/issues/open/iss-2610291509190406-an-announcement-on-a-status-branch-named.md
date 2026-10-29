---
schema_version: 1
id: "iss-2610291509190406"
slug: "an-announcement-on-a-status-branch-named"
severity: "minor"
category: "ux"
source: "impl-review"
found_during: "holder-id review (iss-2610290809526033)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/status.go"
remedy: "For a status branch named by a holder id, keep asking the forge who pushed each new commit and judge constancy rather than identity: keep, in the local tier, the pusher the forge names for the branch's first commit read here, and mark any later commit whose pusher differs as doubtful; no login enters any ref or trailer."
---

An announcement on a status branch named by a holder id always shows its sender as claimed: abcd no longer asks the forge who pushed it, because a holder id names no account the forge's record could match. So a later commit on that branch from a different pushing account looks the same as the branch owner's own, where on a branch named by a username the forge's record could mark it doubtful. Nothing acts on the sender check beyond what status, peers and the prompt hook display.
