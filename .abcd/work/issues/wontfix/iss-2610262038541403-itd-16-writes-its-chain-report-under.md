---
schema_version: 1
id: "iss-2610262038541403"
slug: "itd-16-writes-its-chain-report-under"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 826 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-16-hash-chain-merkle-audit.md:50"
remedy: "none (filed automatically)"
related_intents: [itd-16]
finding_ref: cfk-b0009f7785bcbaf6
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031298011. Same draft edit as iss-2610262031298011, set against a different statement of the same current state."
duplicates: [iss-2610262031298011]
---

itd-16 writes its chain report under .abcd/logbook/; 03-configuration says logbook/ is retired in favour of the local tier's logs

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-16-hash-chain-merkle-audit.md:50` — "writes a report to `.abcd/logbook/audit/chain-<ts>/report.{json,md}`"
- End B: `.abcd/development/brief/05-internals/03-configuration.md:1368` — "`logbook/`, retired in favour of the local tier's logs"

itd-16's /abcd:audit chain writes its verification report to .abcd/logbook/audit/chain-< ts>/. 03-configuration says logbook/ was retired in favour of the local tier's logs. The draft puts its output in a namespace the configuration model has removed.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 826.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031298011. Same draft edit as iss-2610262031298011, set against a different statement of the same current state.
