---
id: dcn-2611061149567338
slug: a-claim-over-unsaved-work-keeps-renewing
date: 2026-11-06
---
2026-11-06 — A claim over unsaved work stays renewable under a stop. This narrows dcn-2611041550139264, by which a stop releases the run's claims: where a lane's save failed, or a changed configuration left the lane unsettled, the release waits until the work is saved (`internal/core/implement/loop/stop.go:612`, `:803`). Under a stop the claim renews only while a gated waiter, `abcd implement status` or a running drive runs (`renewruns.go`); a step and a drain's move refuse at the stop before they reach their renewal (`loop.go:994`, `drain.go:442`). With none of those three running it lapses with its lease, and a peer can then reserve over the unsaved work. Where a stop and a drive's own stop both stand on a run, nothing is settled, so the work stays unsaved and the reservation lapses with its lease (`internal/core/implement/loop/stop.go:294`).
