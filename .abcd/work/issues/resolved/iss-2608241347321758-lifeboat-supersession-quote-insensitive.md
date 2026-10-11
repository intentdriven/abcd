---
schema_version: 1
id: "iss-2608241347321758"
slug: "lifeboat-supersession-quote-insensitive"
severity: "minor"
category: "architectural-insight"
source: "review-followup"
found_during: "pr-294"
found_at: "internal/core/lifeboat/graveyard_abandoned.go"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Keep the quote-insensitive null sentinel for lifeboat supersession as a documented heuristic, or drop the pre-IsNull unquote?"
resolution: "gvSupersededADRs asks frontmatter.IsNull of the raw superseded_by scalar, as IsNull's contract and lint's provenance reader require: a quoted null spelling is a string and reports the ADR superseded, a quoted empty string stays absent (TestAbandonedQuotedNullIsAStringOnTheLifeboatPath)."
impact: fix
resolved_by:
  commit: "888d2174f"
---

gvSupersededADRs unquotes superseded_by BEFORE frontmatter.IsNull, so a quoted null (`"NULL"`) reads as absent on the lifeboat path even though YAML semantics make it a string -- decide whether lifeboat supersession wants quote-insensitive sentinel semantics, document it as an explicit lifeboat heuristic if kept, or drop the pre-IsNull unquote; surfaced by external review of pr-294 takeover 2026-08-24

## Grounds

- pursued: every reader asks null of the raw scalar, so a quoted null is a value on the lifeboat path as elsewhere; an accepted ADR with superseded_by "NULL" missing from the graveyard would show it wrong
