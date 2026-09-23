---
schema_version: 1
id: "iss-2608241347321758"
slug: "lifeboat-supersession-quote-insensitive-null-heuristic"
severity: "minor"
category: "architectural-insight"
source: "review-followup"
found_during: "pr-294"
found_at: "internal/core/lifeboat/graveyard_abandoned.go"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (ruling owed: quote-insensitive null sentinel for lifeboat supersession, documented as a heuristic, or drop the pre-IsNull unquote). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

gvSupersededADRs unquotes superseded_by BEFORE frontmatter.IsNull, so a quoted null (`"NULL"`) reads as absent on the lifeboat path even though YAML semantics make it a string -- decide whether lifeboat supersession wants quote-insensitive sentinel semantics, document it as an explicit lifeboat heuristic if kept, or drop the pre-IsNull unquote; surfaced by external review of pr-294 takeover 2026-08-24