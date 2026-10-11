---
schema_version: 1
id: "iss-2608240815453131"
slug: "shipped-readme-names-retired-intent"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "bug-hunt loop round 5 (state issue #368), internal record-consistency hunt + independent adversarial refutation"
found_at: ".abcd/development/intents/shipped/README.md:6"
resolution: "Renamed intent-fidelity-reviewer to intent-auditor in shipped/README.md:6, matching the landed rename (spc-28) and the parent index that iss-334 already swept."
impact: internal
resolved_by:
  commit: "1a9d2b88837c2ec74b41d9ffd53a51b48ca1125a"
---

shipped-intents README names the retired intent-fidelity-reviewer agent (renamed to intent-auditor per spc-28); the sibling front-door the iss-334 sweep of the parent index missed