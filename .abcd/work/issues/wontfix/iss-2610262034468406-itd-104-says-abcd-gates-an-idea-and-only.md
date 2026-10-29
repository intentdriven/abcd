---
schema_version: 1
id: "iss-2610262034468406"
slug: "itd-104-says-abcd-gates-an-idea-and-only"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 522 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-104-abcd-gates-a-new-idea-before-it-becomes.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-104]
finding_ref: cfk-084488b13f75fba2
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The shipped intent's later sections settle that capture and intent create need no ideate step, and its audit (ac-1, MET) cites the test that ideate is a pointer and not a gate."
---

itd-104 says abcd gates an idea and only a survivor graduates to a draft intent; the intent chapter calls ideate optional beside a create that drafts any quoted text.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-104-abcd-gates-a-new-idea-before-it-becomes.md:17` — "and only a surviving idea graduates to a draft intent"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:317` — "`abcd ideate` runs the optional admission gauntlet and records the verdict either way"

itd-104 promises that abcd gates a new idea before it can mint a single record and that only a surviving idea graduates to a draft intent. 05-intent's bare status calls abcd ideate the optional admission gauntlet, and its canonical create mints a draft intent from any leading quoted text with no admission step. Under the chapter a draft intent can exist that never met the gate, which the intent says cannot happen.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 522.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The shipped intent's later sections settle that capture and intent create need no ideate step, and its audit (ac-1, MET) cites the test that ideate is a pointer and not a gate.
