---
schema_version: 1
id: "iss-2609252054322226"
slug: "bare-intent-audit-the-read-only-owed-listing-accepts-route"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
---

bare intent audit (the read-only owed listing) accepts --route and silently drops it: the listing dispatches no agent, yet it never resolves the flag, so unlike --issue-drift it does not refuse; a merge-time gap between the owed listing and tier2's --route
