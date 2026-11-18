---
schema_version: 1
id: "iss-2611182334466021"
slug: "abcd-build-next-picked-a-planned-intent"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "autonomous run of 15 Nov 2026, build next refused at reservation"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/next.go"
remedy: "Exclude a candidate whose start-time footprint overlaps another holder's live reservation, by the start's own footprint and overlap rule, and name the picked intent when its start is refused anyway."
resolution: "build next excludes a candidate at the check overlap when the footprint its start reserves with (runFootprint, no step built) shares a path with another copy's live reservation as last fetched, judged by claim.Snapshot.Contention; a start refused anyway names the intent picked, and the board's next-up head follows"
impact: fix
---

abcd build next picked a planned intent whose start had to be refused at the reservation, because another copy held a path in that intent's footprint; the refusal named the path and the holder but not the intent picked, and the pick never moved on to the next ready intent, although at least one other candidate's remaining steps avoided the held package. The candidate set excludes an intent for a run in progress, a failed readiness check (whose peers row covers only the intent's own key) and an unreadable footprint, but never asks whether the footprint the start would reserve overlaps another key's live reservation as last fetched, so an autonomous run stalls on a pick it cannot start.

## Grounds

- pursued: we expect build next to pass over an intent another copy's hold blocks and start the next ready one; shown wrong if a pick is refused at the reservation over a hold the last fetch already showed
