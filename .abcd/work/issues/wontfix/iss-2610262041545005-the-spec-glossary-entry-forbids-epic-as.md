---
schema_version: 1
id: "iss-2610262041545005"
slug: "the-spec-glossary-entry-forbids-epic-as"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1033 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/spec.md:6"
remedy: "none (filed automatically)"
related_intents: [itd-34]
finding_ref: cfk-1b2d88550d1466be
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. A shipped intent's frontmatter vocabulary from the flow-next design is not live prose, and no code reads epic_id; the glossary's forbidden word concerns current wording."
---

The spec glossary entry forbids 'epic' as a synonym; itd-34 has bundle members share an epic_id that holds a spec id.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/spec.md:6` — "forbidden_synonyms: ["sprint", "milestone", "project", "feature", "epic"]"
- End B: `.abcd/development/intents/shipped/itd-34-three-intent-kinds.md:44` — "they share an `epic_id` and a `bundle: <bundle-id>` frontmatter field"

The spec entry lists 'epic' among its forbidden synonyms, and the glossary README says this repository enforces that word. itd-34 has bundle members 'share an epic_id' and writes 'epic_id: spc-N' and 'epic.intent', which uses the forbidden word for a spec in the very frontmatter the bundle relies on.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1033.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. A shipped intent's frontmatter vocabulary from the flow-next design is not live prose, and no code reads epic_id; the glossary's forbidden word concerns current wording.
