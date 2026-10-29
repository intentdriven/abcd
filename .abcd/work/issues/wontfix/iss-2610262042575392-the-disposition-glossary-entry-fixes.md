---
schema_version: 1
id: "iss-2610262042575392"
slug: "the-disposition-glossary-entry-fixes"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1104 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/ledger/disposition.md:4"
remedy: "none (filed automatically)"
related_intents: [itd-180]
finding_ref: cfk-c17b8090c669a94e
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-180's Audit Notes list 'a five-state disposition vocabulary' as diverged and 'a fifth disposition state' as missing, and the schema comment says the discrepancy is carried, not resolved. The glossary and the code agree on four, so the shipped intent's count is a recorded miscount."
---

The disposition glossary entry fixes four states; itd-180 calls the same disposition vocabulary five-state.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/ledger/disposition.md:4` — "in one of four states: accepted, rejected, declined or held."
- End B: `.abcd/development/intents/shipped/itd-180-a-cold-reading-s-findings-land-as.md:28` — "five-state vocabulary whose availability varies by position"

The glossary defines a disposition as written 'in one of four states: accepted, rejected, declined or held', and says an undispositioned item is outstanding, which is never a state. itd-180 describes the same disposition record as drawn from 'a five-state vocabulary' (and 'five states' in its scope) while naming only those same four values. An implementer sizing the enum, or a lint counting the states, cannot satisfy both counts, and neither document names a fifth value that would reconcile them.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1104.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-180's Audit Notes list 'a five-state disposition vocabulary' as diverged and 'a fifth disposition state' as missing, and the schema comment says the discrepancy is carried, not resolved. The glossary and the code agree on four, so the shipped intent's count is a recorded miscount.
