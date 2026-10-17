---
schema_version: 1
id: "iss-2610161015393523"
slug: "a-drain-s-in-progress-lane-is-renewed"
severity: "major"
category: "bug"
source: "review-followup"
found_during: "multi-account drain investigation, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/reservation.go"
remedy: "Renew the in-progress lane's reservation on every drain move and check the reservation before the landing push (grounds: lease keep-alives independent of progress, as Chubby and etcd)"
related_intents: [itd-2610161016394967]
resolution: "Every drain move renews the lane in progress at half its lease, waiting or not, independent of the lane's steps, and the landing checks the reservation before its push as well as at arming, so a lapsed lease taken over by another pushes nothing, opens no pull request, and the lane is handed back"
impact: fix
resolved_by:
  commit: "3c250f615"
---

A drain's in-progress lane is renewed only when its own step or receipt runs, so if the host pauses the lane for more than the two-hour default lease another account takes the reservation over five minutes after expiry and works the same issue; the land fence runs only at arming, after the push and the pull request, so the duplicate becomes a public pull request.
