---
schema_version: 1
id: "iss-2611031719593367"
slug: "two-notes-from-1378-s-landing-review"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "coordinating the build lanes, 2026-11-03"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/loop.go"
remedy: "Make handedMove name one model for a tiered host-run reviewer, the one the review set resolved; and re-hold the requested model against the fallback runner before a launch falls back; a test covers each."
---

Two notes from #1378's landing review (spc-2610240951109420 step 3, 2026-11-03): handedMove in internal/core/implement/loop/loop.go tells the host two different things about which model to start for a host-run review-set reviewer under tiering; and holdRequestModel in internal/core/runner/config.go vets only the route's own runner, so a launch-time fallback to another runner is not re-held against the requested model (unconfirmed).
