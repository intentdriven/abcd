---
schema_version: 1
id: "iss-2610262034027543"
slug: "itd-16-writes-the-chain-verification"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 465 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-16-hash-chain-merkle-audit.md:50"
remedy: "none (filed automatically)"
related_intents: [itd-16]
finding_ref: cfk-51b28cfc2a2db28f
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031298011. Same draft edit as iss-2610262031298011, set against a different statement of the same current state."
duplicates: [iss-2610262031298011]
---

itd-16 writes the chain verification report under .abcd/logbook/audit/; the intent surface puts the chain sub-verb's receipts in the local ephemeral logs tier

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-16-hash-chain-merkle-audit.md:50` — "writes a report to `.abcd/logbook/audit/chain-<ts>/report.{json,md}`"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:653` — "The later-phase review/audit verbs write their per-run receipts under the local ephemeral logs tier, `.abcd/.work.local/logs/audit/<sub-tier>-<ts>/`"

itd-16 has the chain sub-verb write its verification report to .abcd/logbook/audit/chain-< ts>/. The intent surface's audit-trail layout says the later-phase audit verbs, the chain sub-verb among them, write their per-run receipts under .abcd/.work.local/logs/audit/< sub-tier>-< ts>/. The same output cannot live in both places, which sit in different tiers.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 465.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031298011. Same draft edit as iss-2610262031298011, set against a different statement of the same current state.
