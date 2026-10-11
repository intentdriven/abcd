---
schema_version: 1
id: "iss-389"
slug: "the-examples-use-reserved-identifiers"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-round-3"
found_at: ".abcd/development/principles/examples-use-reserved-identifiers.md"
resolution: "record the iss-154 lint as enforced in the principle and point the owed promotion at iss-390"
impact: fix
---

the examples-use-reserved-identifiers principle still says a fixture-identifier lint would make it a discipline — the allowlist-inversion lint shipped (iss-154) and cites the principle by name, so the Promotion paragraph misdirects an agent to build an existing rule
## Evidence

- `.abcd/development/principles/examples-use-reserved-identifiers.md:45-46` ("Promotion.") — "A docs-lint / privacy-hygiene rule that flags any committed network identifier outside the reserved ranges (the allowlist inversion recorded in iss-154) would make this a discipline."
- The rule shipped: `internal/adapter/scanner/network.go:10-14` implements the allowlist inversion and cites this principle by name; `internal/core/repolint/rule_privacy.go:100-102` consumes it as the armed `privacy-hygiene` rule; iss-154 is resolved — "privacy-hygiene now flags network identifiers outside the reserved documentation ranges".
- Wrong-action risk: an agent walking the promotion path sets out to build an already-shipped lint; the file under-claims live enforcement — the mirror image of the enforcement-claims-are-facts class (iss-37, iss-142).
- Refuter verdict: PARTIALLY CONFIRMED (minor) — the "Live instance" sentence ("no fixture-identifier equivalent exists yet") was REFUTED as still true: it is about a committed fixture registry analogous to personas.json, which genuinely does not exist. Only the Promotion paragraph is stale; the owed rung move is iss-390.
