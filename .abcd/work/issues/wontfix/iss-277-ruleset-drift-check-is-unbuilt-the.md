---
schema_version: 1
id: "iss-277"
slug: "ruleset-drift-check-is-unbuilt-the"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "manual-capture"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Does the ruleset drift check belong in itd-92's scope when it is planned (M4)?"
wontfix_reason: "Duplicate of itd-92 (draft): its scope names the live-vs-mirror drift diff against .abcd/work/rulesets/ as closing iss-277, normalising server-assigned ids and timestamps, and its acceptance criteria report a drifted live ruleset as 'applied but drifted', naming the rule that moved. The deferral's question (does the drift check belong in itd-92) is answered by itd-92's own text."
---

Ruleset drift check is unbuilt: the applied branch rulesets are mirrored under .abcd/work/rulesets/ but nothing diffs the live rulesets against the committed JSON, so drift is invisible until a human re-reads the console; itd-92 owns the verb

## Grounds

- declined: the drift check is in itd-92's scope and acceptance; this would be wrong if itd-92 were planned without the drift diff
