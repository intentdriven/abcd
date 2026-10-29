---
schema_version: 1
id: "iss-2610262042284906"
slug: "disposition-has-four-states-in-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1071 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/ledger/disposition.md:4"
remedy: "none (filed automatically)"
related_intents: [itd-2610141956319695]
finding_ref: cfk-33a66a4b7f27f321
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262033006141. Same contradiction as iss-2610262033006141: the deferred-finding discipline and the launch receipts protocol describe one release-receipt vocabulary."
duplicates: [iss-2610262033006141]
---

Disposition has four states in the glossary; the deferred-finding discipline gives it three different words

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/ledger/disposition.md:4` — "in one of four states: accepted, rejected, declined or held"
- End B: `.abcd/development/intents/disciplines/itd-2610141956319695-a-deferred-finding-is-its-own-captured.md:35` — "In a release receipt the three states are the three words of the finding's `disposition`, spelled exactly"

The glossary defines a disposition as one of accepted, rejected, declined or held (and, for scope conditions, survived, narrowed, falsified or untested). The deferred-finding discipline makes disposition a release-receipt field whose only words are fixed, invalid and deferred. One field name now carries a third vocabulary the glossary does not register.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1071.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262033006141. Same contradiction as iss-2610262033006141: the deferred-finding discipline and the launch receipts protocol describe one release-receipt vocabulary.
