---
schema_version: 1
id: "iss-2610262045442294"
slug: "itd-80-names-the-verdict-ingest-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1294 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-80-intent-lifecycle-automation.md:32"
remedy: "none (filed automatically)"
related_intents: [itd-80, itd-2609150819445595]
finding_ref: cfk-a7665b08d5cfdf7f
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-80's review ingest spelling predates the rename recorded in itd-123 and adr-40."
---

itd-80 names the verdict ingest abcd intent review ingest; itd-2609150819445595 names the same review family abcd intent audit

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-80-intent-lifecycle-automation.md:32` — "`abcd intent review ingest` ingests a verdict"
- End B: `.abcd/development/intents/shipped/itd-2609150819445595-a-debt-nothing-lists-owed-fidelity.md:42` — "`abcd intent audit <itd-N>` re-emits it on demand"

itd-80 puts the fidelity-verdict ingest at abcd intent review ingest. itd-2609150819445595 has the owed-review listing at bare abcd intent audit and the request re-emitted by abcd intent audit < itd-N>. Two verb names cover one fidelity-review family, so a caller following one record runs a verb the other does not name.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1294.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-80's review ingest spelling predates the rename recorded in itd-123 and adr-40.
