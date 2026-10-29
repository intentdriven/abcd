---
schema_version: 1
id: "iss-2610262044237764"
slug: "the-cold-reading-draft-introduces-a-cold"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1203 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-86-cold-reading-surface.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-86, itd-2609081951381895]
finding_ref: cfk-afc611ca145ba73b
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262039216588. Same contradiction: the draft introduces the cold reading as a surface abcd gains, while the position-dispatched reading (reading ingest --dispatch) already operates."
duplicates: [iss-2610262039216588]
---

The cold-reading draft introduces a cold reading as new, while a shipped record already dispatches cold readings by position

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-86-cold-reading-surface.md:17` — "abcd gains a cold reading: a pass that reads one design document as though it had never been seen before"
- End B: `.abcd/development/intents/shipped/itd-2609081951381895-abcd-ships-an-openai-compatible-api.md:75` — "A cold reading is sent by `reading ingest --dispatch <rdg-N>`, the parked run's bundle with the position's definition"

itd-86 presents the cold reading as a surface abcd gains, a single pass over one document. itd-2609081951381895 already routes cold readings by position through `reading ingest --dispatch <rdg-N>` and names four cold-reading positions. The surface is claimed by a draft and treated as operating by a shipped record, and the two shapes differ (one detection pass against four positions).

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1203.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262039216588. Same contradiction: the draft introduces the cold reading as a surface abcd gains, while the position-dispatched reading (reading ingest --dispatch) already operates.
