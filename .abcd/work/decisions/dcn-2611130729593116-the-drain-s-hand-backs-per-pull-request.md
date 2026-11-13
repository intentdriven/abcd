---
id: dcn-2611130729593116
slug: the-drain-s-hand-backs-per-pull-request
date: 2026-11-13
---
2026-11-13 — The drain's hand-backs per pull request are derived from the lanes and hand-backs its state file already keeps, not stored as a new pull-request field on each hand-back as spc-2610301925429250 approach 2 writes it: the drain state is decoded strictly, so a new field would make a pinned older build refuse the file, and the derivation needs no schema version change. A hand-back belongs to the lane on its issue that opened latest before it was routed.
