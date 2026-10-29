---
schema_version: 1
id: "iss-2610262038565943"
slug: "itd-25-writes-dredge-reports-under-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 828 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:35"
remedy: "none (filed automatically)"
related_intents: [itd-25]
finding_ref: cfk-d049c6d7ada56bed
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031301413. Same draft edit as iss-2610262031301413, set against a different statement of the same current state."
duplicates: [iss-2610262031301413]
---

itd-25 writes dredge reports under .abcd/logbook/dredge/; 04-universal-patterns puts dredge output in the local tier's logs

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:35` — "writes `synthesis-report.{json,md}` to `.abcd/logbook/dredge/<timestamp>/`"
- End B: `.abcd/development/brief/05-internals/04-universal-patterns.md:142` — "`dredge/<timestamp>/` — cross-corpus synthesis output (itd-25)"

itd-25's dredge synth writes its synthesis report to .abcd/logbook/dredge/< timestamp>/. 04-universal-patterns lists dredge/< timestamp>/ as a later addition under .abcd/.work.local/logs/, and says the logbook name is retired. The two documents put the same output in different trees.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 828.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031301413. Same draft edit as iss-2610262031301413, set against a different statement of the same current state.
