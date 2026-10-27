---
schema_version: 1
id: "iss-2610262033596296"
slug: "itd-196-keys-its-cumulative-and-closing"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 461 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-196-a-capability-is-rehearsed-end-to-end.md:36"
remedy: "none (filed automatically)"
related_intents: [itd-196]
finding_ref: cfk-476366dbfe40d769
---

itd-196 keys its cumulative and closing rehearsals on the phase; the reflect surface says the phase was retired and the release is the grain

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-196-a-capability-is-rehearsed-end-to-end.md:36` — "One full run of everything the phase set out to"
- End B: `.abcd/development/brief/04-surfaces/09-reflect.md:14` — "retired the phase; itd-24 decisions 4 and 5)"

itd-196 sets its second and third rehearsal rungs on the phase, ending with one full run of everything the phase set out to deliver. The reflect surface records that adr-2609212115255771 retired the phase and makes the release the unit a delivery is closed on. The discipline's rungs are keyed on a unit the brief says no longer exists, so when its closing rung is owed is undefined.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 461.
