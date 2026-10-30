---
schema_version: 1
id: "iss-2610291130262302"
slug: "the-retrospective-seed-counts-an-intent"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "applying the hand-audit rule to the retrospective seed (iss-2610241431221889)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reflect/audit.go"
refines: ["iss-2610241431221889"]
remedy: "Have the seed treat a dead-lettered review as unaudited and offer its re-emit, through the same shared reader the owed listing uses, with a test that a dead-lettered intent appears among the seed's unaudited intents."
---

The retrospective seed counts an intent whose fidelity review was dead-lettered as audited, and offers no audit for it, while the list of reviews owed reports the same intent as unreviewed with its reason. A review that never landed reads as an audit in one reader and not in the other, which runs against the rule of 29 October that both readers judge audited by one rule.
