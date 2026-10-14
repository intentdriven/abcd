---
schema_version: 1
id: "iss-2610142200389875"
slug: "the-reservations-ruleset-forbids"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "a peer's read of the spc-2610031753359849 design"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/work/rulesets/reservations-protection.json"
remedy: "Decide with itd-118: a sweep of tombstoned reservation branches older than N days through a ruleset bypass for one named automation, or the ruleset amended to allow deleting a tombstoned tip."
---

The reservations ruleset forbids deleting refs/heads/abcd-reserved/** with no bypass, so every spent reservation branch stays on the forge forever, counting against its per-repository branch limits and against the intent's 'needs occasional clean-up' (decision 10 of itd-2609091034175565); the rulesets README notes the itd-118 sweep needs the ruleset amended first.
