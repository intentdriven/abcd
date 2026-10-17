---
id: dcn-2610161135122294
slug: reserving-a-record-being-reopened-on-a
date: 2026-10-16
---
2026-10-16 — Reserving a record being reopened on a branch backs off until the reopen is merged (facilitator-on-delegation, drain-many-accounts): the finished-record back-off reads only origin's default branch, so while that branch holds an issue resolved or wontfix, or an intent shipped or superseded, a branch that moves it back to open cannot reserve it, and no flag gets past the back-off; the reopen is merged first, and the record is reserved once origin's default branch holds it open again; accepted because the default branch is the one view every account shares, and an exception for one branch would let a checkout reserve work that origin still calls done (spc-2610161044461448 step 1; iss-2610160957404827).
