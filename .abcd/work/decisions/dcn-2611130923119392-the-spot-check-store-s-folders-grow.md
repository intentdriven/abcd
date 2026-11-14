---
id: dcn-2611130923119392
slug: the-spot-check-store-s-folders-grow
date: 2026-11-13
---
2026-11-13 — The spot-check store's folders grow without bound in step 1 of spc-2610301925429250. A write lists the local copy's whole tree, a listing bounded at 8 MiB, and a read takes at most 4096 records and otherwise refuses, naming the count. Once the store outgrows those bounds every write fails, which the drain's note and the merge gate's reason say and nothing waits on. Step 2, which reads the records by windows, settles how the folders are split or pruned so that a listing stays bounded.
