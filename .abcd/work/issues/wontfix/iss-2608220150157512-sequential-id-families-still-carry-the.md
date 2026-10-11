---
schema_version: 1
id: "iss-2608220150157512"
slug: "sequential-id-families-still-carry-the"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "abcdev-site session close-out 2026-08-22"
found_at: ".abcd/development/decisions/adrs"
wontfix_reason: "same remediation path as iss-2608210737260468 (adr-45 ruling 3): one timestamp-mint tracker, distinct framings merged"
---

ADRs, intents and specs still mint sequential ids (adr-NNNN, itd-N, spc-N), so two parallel branches minting in the same family produce add/add collisions on the same id — the exact mechanism adr-45 removed for issue captures, whose timestamp-numeric ids produced zero conflicts across twelve same-day captures on 2026-08-22 while the sequential families carry the standing risk. adr-45 rollout note 3 already names the path: one allocator, per-family adoption as configuration. Extending timestamp minting to the remaining families closes the class