---
schema_version: 1
id: "iss-2611201418535169"
slug: "abcd-build-reserves-an-intent-with-the"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "autonomous run of 15 Nov 2026, starting itd-2611041815429986"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/reservation.go"
remedy: "Reserve a start or resume with the packages of the step it opens next (with the Footprint line as the fallback where the steps cannot say), and widen the reservation as each later step's lane opens, refusing that step (exit 3, the run kept) when a peer holds one of its packages; a test starts step 1 of a two-step spec while a peer holds only step 2's package and finds step 1 reserved and started."
---

abcd build reserves an intent with the packages of every spec step not yet landed (specPackages in internal/core/implement/loop/reservation.go), so a peer's hold on a package that only a later step touches refuses the start of step 1, whose own packages nobody holds. On 19-20 Nov itd-2611041815429986 could not start for over a day: its step 1 names internal/core/needs and internal/core/runner, but step 7 names internal/surface/cli, which another account's drain held. The build opens one step's lane at a time, so the steps after the one being built claim packages the run will not touch until later.
