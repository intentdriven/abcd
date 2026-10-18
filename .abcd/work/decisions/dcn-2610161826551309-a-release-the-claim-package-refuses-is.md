---
id: dcn-2610161826551309
slug: a-release-the-claim-package-refuses-is
date: 2026-10-16
---
2026-10-16 — A release the claim package refuses is never logged as released, and a hand-back never refuses a release (facilitator-on-delegation, drain-many-accounts review round): a tombstone that would pass the 4096-byte message cap, or the outbound policy, with its hand-back on it is pushed plain, and the result says the hand-back was not published and why (HandBackUnpublished), because a refused release leaves the issue held off every account until its lease runs out, which is worse than a hand-back a person can re-publish; a release the claim package still refuses for a reason of the tip's own is logged at reservation_release_failed, which the drain routing the hand-back tries again, never at reservation_released (grounds: review finding R1 on spc-2610031753359849 steps 1-3; spc-2610161044461448 step 3; the review of steps 1 to 18).
