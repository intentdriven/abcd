---
id: dcn-2609010000000005
slug: the-brief-is-the-shipped-state
date: 2026-09-01
---
2026-09-01 — The brief is the shipped state, consolidated without a new intent (maintainer). The request "a product thinker must be able to understand the current shipped state of the repository by looking only at the brief" is already adr-5's decision and is delivered by two existing drafts, itd-60 (the doc-fidelity pass that gates spec close on the brief reflecting delivery) and itd-147 (generated surface blocks so a shape claim cannot drift), so no umbrella intent is filed; the drift records from the v0.7.0 receipts link to them and the two exact duplicate pairs close as duplicates. Added to the decision: brief, record and release stay in sync, with one legitimate lead — a brief edited after a release was cut is ahead of that release until the next cut, and the gap closes at the cut rather than between cuts. The work is sequenced as Phase 8, "the brief is the shipped state"; the design branch's unmerged "closed loop" phase takes 9 when it merges. itd-61, the reverse direction, stays adjacent as an open question of the phase.
