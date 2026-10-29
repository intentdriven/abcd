---
schema_version: 1
id: "iss-2610290033556553"
slug: "the-load-preflights-setting-is-described"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "review of iss-2610282016250805's change, relayed by the coordinator 2026-10-28"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/load_slots.go"
remedy: "Say the setting is how many preflights the first gate admits to their test stages at once, with one race stage able to run beside them, everywhere the setting is defined, and correct the configuration chapter's range to 1 to 4"
resolution: "the setting is defined everywhere as how many preflights the first gate admits to their test stages at once, with one race stage able to run beside them, and the configuration chapter gives its range as 1 to 4"
impact: fix
---

The load.preflights setting is described as how many preflights may be in their test stages at once, in its refusal, the help, the command page, the bundled LOAD rule and the code comments, but since iss-2610282016250805 a preflight queued at its race gate takes no slot, so a race stage can run beside as many unit stages as the setting: it bounds what the first gate admits, not what runs. The configuration chapter of the brief still says the setting takes 1 or 2, where 4 is now the most.
