---
id: dcn-2611130923121462
slug: the-spot-check-store-s-fetch
date: 2026-11-13
---
2026-11-13 — The spot-check store's fetch (spc-2610301925429250, built in step 1, called by no verb yet) runs under the caller's context with a one-minute bound of its own. Its push runs under the network runner's ten-minute bound: no runner in internal/gitutil pushes with the repository's hooks running under a caller's context, and internal/gitutil is outside step 1's footprint. Step 2, which wires the push, adds that runner before any verb calls it.
