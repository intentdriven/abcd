---
id: dcn-2609010000000007
slug: adrs-take-timestamp-ids-through-the-same
date: 2026-09-01
---
2026-09-01 — ADRs take timestamp ids through the same seam as issues (maintainer, the turn adr-45 ruling 3 deferred). Two branches minted `0055` and `0056` on the same day for different decisions, so the hand-numbered ordinal has the add/add collision the timestamp mint removed for captures. From here an ADR id is minted by the binary as `adr-<timestamp>` with a filename ordered by that stamp; `0001`–`0058` keep their ids and filenames, and nothing is renumbered. Rejected: minting ordinals from one checkout only (a convention, not a gate), and renumbering at merge (the collision tonight was found by reading, and a reader is not a gate). The mint verb and the lint rule that admits both shapes are implementation, tracked in the ledger.
