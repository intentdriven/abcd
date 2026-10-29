---
schema_version: 1
id: "iss-2610280910566850"
slug: "an-intent-build-re-widens-its"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "autonomous-run-2026-10-28"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement"
remedy: "Reserve per lane: at each step the run reserves the open lanes' own packages (from the spec step's packages: list), widening only when a lane opens that needs more, and a renewal keeps the narrowed set."
---

An intent build re-widens its reservation to the spec's whole footprint at every step, instead of reserving only the packages of the lanes it has open. A spec footprint that names whole packages (or the whole of commands/) then overlaps peers' narrow file reservations, so the build's renewal at land is refused until the peer releases by hand, and a hand-narrowed reservation is widened again at the next step. Seen 26-28 Oct on run-2610252354599453: a peer's commands/land.md and commands/guard.md reservations blocked lane 1's renewal.
