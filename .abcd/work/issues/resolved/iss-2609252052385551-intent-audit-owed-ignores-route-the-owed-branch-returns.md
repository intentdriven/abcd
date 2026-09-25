---
schema_version: 1
id: "iss-2609252052385551"
slug: "intent-audit-owed-ignores-route-the-owed-branch-returns"
severity: "major"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/intent_drain.go"
resolution: "The --owed branch resolves the auditor's route and emits the head through ReEmitAuditWith with the routing section; next carries the routing member (TestIntentAuditOwedCarriesTheRouting)."
impact: internal
resolved_by:
  commit: "88020d77"
---

intent audit --owed ignores --route: the --owed branch returns before auditRoute.resolve and emits the head through the option-less ReEmitAudit, so the drain's request carries no Routing section, its JSON next has no routing member, and --owed --route <x> is accepted and silently dropped, unlike audit <itd-N>

## Grounds

- pursued: --owed --route intent-auditor=economy puts tier economy in the request's Routing section and in next.routing; a drain request without the section, or an override that does not reach it, would show it wrong
