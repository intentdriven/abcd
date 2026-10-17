---
schema_version: 1
id: "iss-2610160925028248"
slug: "abcd-reserve-holds-one-record-per"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "v0.15.0 deferred findings, reserving PR3, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "abcd reserve"
remedy: "Let one reservation name several records (one hold, one footprint, one lease, released together), or count holds per pull request rather than per record; test that a four-record reservation takes one slot of the cap and that each named record reads as held."
---

abcd reserve holds one record per reservation and caps a person at ten live reservations across every session and copy of that account, so a small documentation pull request that fixes N records spends N of the ten: on 2026-10-16 two four-record follow-up PRs plus two intents another session held filled the cap, and every further reservation by any session of that person was refused until those PRs merged. The cap meant to bound one person's holds instead bounds how many record-sized fixes can be in flight.
