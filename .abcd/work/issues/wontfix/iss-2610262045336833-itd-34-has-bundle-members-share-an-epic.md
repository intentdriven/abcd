---
schema_version: 1
id: "iss-2610262045336833"
slug: "itd-34-has-bundle-members-share-an-epic"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1282 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-34-three-intent-kinds.md:44"
remedy: "none (filed automatically)"
related_intents: [itd-34, itd-43]
finding_ref: cfk-3fee275c313420e4
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Both shipped records were true when written and the later rename names the field; internal/core/intent/bundle.go links each member's spec_id."
---

itd-34 has bundle members share an epic_id field; itd-43 says the intent field is spec_id and epic is gone

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-34-three-intent-kinds.md:44` — "they share an `epic_id` and a `bundle: <bundle-id>` frontmatter field"
- End B: `.abcd/development/intents/shipped/itd-43-epic-to-spec-terminology.md:18` — "abcd's `spec_id` intent-frontmatter field already carries the concept"

itd-34 defines bundle-member frontmatter with an epic_id (and epic.intent on the shared spec). itd-43 states that the intent-frontmatter field is spec_id, that the epic_id to spec_id rename is already complete, and that epic is a forbidden synonym. A builder of the bundle path reads a field name the glossary forbids.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1282.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Both shipped records were true when written and the later rename names the field; internal/core/intent/bundle.go links each member's spec_id.
