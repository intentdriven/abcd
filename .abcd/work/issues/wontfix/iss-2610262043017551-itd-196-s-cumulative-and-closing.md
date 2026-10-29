---
schema_version: 1
id: "iss-2610262043017551"
slug: "itd-196-s-cumulative-and-closing"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1109 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-196-a-capability-is-rehearsed-end-to-end.md:32"
remedy: "none (filed automatically)"
related_intents: [itd-196, itd-24]
finding_ref: cfk-20e5dde37a9867c7
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262042291241. Same unit problem in itd-196 as iss-2610262042291241, cited against itd-24."
duplicates: [iss-2610262042291241]
---

itd-196's cumulative and closing rehearsal rungs are defined over a phase; itd-24 records that phases are retired.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-196-a-capability-is-rehearsed-end-to-end.md:32` — "Run what the PHASE can deliver so far"
- End B: `.abcd/development/intents/shipped/itd-24-reflect-command.md:90` — "phases are retired, so `<phase-id>` reads as the release tag"

itd-196 defines its second rung as running 'what the PHASE can deliver so far' and its third as one full run 'at the phase's close', and calls the cumulative rung what turns a phase into more than a list of merged intents. itd-24 records that phases are retired and the release is the unit. The discipline's two most important rungs are keyed on a unit that no longer exists, so when the cumulative run starts, and what the closing run must cover, are undefined.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1109.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262042291241. Same unit problem in itd-196 as iss-2610262042291241, cited against itd-24.
