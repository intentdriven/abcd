---
id: dcn-2610181003464379
slug: while-the-person-is-away-a-pull-request
date: 2026-10-18
---
2026-10-18 — While the person is away, a pull request is behind when the forge says so (facilitator-on-delegation, itd-2610180912189125 review round 2, blocker 2; refines dcn-2610180936571036): the stop check reads "behind" from the merge state (`mergeStateStatus` BEHIND) in its one pull-request listing, not from the last-fetched default branch, which nothing refreshes overnight, so a pull request that fell behind after the last fetch still reads as owed. Only a branch with no pull request is compared on disk, and the reason names when that ref was last fetched (grounds: plan review round 2, blocker 2; the pinned pull-request fields in internal/ghcontract)
