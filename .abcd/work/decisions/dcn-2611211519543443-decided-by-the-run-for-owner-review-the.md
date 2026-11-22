---
id: dcn-2611211519543443
slug: decided-by-the-run-for-owner-review-the
date: 2026-11-21
---
2026-11-21 — Decided by the run for owner review: the records commit of a drive landing holds the issue store to the landing's own output. An added, deleted or moved issue record is admitted only when it is one of the lane's own resolutions, or the open-to-resolved move of an issue the closing intent names in related_issues (the documented resolve-it remedy that TestFollowingTheShipLinkRemedyLands covers). A modified issue record is admitted only when its content equals the link rewrite of the landing's own moves. The check lives in landingDiff, so judgeLandingDiff stays the one judge for the pre-stage, staged and found-commit paths. Rejected: dropping the resolve-it remedy and keeping only move-the-link, because users are already told to use it and the narrower rule closes the window just as well. To reverse: remove the related_issues allowance and the remedy text with it.
