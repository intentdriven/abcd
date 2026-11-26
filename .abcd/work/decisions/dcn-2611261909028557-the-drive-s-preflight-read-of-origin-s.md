---
id: dcn-2611261909028557
slug: the-drive-s-preflight-read-of-origin-s
date: 2026-11-26
---
2026-11-26 — The drive's preflight read of origin's default-branch tip (originDefaultTip in internal/surface/cli/drain_drive.go) keeps its own whole read of the ls-remote listing for now, rather than moving onto gitutil.RemoteHeadTip with the other readers: the drive-loop lane has first claim on that file until its step 4 lands. Past the output bound the read gives unverified, so the lane preflight runs its full rule; it moves onto the shared reader in the first change to that file after step 4.
