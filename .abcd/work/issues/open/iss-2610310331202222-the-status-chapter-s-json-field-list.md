---
schema_version: 1
id: "iss-2610310331202222"
slug: "the-status-chapter-s-json-field-list"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/39-status.md"
remedy: "Add the two holder-id refusal fields to the JSON list in 04-surfaces/39-status.md."
refines: [iss-2610171626364475]
---

The status chapter's JSON field list omits holder_ids_refused and holder_id_refused, which status --help says it carries.
