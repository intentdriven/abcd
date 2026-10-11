---
schema_version: 1
id: "iss-2608261133091022"
slug: "invariant-12-sessionend-clause-outlived"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-round-8"
found_at: ".abcd/development/brief/02-constraints/03-invariants.md:37"
resolution: "invariant 12 asserts the SessionEnd no-network clause present-tense, spc-35 mirrored bullet updated, adr-46 consequences record the condition as met at baf0551"
impact: internal
resolved_by:
  commit: "6206f9d6"
---

invariant 12 and spc-35 still withhold the SessionEnd no-network clause pending iss-2608210934566223, which resolved with a pinning test, so the register under-asserts a property the tree holds