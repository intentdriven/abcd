---
schema_version: 1
id: "iss-2608211849583208"
slug: "intents-default-persona-rule"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "bughunt-round-6"
found_at: "internal/core/rules/defaults/rules.json:51"
resolution: ".abcd/rules.json overrides INTENTS to point authors at the registry"
impact: internal
resolved_by:
  commit: "1395e72"
---

the bundled INTENTS default rule (Personas are always Alice, Bob, Carol) reaches abcd own intent authors because .abcd/rules.json never overrides INTENTS, contradicting the armed 14-persona registry