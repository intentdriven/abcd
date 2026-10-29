---
schema_version: 1
id: "iss-2610262031362638"
slug: "itd-190-says-the-discipline-template"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 152 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-190-the-claim-recording-gradient-an-intent-s.md:41"
remedy: "none (filed automatically)"
related_intents: [itd-190]
finding_ref: cfk-520740ee06c6670b
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262042332488. Same question as iss-2610262042332488, cited against the mental model, which says disciplines use Given-When-Then under a Rule heading."
duplicates: [iss-2610262042332488]
---

itd-190 says the discipline template carries no claim sections, while the mental model says disciplines carry Given-When-Then acceptance

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-190-the-claim-recording-gradient-an-intent-s.md:41` — "discipline-kind records are exempt — their template carries no claim sections."
- End B: `.abcd/development/brief/01-product/03-mental-model.md:85` — "Discipline-kind intents skip the press release but use the same Given-When-Then format under a `## Rule` heading"

itd-190 counts the Acceptance Criteria section as an intent's first claim section and exempts discipline-kind records because their template carries no claim sections. The mental model says discipline-kind intents skip the press release but use the same Given-When-Then format under a Rule heading. The two records disagree on whether a discipline record carries acceptance criteria.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 152.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262042332488. Same question as iss-2610262042332488, cited against the mental model, which says disciplines use Given-When-Then under a Rule heading.
