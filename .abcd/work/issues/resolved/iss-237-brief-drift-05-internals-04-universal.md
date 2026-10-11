---
schema_version: 1
id: "iss-237"
slug: "brief-drift-05-internals-04-universal"
severity: "major"
category: "drift"
source: "agent-finding"
found_during: "intent-planning-prep"
found_at: ".abcd/development/brief/05-internals/04-universal-patterns.md"
resolution: "the docfidelity/spc-74-76/validation-disciplines passages are reframed as planned draft-intent work"
impact: internal
resolved_by:
  commit: "b760a67c"
---

Brief drift: 05-internals/04-universal-patterns.md and 01-product/03-mental-model.md describe internal/core/docfidelity, the spc-74/75 marker grammar, and a .abcd/validation_disciplines/ registry as current state; none exists in the Go tree and those spc ids belong to the pre-adr-21 spec space (live specs are spc-1..spc-22).