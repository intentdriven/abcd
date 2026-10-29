---
schema_version: 1
id: "iss-2610262042019151"
slug: "persona-names-identify-stakeholder"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1040 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/persona.md:4"
remedy: "none (filed automatically)"
related_intents: [itd-2610111707085098]
finding_ref: cfk-08bd3586f19a1b11
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262038329713. Same End A and same cause; decision 2 of itd-2610111707085098 is the product thinker's ruling that session names reuse the persona list."
duplicates: [iss-2610262038329713]
---

Persona names identify stakeholder archetypes in press releases; the session-naming intent reuses them, in list order, as names for live sessions.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/persona.md:4` — "A placeholder stakeholder character drawn from the abcd personas registry"
- End B: `.abcd/development/intents/shipped/itd-2610111707085098-the-status-line-should-show-which.md:57` — "Alice, Bob, Carol and on through the list, then Alice Two, Bob Two and so on"

The persona entry defines Alice, Bob, Carol and the rest as placeholder stakeholder characters from the personas registry, used in press releases and design documents and selected by role, never by name. The shipped session-naming intent assigns the same names in list order to running agent sessions. 'Alice' now means both a modelled archetype in a press release and a particular live session on the machine, and the selection rule is ignored. Some intents in the corpus use both senses in one document.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1040.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262038329713. Same End A and same cause; decision 2 of itd-2610111707085098 is the product thinker's ruling that session names reuse the persona list.
