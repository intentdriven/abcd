---
schema_version: 1
id: "iss-277"
slug: "ruleset-drift-check-is-unbuilt-the-applied-branch-rulesets-a"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "manual-capture"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Does the ruleset drift check belong in itd-92's scope when it is planned (M4)?"
---

Ruleset drift check is unbuilt: the applied branch rulesets are mirrored under .abcd/work/rulesets/ but nothing diffs the live rulesets against the committed JSON, so drift is invisible until a human re-reads the console; itd-92 owns the verb