---
schema_version: 1
id: "iss-2611270400385786"
slug: "a-renewal-that-arrives-seconds-after-a"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "autonomous surge run, coordinating session, 27/11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim"
remedy: "Treat an overlap with the same holder's own live reservation alike for every path: warn, never refuse, as a fresh reservation over the json pages already does; and let --renew within a short grace after effective expiry renew the lapsed reservation in place when no other holder took any of its paths meanwhile, with a test for each"
---

A renewal that arrives seconds after a reservation lapses is refused by the holder's own other reservations, and only for some paths. On 27/11 five reservations of one holder lapsed between 03:57:58Z and 03:58:19Z, and their --renew ran moments later. Each was refused because a file of its footprint (commands/build.md, commands/drain.md, commands/land.md, internal/surface/cli/implement_load.go) is also in a sixth live reservation of the same holder. A fresh reservation by the same holder over other shared paths (docs/reference/cli/commands/drain.json, land.json) only warned. The holder had to drop the shared files from five footprints to get the reservations back, so those files stay fenced only while the sixth reservation lives.

The split follows the call, not the paths: a fresh reservation passes WarnOwnOverlap (internal/surface/cli/reserve.go:469), so an overlap with the holder's own reservation only warns, while --renew of a lapsed reservation reserves afresh without it and refuses.
