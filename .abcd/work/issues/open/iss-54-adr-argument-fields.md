---
schema_version: 1
id: "iss-54"
slug: "adr-argument-fields"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "2026-07-09 practice/MVP/tool extraction"
found_at: ".abcd/development/decisions/adrs"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (ruling owed: adopt related_principles/confidence/revisit_when ADR fields; no ADR carries them). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

ADR frontmatter selectively gains three argument fields: related_principles (which standing principles the decision leans on), a confidence qualifier, and revisit_when — the rebuttal slot naming the conditions under which the decision no longer holds. The convention serves decision durability: an ADR that records only the choice invites re-litigation, while one that records its confidence and its expiry conditions tells a future session exactly when reopening is legitimate. Adopt the fields selectively per the 2026 template-comparison evidence rather than the full heavyweight argumentation set, which adds ceremony without proportionate recall value. Acceptance: new ADRs carry the three fields where they apply, and a revisit_when condition coming true is sufficient grounds to reopen without further debate.