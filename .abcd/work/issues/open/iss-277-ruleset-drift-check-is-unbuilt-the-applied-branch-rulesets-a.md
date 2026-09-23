---
schema_version: 1
id: "iss-277"
slug: "ruleset-drift-check-is-unbuilt-the-applied-branch-rulesets-a"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "manual-capture"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (planning owed: ruleset drift check belongs to itd-92, still in drafts). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

Ruleset drift check is unbuilt: the applied branch rulesets are mirrored under .abcd/work/rulesets/ but nothing diffs the live rulesets against the committed JSON, so drift is invisible until a human re-reads the console; itd-92 owns the verb