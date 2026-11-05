---
schema_version: 1
id: "iss-2611050840164467"
slug: "an-auto-plan-review-on-a-role-routed-to"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "coordinating build lanes, 2026-11-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/drive.go"
remedy: "Have ListHostRole read what the step did rather than the configured route: a handed review on an auto-plan run is the host's whatever the route says. A test routes the reviewer to a runner, steps an auto-plan run and asserts one host placeholder."
---

An auto-plan review on a role routed to a runner goes unlisted in the launch ledger. Drive keeps an auto-plan run's reviews on the host by dropping its runner configuration (internal/core/implement/loop/drive.go:109), but ListHostRole (drive.go:172) still asks the configured route, finds a runner rather than the host and writes no placeholder, so the host agent that runs the review has no line.
