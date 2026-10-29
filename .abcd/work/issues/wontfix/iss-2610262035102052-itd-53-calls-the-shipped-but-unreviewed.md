---
schema_version: 1
id: "iss-2610262035102052"
slug: "itd-53-calls-the-shipped-but-unreviewed"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 553 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-53-review-queue-auto-drain-fidelity-gate.md:36"
remedy: "none (filed automatically)"
related_intents: [itd-53]
finding_ref: cfk-c1111b29b12f064c
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The re-scope says the gate reports and never blocks and hands the standing list to another record, so the consistency gate wording is read through the banner."
---

itd-53 calls the shipped-but-unreviewed listing a consistency gate; the intent chapter uses Consistency for the Role 2 cross-document pass

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-53-review-queue-auto-drain-fidelity-gate.md:36` — "A consistency gate / report that lists shipped intents whose latest fidelity review is missing or not-met"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:332` — "Consistency (the whole corpus, or one intent id)"

itd-53 names its list of shipped intents whose fidelity review is missing or not met a consistency gate. In the intent chapter, Consistency is the Role 2 cross-document pass over the whole corpus or one intent, and the shipped-but-unreviewed listing is the owed-reviews form of the audit sub-verb. One name covers two concepts, so a reader cannot tell which surface consistency means.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 553.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The re-scope says the gate reports and never blocks and hands the standing list to another record, so the consistency gate wording is read through the banner.
