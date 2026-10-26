---
id: dcn-2610250109070342
slug: the-records-branch-docs
date: 2026-10-25
---
2026-10-25 — The records branch docs/review-set-and-planning-autonomy is parked, unpushed, from 2026-10-25 01:15 UTC. Four preflights failed on tests the branch does not touch: two timed out at the load gate, and two failed in main's suite. Those failures were the deadline and forge-budget tests under the live reservation iss-2610241911001655, three packages past their 20-minute timeout, and the stop-hook tests in the daylight-saving fold (iss-2610250107576049). The release cut's merge hold starts at 02:30 UTC. The branch is rebased and gets one gated preflight once that fix is on main and the hold has lifted.
