---
schema_version: 1
id: "iss-2610222359046244"
slug: "abcd-build-and-abcd-reserve-refuses-a"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "lane D merge-gate build start"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim"
remedy: "List every live reservation that overlaps the footprint in one refusal, each with its key, holder, expiry and the overlapping paths (a directory footprint named as such), so the person can route round all of them at once."
---

abcd build (and abcd reserve) refuses a reservation by naming only the first footprint path that overlaps a live reservation, so a build whose footprint collides with several reservations is refused once per collision: on 2026-10-22/23 the merge gate's build (itd-2610150602547271) was refused in turn on internal/core/implement/loop/land.go (iss-2610120138028442), internal/surface/cli/cli.go (iss-2610171110154902; iss-2610171200535090 held it too, unnamed) and internal/core/landing/config.go (iss-2610222014550692, a directory footprint), each discovered only after the previous one was routed round.
