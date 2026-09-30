---
schema_version: 1
id: "iss-2608301747006182"
slug: "no-gate-requires-a-terminal-folder-record-to-carry-a-grounds"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "itd-179-round-5-builder"
found_at: "internal/core/lint"
remedy: "Waits on ruling BY1 (the cutover): add a record-lint rule in `internal/core/lint` refusing a record in resolved/ or wontfix/ with no Grounds section, scoped by the answer: (a) records minted from 2026-08-31, read from the id's timestamp, after backfilling iss-2609091009111294; (b) records closed after a named release; (c) no rule, and this record is wontfixed; proven by a test that a pre-cutover record without grounds passes and a post-cutover one is refused."
deferred_after: "v0.11.1"
deferral_reason: "cutover ruling F owed to the product thinker: 690 of 1663 resolved records and 17 of 46 wontfix records carry no ## Grounds section, so a gate cannot be armed without a cutover. Measured for the ruling: every terminal record minted from 2026-08-31 onwards carries grounds except iss-2609091009111294, so a mint-date cutover at 2026-08-31 would be green after one backfill. Which cutover applies (mint date, resolve date or a named release) is the product thinker's. (re-checked at e792a2314 by lane drainDQ3, run A, 2026-09-29)"
---

no gate requires a terminal folder record to carry a grounds entry so fourteen resolved records have none

Reported by the round-5 builder. PRE-EXISTING in substance -- the old rule
checked the grammar of a `grounds` value only if one was present -- but far
more visible now that grounds is a section rather than an optional key.

A gate demanding one would be red today: the fourteen records that reached
`resolved/` before grounds existed carry none, and nothing backfills them. So
the gate cannot simply be armed; it needs a cutover rule saying from when the
obligation runs.

## Remedy grounds (2026-09-29)

- The deferral measured 690 of 1663 resolved and 17 of 46 wontfix records without grounds, and one post-cutover exception, so option (a) is green after one backfill; ordinal ids such as iss-59 carry no timestamp and predate the cutover, so they pass by construction.
- No outside-practice check: the question is a cutover date for an internal rule.
- Rejected: arming the rule without a cutover, which would refuse hundreds of settled records.
