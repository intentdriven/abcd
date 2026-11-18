---
schema_version: 1
id: "iss-2611181755106054"
slug: "abcd-build-of-an-intent-reserves-the"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "autonomous run of 15 Nov 2026, build of itd-2610240922461281"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/reservation.go"
remedy: "Reserve the union of the packages lines of the steps not yet landed, falling back to the Footprint packages line when a remaining step names none or every step has landed."
resolution: "A build reserves the union of the packages lines of the spec steps still to build, the Footprint line when the steps cannot say, and the steps the run built once every step has landed."
impact: fix
resolved_by:
  commit: "7e2f2051b"
---

abcd build of an intent reserves the whole packages line of its spec's Footprint section even when some of the spec's steps already carry landed lines and only later steps are left to build. In the autonomous run of 15 Nov 2026, abcd build itd-2610240922461281 (spec spc-2610240951109420) was refused at the reservation with exit 3: steps 1 to 4 had landed and step 5 touches only internal/core/implement/loop and commands, but the build reserved internal/surface/cli too, where another issue held a live reservation. Every READY intent whose spec names a directory some peer holds is blocked the same way, though the steps still to build never touch it.
