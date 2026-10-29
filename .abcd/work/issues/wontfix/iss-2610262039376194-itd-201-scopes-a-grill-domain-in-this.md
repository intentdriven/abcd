---
schema_version: 1
id: "iss-2610262039376194"
slug: "itd-201-scopes-a-grill-domain-in-this"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 873 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-201-every-question-abcd-s-agents-put-to-a.md:83"
remedy: "none (filed automatically)"
related_intents: [itd-201]
finding_ref: cfk-a825bdfbdab1db01
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. This repository's .abcd/rules.json declares no GRILL key; the intent records the replacement in its criteria."
---

itd-201 scopes a GRILL domain in this repository's `.abcd/rules.json` kept in step with the bundled copy; the configuration chapter says GRILL is generated from one Go source and abcd's own repository declares no override of it

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-201-every-question-abcd-s-agents-put-to-a.md:83` — "The GRILL rule domain in this repository's `.abcd/rules.json`, and the"
- End B: `.abcd/development/brief/05-internals/03-configuration.md:963` — "abcd's own repository included, which declares no override of it"

itd-201's scope puts the GRILL rule domain in this repository's `.abcd/rules.json` and adds a check that the bundled default and the repository's copy do not drift. The configuration chapter says GRILL's rules are written once in `internal/core/question`, a bundled `rules.json` that declares it by hand panics at load, and abcd's own repository declares no override of it. So the brief has no repository copy for that drift check to compare against.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 873.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. This repository's .abcd/rules.json declares no GRILL key; the intent records the replacement in its criteria.
