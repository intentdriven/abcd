---
schema_version: 1
id: "iss-2610231004140527"
slug: "the-live-reservation-cap-is-counted-per"
severity: "major"
category: "ux"
source: "agent-finding"
found_during: "hand-rolled coordination log, 2026-10-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/crowd.go"
remedy: "First, with no ruling needed: abcd peers lists each live hold of a login with its session and copy (as iss-2610161457209287 asks), and warns when the holder is at or near this machine's reservation.max_live or any other ceiling it can read, saying that another account pushing as the same login counts against the same total and its ceiling is not visible here; the cap refusal names which holds are this machine's. Test first: a fixture origin with holds from two copies under one login shows both copies and the warning at the ceiling. Then, waiting on the ruling (one ceiling per person, per iss-2610170717515674, or a count per account and machine): if per account and machine, count only the holds whose copy key belongs to this machine's accounts against its reservation.max_live"
resolution: "abcd peers now counts the reader's live reservations by copy (this copy, each sibling worktree here by name, elsewhere) against this machine's reservation.max_live, and from four fifths of it warns that every session and account on the same login counts toward one total whose other ceilings are not readable here; the cap refusal groups the held records by copy. Whether the count itself changes waits on a ruling, captured as iss-2610231127555243."
impact: additive
---

The live-reservation cap is counted per GitHub login across every session, copy and account that pushes under it (internal/core/claim/crowd.go), while each machine sets its own ceiling in reservation.max_live. On 2026-10-23 six interactive sessions on one login reached 30 live holds and abcd-43's reserve was refused at 10:40; abcd-ae raised the ceiling to 40 by hand, the third raise since 10-22 (hand-rolled coordination log, 10:40 and 10:41). A drain run under another account that pushes with the same login is counted against the same holder under its own, possibly lower, ceiling, which no session here can read, so the interactive sessions can hold the drain at its cap with nothing on either side saying so. Graded minor when filed, because the drain waits at the cap rather than failing (iss-2610161015371794), and whether the drain shares this login and at what ceiling is not yet confirmed; it was confirmed the same day, and the grade raised (below). This revisits the ruling recorded in iss-2610170717515674 (one ceiling per person, raised in the machine's configuration); related: iss-2610160925028248 (one reservation per record spends the cap) and iss-2610161457209287 (holds do not name their copy).

## Severity raised to major, 2026-10-23

Both unknowns were settled the same day. The drain pushes as the same login: its reservation of iss-2610120138028442 is authored and committed under it. Its ceiling cannot be read from this account, because the drain account's configuration is readable only by its owner. The drain had opened no pull request since 10-22, with five of its preflight waiters queued. So the interactive sessions of one login can hold the drain at a ceiling none of them can see, and neither side is told. That breaks the parallel-accounts arrangement abcd is being made to run, so the coordinator graded it major: it is fixed before the next cut, not deferred. The fix splits in two. The warning half needs no ruling and goes first: peers lists each live hold's session and copy, and warns when a holder is at its own ceiling or any it can read. The per-account count waits on the ruling the remedy names, which is going to the product thinker.

## Grounds

- pursued: a person at or near the shared total now sees it, and which copy holds what, before a drain on the same login stalls unseen; shown wrong if a session reaches the cap with no warning in peers, or a sibling's hold reads as elsewhere
