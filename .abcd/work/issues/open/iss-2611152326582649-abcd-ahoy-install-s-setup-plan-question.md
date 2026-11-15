---
schema_version: 1
id: "iss-2611152326582649"
slug: "abcd-ahoy-install-s-setup-plan-question"
severity: "major"
category: "ux"
source: "agent-finding"
found_during: "itd-2610301914350541 intent audit (ac-23)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/setup_plan.go"
deferred_after: "v0.16.2"
deferral_reason: "Shortening the plan changes what setup's first screen says, at a terminal and through the plugin page alike, and wants its own design and review; found at the close of itd-2610301914350541 and carried one cycle (dcn-2611152337093573)."
remedy: "Quote the plan shorter as one decision (group counts in place of every item, the full list behind the ? help), so it fits the default and the floor, and delete the lines in setupPlanLimitsOwed"
---

abcd ahoy install's setup plan question is taller than abcd's own rows limit: for a fresh repository it measures 63 rows (question text 44) against the default 32, and with nothing to change it measures 25 rows, over the 24-row floor a repository may set. The plan is drawn at a terminal, and `commands/ahoy.md` has the host relay it through its question tool as the material of the plan screen (`Setup Q1`). abcd's check admits a rows overrun with a note, so in a host session the plan reaches the person cut off at the window, the failure itd-2610301914350541 exists to prevent; at a terminal the drawn screen overruns the same way. Found when itd-2610301914350541's audit (ac-23) asked for the plan to be held at the floors; the test now owes these findings by name in setupPlanLimitsOwed.
