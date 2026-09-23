---
schema_version: 1
id: "iss-2609230733540527"
slug: "itd-67-ac-4-promises-a-phase-completion-bump-tier-named-in"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "autonomous run 2026-09-23 fidelity audit"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md"
---

itd-67 ac-4 promises a phase-completion bump tier named in the launch report and a major bump only via an explicit --version <x.0.0>; delivered reality: the tier derives from record impact (internal/core/changelog/version.go), the report names the deciding impact and record and no phase, and the ship verb carries no --version flag at all — spc-11 records the supersession under Reconciliations but the intent's criterion was never amended, and the documented human override for the first 1.0.0 has no verb: the only path is a hand-written CHANGELOG heading. Found by the itd-67 fidelity audit (rcp-7af1556ce4f7).
