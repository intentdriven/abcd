---
id: dcn-2610170118506273
slug: a-drain-lane-s-sixty-minute-stuck-bound
date: 2026-10-17
---
2026-10-17 — A drain lane's sixty-minute stuck bound counts only the time across which its refusals kept coming (facilitator-on-delegation, drain-many-accounts; refines dcn-2610170005357588): the minutes run from the first refusal at the stage to the last one logged, never to the move that reads them, so a lane that refused once and then sat idle is not handed back by a move that came only to renew it; a refusal more than fifteen minutes after the lane's last, the drain's look-again time, starts the time afresh at that stage, as a quiet that long is not refusing; the count of identical refusals in a row is unchanged (grounds: the dead-letter bound counts receives, not wall time, so time without a receive is not evidence the lane cannot finish; the drain's fifteen-minute look-again as the cadence a host stepping as told keeps; review of spc-2610161044461448 step 5)
