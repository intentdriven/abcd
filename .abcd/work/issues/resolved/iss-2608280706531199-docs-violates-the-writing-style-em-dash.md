---
schema_version: 1
id: "iss-2608280706531199"
slug: "docs-violates-the-writing-style-em-dash"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-review"
found_at: "docs/"
resolution: "armed punctuation/em-dash-in-list-item as a docs-lint blocker (plus advisory spelling/* warns) and fixed all 17 violations; writing-style labels updated to match shipped machinery"
impact: additive
---

docs/ violates the writing-style em-dash rule in list items across explanation, how-to and reference pages; the rule is review-only, so nothing gates it