---
schema_version: 1
id: "iss-2610231004140527"
slug: "the-live-reservation-cap-is-counted-per"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "hand-rolled coordination log, 2026-10-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/crowd.go"
remedy: "Waits on ruling (keep one ceiling per person, per iss-2610170717515674, or count per account and machine): either way, abcd peers warns when a holder is at or near any ceiling it can read and lists each live hold's session and copy, and the cap refusal names which of those holds are this machine's; if the ruling is per account and machine, count only the holds whose copy key belongs to this machine's accounts against its reservation.max_live"
---

The live-reservation cap is counted per GitHub login across every session, copy and account that pushes under it (internal/core/claim/crowd.go), while each machine sets its own ceiling in reservation.max_live. On 2026-10-23 six interactive sessions on one login reached 30 live holds and abcd-43's reserve was refused at 10:40; abcd-ae raised the ceiling to 40 by hand, the third raise since 10-22 (hand-rolled coordination log, 10:40 and 10:41). A drain run under another account that pushes with the same login is counted against the same holder under its own, possibly lower, ceiling, which no session here can read, so the interactive sessions can hold the drain at its cap with nothing on either side saying so. Severity minor: the drain waits at the cap rather than failing (iss-2610161015371794), and whether the drain shares this login and at what ceiling is not yet confirmed; if it is confirmed, the drain making no progress for as long as the sessions hold more than its ceiling makes this major. This revisits the ruling recorded in iss-2610170717515674 (one ceiling per person, raised in the machine's configuration); related: iss-2610160925028248 (one reservation per record spends the cap) and iss-2610161457209287 (holds do not name their copy).
