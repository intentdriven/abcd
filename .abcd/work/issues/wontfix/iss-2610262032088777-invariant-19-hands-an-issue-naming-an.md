---
schema_version: 1
id: "iss-2610262032088777"
slug: "invariant-19-hands-an-issue-naming-an"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 227 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/03-invariants.md:175"
remedy: "none (filed automatically)"
related_intents: [itd-2610161016394967]
finding_ref: cfk-03067781515c3911
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The code hands such an issue back as promoted, and the intent's own audit note says so. The loose word in a dated decision needs no edit."
---

Invariant 19 hands an issue naming an intent back by the rule `promoted`; itd-2610161016394967, citing invariant 19, calls such an issue ineligible.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/03-invariants.md:175` — "an issue whose `related_intents` names an intent is handed back by the rule `promoted`"
- End B: `.abcd/development/intents/shipped/itd-2610161016394967-several-people-each-on-their-own-github.md:108` — "an issue naming an intent is ineligible (brief invariant 19)"

Invariant 19 gives every excluded issue exactly one disposition: handed back, listed as ineligible, or skipped. It places an issue whose `related_intents` names an intent under handed back, by the rule `promoted`, and judges a consistency-pass record by its other fields. The drain intent cites the same invariant yet calls every issue naming an intent ineligible, which is a different disposition in the invariant's own terms.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 227.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The code hands such an issue back as promoted, and the intent's own audit note says so. The loose word in a dated decision needs no edit.
