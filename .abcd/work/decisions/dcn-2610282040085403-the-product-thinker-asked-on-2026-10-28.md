---
id: dcn-2610282040085403
slug: the-product-thinker-asked-on-2026-10-28
date: 2026-10-28
---
2026-10-28 — The product thinker asked on 2026-10-28 for the machine's capacity to be used, so load.preflights now takes a whole number from 1 to 4 (MaxPreflightSlots rises from the 2 of the trial in dcn-2610161814279204), and above one slot a preflight queued at its race gate takes no first-gate slot until its race run starts (iss-2610282016250805). The grounds: one race run at a time, kept by the race gate, and the load wait limit stay the bounds; each stage's go test -p is still the cores divided among the slots, never below a quarter of them; and the count is the person's own machine setting, never a repository's, one by default.
