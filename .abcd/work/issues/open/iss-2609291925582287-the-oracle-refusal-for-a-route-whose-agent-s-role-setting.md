---
schema_version: 1
id: "iss-2609291925582287"
slug: "the-oracle-refusal-for-a-route-whose-agent-s-role-setting"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/oracle/resolve.go"
---

The oracle refusal for a route whose agent's role setting does not point at the routed connection still says which model such a route asks for is not yet decided, and its doc comment says the record does not yet decide it. The product thinker ruled on 2026-09-29 (BR1) that such a route is refused, naming the one setting to add, as built in e1c4e38ad and 35100512e, so the refusal is now the decided behaviour and the words not yet decided tell the person a ruling is still coming when none is. The fix is wording only: drop the not-yet-decided clause from the error in internal/core/oracle/resolve.go (take) and the comments in resolve.go and tier.go, keeping the named setting, and adjust any test that pins the old sentence.
