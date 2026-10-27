---
schema_version: 1
id: "iss-2610262043005827"
slug: "itd-192-s-verdict-rule-turns-on-this"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1108 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-192-an-acceptance-criterion-whose-producer.md:25"
remedy: "none (filed automatically)"
related_intents: [itd-192, itd-24]
finding_ref: cfk-01a24050a18f6a8d
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262042302710. Same unit problem in itd-192 as iss-2610262042302710, cited against itd-24."
duplicates: [iss-2610262042302710]
---

itd-192's verdict rule turns on 'this phase'; itd-24 records that phases are retired and the release is the unit.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-192-an-acceptance-criterion-whose-producer.md:25` — "Does this phase wire it?"
- End B: `.abcd/development/intents/shipped/itd-24-reflect-command.md:90` — "phases are retired, so `<phase-id>` reads as the release tag"

itd-192 decides between MET_WITH_CONCERNS and NOT_MET by one question, 'Does this phase wire it?'. itd-24 records the ruling that 'phases are retired' and that a phase now reads as a release tag. The discipline still keys its test on a retired sequencing unit and does not say whether 'this phase' now means the release, the intent's spec or something else, so an auditor applying it has no defined scope to judge wiring against.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1108.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262042302710. Same unit problem in itd-192 as iss-2610262042302710, cited against itd-24.
