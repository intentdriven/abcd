---
id: dcn-2607210000000002
slug: itd-67-s-a-phase-completed-since-the
date: 2026-07-21
---
2026-07-21 — itd-67's "a phase completed since the last launch -> minor" bump
heuristic is SUPERSEDED by itd-73's `impact` derivation (spc-10). The bump falls
out of the records' declared `impact`, not out of phase membership; spc-11
consumes the derived number and never computes one. Recorded so the two intents
do not both claim version selection.
