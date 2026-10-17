---
schema_version: 1
id: "iss-2610142151400382"
slug: "remove-placed-claims-a-peers-removal"
severity: "nitpick"
category: "bug"
source: "review-followup"
found_during: "PR-976-final-review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/validate.go"
remedy: "Return removed=false when root.Remove reports ErrNotExist, so takeBack words its refusal from what this call removed."
---

removePlaced reports removed=true when root.Remove returns ErrNotExist, a peer having removed the file between the digest check and the Remove, so takeBack's refused-drop message says the return was taken back though this call removed nothing.
