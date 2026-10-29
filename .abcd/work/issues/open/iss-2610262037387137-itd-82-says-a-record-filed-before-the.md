---
schema_version: 1
id: "iss-2610262037387137"
slug: "itd-82-says-a-record-filed-before-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 738 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-82-drain-ledger-triage.md:44"
remedy: "In itd-82's remedy bullet, say a record filed before the field reads its older suggested_fix: as its remedy and is eligible on that, as 35-drain.md and the code do."
related_intents: [itd-82]
finding_ref: cfk-2b96752b08a4f98c
---

itd-82 says a record filed before the remedy rule has no remedy and is ineligible; 35-drain reads such a record's legacy suggested_fix as its remedy

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-82-drain-ledger-triage.md:44` — "a record filed before the rule carries none, and is ineligible and listed as such"
- End B: `.abcd/development/brief/04-surfaces/35-drain.md:75` — "record written before the field existed reads its `suggested_fix:` as its remedy."

In itd-82 a remedy is only the remedy: field, so a record filed before the rule carries none and is ineligible and listed as such until a person adds one. 35-drain uses remedy in a wider sense: a record written before the field existed has its suggested_fix: read as its remedy, so it can pass the remedy test and be drained. A builder cannot follow both for the older backlog. Under one reading those records are listed as ineligible, and under the other they are worked unattended.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 738.
