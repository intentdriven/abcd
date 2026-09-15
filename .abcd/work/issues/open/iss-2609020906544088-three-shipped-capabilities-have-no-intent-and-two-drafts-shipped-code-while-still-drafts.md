---
schema_version: 1
id: "iss-2609020906544088"
slug: "three-shipped-capabilities-have-no-intent-and-two-drafts-shipped-code-while-still-drafts"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "phase-8-planning-2026-09-02"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents"
---

Found while readying Phase 8: the generated CLI reference (docs/reference/cli/commands.md), the surface_coverage record-lint rule and the brief-to-surface crosscheck gate were all shipped under issue ids with no intent record, and itd-60 and itd-85 each shipped substantial code while sitting in drafts/ (itd-60's deterministic docs lint is live as abcd docs lint; itd-85's delivery is on main). Phase 8's expectation is that the record is the shipped state, and these are shipped capabilities the record does not name as intents, so a product thinker reading the brief and the intents together still cannot see them as delivered. Directions, none adopted: file a shipped intent per capability by hand with an audit note saying it was delivered before the ceremony existed; or list them in the Phase 8 file as pre-ceremony deliveries and leave the intents as is; or resolve the gap through the doc-fidelity gate itself once it exists, since it will name every surface the brief describes without an intent.
