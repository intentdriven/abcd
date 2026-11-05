---
schema_version: 1
id: "iss-2611050840151381"
slug: "a-host-review-can-stay-unlisted-in-the"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "coordinating build lanes, 2026-11-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/drive.go"
remedy: "Have each later step that finds a role still with the host list it when the ledger holds no entry for that run, lane and await, so a missed placeholder heals. A test kills a step between the hand-out and the listing and asserts the next step lists the role once."
---

A host review can stay unlisted in the launch ledger. ListHostRole (internal/core/implement/loop/drive.go:171) writes the placeholder only after DriveReview returns, at the end of the step that handed the role out (internal/surface/cli/build.go:1317). A DriveReview that returns the run's busy lock skips it, a step killed between the two never reaches it, and a later step reads the await with handed false, so nothing lists the role afterwards and the run's listing shows fewer agents than it had.
