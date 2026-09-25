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
---

intent audit --owed ignores --route: the --owed branch returns before auditRoute.resolve and emits the head through the option-less ReEmitAudit, so the drain's request carries no Routing section, its JSON next has no routing member, and --owed --route <x> is accepted and silently dropped, unlike audit <itd-N>
