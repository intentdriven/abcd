---
id: dcn-2610181003484631
slug: abcd-s-count-of-blocks-resets-only-when
date: 2026-10-18
---
2026-10-18 — abcd's count of blocks resets only when the host is not already continuing (facilitator-on-delegation, itd-2610180912189125 review round 2, choice (b); refines dcn-2610180936577720): the count is set to one on a block with `stop_hook_active` false and adds one with it true; a change in the owed set no longer resets it, since the host's count does not, and a reset abcd made on one could leave a stop the host forced unrecorded. A stop recorded at the cap is labelled "possibly forced by the cap", because the host's count also resets on a tool call abcd cannot see, and the live capture checks where the host's count resets (grounds: plan review round 2, choice (b); the host's hooks page on the Stop cap)
