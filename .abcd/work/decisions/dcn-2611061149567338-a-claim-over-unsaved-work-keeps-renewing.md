---
id: dcn-2611061149567338
slug: a-claim-over-unsaved-work-keeps-renewing
date: 2026-11-06
---
2026-11-06 — A claim over unsaved work keeps renewing under a stop, so that a failed save never lets a peer take work that has not been saved. This narrows dcn-2611041550139264, by which a stop releases the run's claims: after a failed save the release waits until the work is saved. It covers a failed save alone. Where a stop and a drive's own stop both stand on a run, nothing is settled, so the work stays unsaved and the reservation lapses with its lease (`internal/core/implement/loop/stop.go:294`).
