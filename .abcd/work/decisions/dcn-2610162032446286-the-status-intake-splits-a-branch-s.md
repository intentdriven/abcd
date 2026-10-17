---
id: dcn-2610162032446286
slug: the-status-intake-splits-a-branch-s
date: 2026-10-16
---
2026-10-16 — The status intake splits a branch's share between its newest and its oldest unread commits whenever the last intake left the branch unread: one the cap cut short in a walk, and one left without a walk once the cap was spent. The newest side takes the rounded-down half and the oldest the rest, so a share of one takes the oldest unread commit and every walk of a branch left moves its read mark. No tip is kept as asked because an app pushed it: a branch leaves the forge's queue once every standing announcement on it has a verdict, so an announcement an intake reads later at the same tip is still asked about. This refines dcn-2610161949032458 (spc-2610150728396006 step 3, on review), which named only a branch the cap cut short and kept a tip an app pushed as asked until it moved.
