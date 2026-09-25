---
schema_version: 1
id: "iss-2608241347321758"
slug: "lifeboat-supersession-quote-insensitive-null-heuristic"
severity: "minor"
category: "architectural-insight"
source: "review-followup"
found_during: "pr-294"
found_at: "internal/core/lifeboat/graveyard_abandoned.go"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Keep the quote-insensitive null sentinel for lifeboat supersession as a documented heuristic, or drop the pre-IsNull unquote?"
---

gvSupersededADRs unquotes superseded_by BEFORE frontmatter.IsNull, so a quoted null (`"NULL"`) reads as absent on the lifeboat path even though YAML semantics make it a string -- decide whether lifeboat supersession wants quote-insensitive sentinel semantics, document it as an explicit lifeboat heuristic if kept, or drop the pre-IsNull unquote; surfaced by external review of pr-294 takeover 2026-08-24