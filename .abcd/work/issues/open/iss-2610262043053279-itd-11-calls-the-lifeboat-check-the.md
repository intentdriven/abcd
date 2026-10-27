---
schema_version: 1
id: "iss-2610262043053279"
slug: "itd-11-calls-the-lifeboat-check-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1114 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-11-pass-b-pitfall-mitigation.md:15"
remedy: "none (filed automatically)"
related_intents: [itd-11, itd-125]
finding_ref: cfk-7f05cfd9966d06ed
---

itd-11 calls the lifeboat check 'the lifeboat-oracle's audit'; itd-125 consolidated that oracle/audit/review triple into 'review'.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-11-pass-b-pitfall-mitigation.md:15` — "the lifeboat-oracle's audit catches and quarantines low-confidence Pass B output"
- End B: `.abcd/development/intents/shipped/itd-125-the-lifeboat-verdict-says-what-it-is.md:22` — "produced by an oracle, holding a review verdict — three words for one thing."

itd-11 has 'the lifeboat-oracle's audit' catch and quarantine low-confidence Pass B output. itd-125 records that the lifeboat verdict 'used to be called an audit, produced by an oracle, holding a review verdict' and that the verb, flag, artefact and prose now all say review. The draft still uses two of the three retired words for the same check, so a builder would reintroduce the naming itd-125 removed.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1114.
