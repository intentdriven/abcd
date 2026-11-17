---
schema_version: 1
id: "iss-2611101316537446"
slug: "a-lane-s-reservation-does-not-follow-the"
severity: "major"
category: "bug"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/widen.go"
remedy: "Grow the reservation with the lane: at each step and before the push, reserve the paths the lane's commits change beyond the held footprint, cover the run's own record paths (decision records, the spec's landed line, the issue record) by default, and let an intent lane widen the same way with its spec's footprint updated for it; a test commits a file outside the found_at and finds it reserved before the push."
resolution: "The loop widens a run's hold, an intent's as an issue's, to every path its lanes' branches change beyond the footprint the reservation holds, at the receipt, before each step at the validators and at the fence before the push and the arming, in one reserve keeping every path already held; a decision record and the records the landing changes are reserved by their own names, never the decisions directory. Another copy's live hold on one stops the push as a wait naming the path and holder. implement widen takes an intent's run, and a resume over this copy's hold keeps the paths it names."
impact: fix
resolved_by:
  commit: "69d3a3686e299e89dc7e329c3a279d5cbf973625"
---

A lane's reservation does not follow the files the lane changes. An issue's build reserves only its found_at path; implement widen works only at the implement or validate stage and refuses an intent's run, whose footprint grows only by editing its spec; and the decision records and the spec's landed line that every lane writes are not covered. So lanes change files they never reserved, and a coordinator widens the hold by hand. On 9 Nov at 07:20 and 11:15 build reserved one file of five and one file of eight, and each hold was widened with a hand reserve naming every path; on 8 Nov at 10:40 a lane had changed four loop files its reservation never covered, and widening that intent lane meant a spec edit inside the lane; on 9 Nov at 13:50 a lane could not record its judgement calls with abcd decide because the new decision file sat outside its reservation; on 10 Nov at 08:54 build reserved internal/core/claim/view.go alone though the remedy named five packages.

## Grounds

- pursued: an intent lane that commits a file beyond its spec's footprint finds it reserved before its push; shown wrong if a lane lands a path its reservation does not name
