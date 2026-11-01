---
schema_version: 1
id: "iss-2611011010445156"
slug: "a-hand-reservation-was-renewed-with"
severity: "minor"
category: "lapse"
source: "agent-finding"
found_during: "landing the lane widening fix after its preflight blockers"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/reserve.go"
lapsed_at: "2026-10-31T23:09:38Z"
remedy: "Once iss-2610311539091723 refuses a hand reservation of an issue or intent, this route is closed: the build reserves the found_at and the lane widens by named paths; confirm then that no caller builds an issue footprint from a diff, and resolve this with that check"
blocked_by: [iss-2610311539091723]
---

A hand reservation was renewed with eleven paths its work never changed: widening the hold on iss-2610311224444576 to one test file, the footprint was rebuilt by hand from a two-dot diff against a default branch that had moved, so the renewal also named a decision record and ten issue records other merged changes had written, and held them from 2026-10-31T23:09:38Z until the release at 03:06:16Z. reserve replaces the footprint wholesale with whatever paths it is given and asks nothing about them. No copy that left a trace met the extra paths in that window, by the search iss-2611011010338815 records; a refused attempt that left none cannot be ruled out.
