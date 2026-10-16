---
id: dcn-2610161040350256
slug: a-drain-plans-from-origin-s-default
date: 2026-10-16
---
2026-10-16 — A drain plans from origin's default branch (facilitator-on-delegation, drain-many-accounts): each drain move fetches origin inside the drain's lock before it reads anything, and takes one snapshot of the ledger from origin's default branch, which the plan, the lane's start check, the judgement's remedy digest and the reservation footprint all read; a fetch that fails, a default branch the fetch could not read, or a missing origin/HEAD means the move cannot plan and enters the drain's outage wait, never planning from what the checkout last saw; the dry run plans from the last fetched view and says how old it is, so it still writes nothing; a checkout behind origin is warned about, not refused, because the lane is cut from origin anyway; and any reservation of a record finished on origin's default branch backs off, with or without a tip (grounds: a queue consumer reads the row it locks, as SELECT ... FOR UPDATE SKIP LOCKED; iss-2610160957404827, iss-2610161015448977).
