---
schema_version: 1
id: "iss-2610262031159277"
slug: "02-context-lists-grill-and-ship-as"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 106 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/02-context.md:11"
remedy: "none (filed automatically)"
finding_ref: cfk-852a6f85b022375a
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262030322001. Same context sentence naming ship and grill as design targets, set against the build sequence's statement that neither exists."
duplicates: [iss-2610262030322001]
---

02-context lists grill and ship as remaining intent design targets; the build sequence says there is no intent ship and no grill sub-verb, shipping running through spec close and grilling through the planning interview.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/02-context.md:11` — "with `refine` / `grill` / `ship` / `shape` remaining design targets"
- End B: `.abcd/development/brief/06-delivery/01-build-sequence.md:99` — "one or more specs — so there is no `intent ship`. There is no `grill` sub-verb"

02-context gives the intent command its sub-verbs 'with `refine` / `grill` / `ship` / `shape` remaining design targets' and describes ship as driving the spec to completion. The build sequence says an intent moves to shipped/ as the close-hook of abcd spec close, 'so there is no `intent ship`. There is no `grill` sub-verb', the grilling being the planning interview under itd-94. One document holds both verbs as designed and pending; the other holds the design settled without them.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 106.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262030322001. Same context sentence naming ship and grill as design targets, set against the build sequence's statement that neither exists.
