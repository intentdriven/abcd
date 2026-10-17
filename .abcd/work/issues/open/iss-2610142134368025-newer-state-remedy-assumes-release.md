---
schema_version: 1
id: "iss-2610142134368025"
slug: "newer-state-remedy-assumes-release"
severity: "nitpick"
category: "ux"
source: "review-followup"
found_during: "PR-976-review-round-5"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/state.go"
remedy: "Word the newer-state remedy so updating is the second option: run the abcd build that wrote the file, which for a development build is that build, and update only when it was a release."
---

The remedy for a run state of a newer schema version suggests abcd update, which assumes the newer abcd that wrote the file was released; a file a development build wrote needs that build, and updating to the latest release still refuses it.
