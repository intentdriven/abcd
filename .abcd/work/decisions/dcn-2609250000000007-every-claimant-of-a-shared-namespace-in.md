---
id: dcn-2609250000000007
slug: every-claimant-of-a-shared-namespace-in
date: 2026-09-25
---
2026-09-25 — Every claimant of a shared namespace in `.abcd/config.json` lists the union of the keys any writer puts there, not only the keys it reads itself (lane implementer, autonomous run A, on the tier1 review's finding 3; this adds to the entry above on the layered resolver and does not amend it). `Stack.Claim` holds no registry across callers: each claim refuses every key it does not list, so two claimants listing different subsets each refuse the other's file. For the `oracle` namespace the union today is `backend` (written by `ahoy install`), `review` (itd-6's reviewer route) and `rp` (itd-6's `oracle.rp.mcp_config_path`, which ahoy's setup discovery writes), so itd-6's claimant calls `Claim("oracle", "backend", "review", "rp")`, and a key a later writer adds under `oracle` is added to every claimant in the same change. Separately, the ceiling of 1 recorded above makes a routing row's `fan_out` inert today: every value either equals the ceiling or is clamped to it, and the field only bites once an agent contract declares a ceiling above 1.
