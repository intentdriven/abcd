---
schema_version: 1
id: "iss-2610241544508861"
slug: "the-default-branch-review-rule-differs"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "verifying forge settings, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/work/rulesets"
remedy: "Decide whether the main review ruleset should carry an admin bypass, then make the live ruleset (id 21045110), the mirror and the rulesets README agree; and correct the README's account of why the owner's pull requests merge without a review, which the removed bypass no longer explains."
---

The default-branch review rule differs between the committed mirror and the live ruleset: the mirror and the rulesets README record an admin bypass, but the live ruleset (id 21045110) has none, and its history shows the bypass was removed on 2026-08-24. The owner's pull requests still merge without a review (#1119 touched code-owned files), so the README's explanation of why is wrong, and the live ruleset carries a setting the mirror lacks.
