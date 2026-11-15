---
schema_version: 1
id: "iss-2611152326582649"
slug: "abcd-ahoy-install-s-setup-plan-question"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "itd-2610301914350541 intent audit (ac-23)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/setup_plan.go"
remedy: "Quote the plan shorter as one decision (group counts in place of every item, the full list behind the ? help), so it fits the default and the floor, and delete the lines in setupPlanLimitsOwed"
---

abcd ahoy install's setup plan question is taller than abcd's own rows limit: for a fresh repository it measures 63 rows (question text 44) against the default 32, and with nothing to change it measures 25 rows, over the 24-row floor a repository may set. The plan is put only to a person at a terminal, never relayed to a host's question tool, so the overrun shows in the drawn terminal screen rather than in a host session. Found when itd-2610301914350541's audit (ac-23) asked for the plan to be held at the floors; the test now owes these findings by name in setupPlanLimitsOwed.
