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
resolution: "Bare intent audit resolves --route for no agent, refusing it at exit 2 as --issue-drift does (TestIntentAuditListingRefusesRoute)."
impact: internal
resolved_by:
  commit: "293d7d4f"
---

bare intent audit (the read-only owed listing) accepts --route and silently drops it: the listing dispatches no agent, yet it never resolves the flag, so unlike --issue-drift it does not refuse; a merge-time gap between the owed listing and tier2's --route

## Grounds

- pursued: bare intent audit --route <x> exits 2 naming that the invocation dispatches none; a listing that still prints with the flag given would show it wrong
