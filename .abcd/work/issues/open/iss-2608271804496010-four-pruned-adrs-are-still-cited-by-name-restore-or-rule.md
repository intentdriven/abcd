---
schema_version: 1
id: "iss-2608271804496010"
slug: "four-pruned-adrs-are-still-cited-by-name-restore-or-rule"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "structural consistency review of .abcd/ and docs/ (2026-08-27)"
found_at: ".abcd/development/decisions/adrs"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (ruling owed: restore adr-4/8/16/18 as superseded records or rule that body-prose citations do not count for retention). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

eight pruned ADR ids are still cited by later records, and for four of them (adr-4, adr-8, adr-16, adr-18) the citations are live by-name references that the decisions charter's retention clause says keep a record restorable: either restore those four from the pruning commits as status superseded records, or record a maintainer ruling that body-prose citations do not count as cite for the retention clause. Needs the maintainer's call — the two remedies diverge and neither is obviously right.