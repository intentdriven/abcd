---
schema_version: 1
id: "iss-2611272325440853"
slug: "status-json-page-omits-reservation"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/status.md"
remedy: "Name the four fields in the status page's reservations bullet, say settled is the peers listing's alone, and add a test that every top-level JSON field of the status and peers documents is named on its command page"
resolution: "commands/status.md names release_withdrawals, lost_withdrawals, ceiling and drain_in_work with their meanings and says settled is the peers listing's alone; a test checks that every top-level field of the status and peers --json documents is named on its page"
impact: fix
---

commands/status.md's --json section omits release_withdrawals, lost_withdrawals, ceiling and drain_in_work, which abcd status --json carries from the reservations block it shares with abcd peers, and it calls its reservation fields the same fields /abcd:peers lists though peers also carries settled, which status never fills
