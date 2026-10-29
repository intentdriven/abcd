---
schema_version: 1
id: "iss-2610262034572130"
slug: "the-capture-surface-page-counts-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 536 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/06-capture.md:74"
remedy: "none (filed automatically)"
related_intents: [itd-2609221656361680]
finding_ref: cfk-a1867b1f355abd1e
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. Read in full, one page says who composes the record and the other says who triggers the filing, so no statement of either is false."
---

The capture surface page counts a promoted inbox report among abcd's automatic filers; itd-2609221656361680 says nothing is filed until a person or a session acts and rules automatic filing out.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/06-capture.md:74` — "abcd's own automatic filers: the consistency pass, every"
- End B: `.abcd/development/intents/shipped/itd-2609221656361680-a-repository-abcd-manages-reports-back.md:45` — "nothing is filed until a person or a session acts"

06-capture.md lists, among abcd's own automatic filers, every promoted inbox report, whose remedy is written as `none (filed automatically)`, a value it says always means a machine filed the record; it also calls the inbox one of the ledger's two unattended writers. The inbox intent says nothing is filed until a person or a session acts and puts any automatic filing out of scope. The word is used in two senses, a record abcd composes versus a filing nobody asked for, so the record a person's promote writes states that it was filed automatically, which the intent says never happens.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 536.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. Read in full, one page says who composes the record and the other says who triggers the filing, so no statement of either is false.
