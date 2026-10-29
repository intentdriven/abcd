---
schema_version: 1
id: "iss-2610262034528274"
slug: "itd-34-links-a-bundle-member-to-its"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 529 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-34-three-intent-kinds.md:52"
remedy: "none (filed automatically)"
related_intents: [itd-34]
finding_ref: cfk-dce44517292ba38b
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262045336833. Same A wording (epic_id on bundle members) and same root cause; the bundle code writes spec_id (internal/core/intent/bundle.go)."
duplicates: [iss-2610262045336833]
---

itd-34 links a bundle member to its shared spec through epic_id and an epic.intent list; the intent surface page links it through spec_id and the spec's intents list.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-34-three-intent-kinds.md:52` — "each intent's `epic_id: spc-N` and `bundle: <bundle-id>`"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:235` — "links each spec_id to the shared spec"

itd-34 says each bundle member carries an `epic_id` naming the shared spec, beside `bundle`, and that the spec lists its members as `epic.intent`. 05-intent.md says the bundle command links each member's spec_id to the shared spec and mints the spec with an `intents:` list beside `intent:`. Two names for the one intent-to-spec link: a member written to itd-34's shape carries a key the lifecycle rules and the readers described on the surface page do not read.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 529.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262045336833. Same A wording (epic_id on bundle members) and same root cause; the bundle code writes spec_id (internal/core/intent/bundle.go).
