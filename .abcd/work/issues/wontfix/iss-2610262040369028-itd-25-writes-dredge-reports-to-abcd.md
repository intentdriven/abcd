---
schema_version: 1
id: "iss-2610262040369028"
slug: "itd-25-writes-dredge-reports-to-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 943 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:35"
remedy: "none (filed automatically)"
related_intents: [itd-25]
finding_ref: cfk-ff81dc755ec5fb4a
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031301413. Same draft edit as iss-2610262031301413, set against a different statement of the same current state."
duplicates: [iss-2610262031301413]
---

itd-25 writes dredge reports to .abcd/logbook/dredge/; the memory chapter puts them in .abcd/.work.local/logs/dredge/.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:35` — "writes `synthesis-report.{json,md}` to `.abcd/logbook/dredge/<timestamp>/`"
- End B: `.abcd/development/brief/05-internals/07-memory.md:174` — "per-run report stays in the gitignored local-ephemeral tier at `.abcd/.work.local/logs/dredge/<ts>/`"

itd-25 has dredge synth write synthesis-report.{json,md} to .abcd/logbook/dredge/< timestamp>/ and has dredge list read them from there. 07-memory says the dredge per-run report stays in the gitignored local tier at .abcd/.work.local/logs/dredge/< ts>/, which replaced the retired location. The same report cannot have both homes.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 943.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031301413. Same draft edit as iss-2610262031301413, set against a different statement of the same current state.
