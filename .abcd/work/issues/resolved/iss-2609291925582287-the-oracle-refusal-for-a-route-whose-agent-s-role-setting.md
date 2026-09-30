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
remedy: "Wording only, per the product thinker's ruling BR1 of 2026-09-29 (refuse and name the one setting to add): drop the not-yet-decided clause from the refusal in providerLeg.take (internal/core/oracle/resolve.go) and from the comments in resolve.go and tier.go and the adapters brief chapter, keep naming the oracle.roles.<agent> setting to add, and pin the decided wording in refusal_test.go with the old clause asserted absent."
resolution: "The unpointed-route refusal in providerLeg.take states the decided behaviour of ruling BR1 (2026-09-29): the route names no model and is refused, naming oracle.roles.<agent> as the setting to point. The not-yet-decided clause is gone from the refusal, the comments in resolve.go and tier.go, and the adapters brief chapter; refusal_test.go pins the new wording and asserts the old clause absent."
impact: fix
resolved_by:
  commit: "e7a659724"
---

The oracle refusal for a route whose agent's role setting does not point at the routed connection still says which model such a route asks for is not yet decided, and its doc comment says the record does not yet decide it. The product thinker ruled on 2026-09-29 (BR1) that such a route is refused, naming the one setting to add, as built in e1c4e38ad and 35100512e, so the refusal is now the decided behaviour and the words not yet decided tell the person a ruling is still coming when none is. The fix is wording only: drop the not-yet-decided clause from the error in internal/core/oracle/resolve.go (take) and the comments in resolve.go and tier.go, keeping the named setting, and adjust any test that pins the old sentence.

## Grounds

- pursued: the refusal a person sees for a route whose agent's role does not point at the routed connection names the setting to add and no longer says a ruling is coming; TestResolveRefusesALegTheAgentsRoleDoesNotPointAt would show it wrong by failing on a refusal that names 'not yet decided' or 'undecided', or omits 'a route that names no model is refused'.
