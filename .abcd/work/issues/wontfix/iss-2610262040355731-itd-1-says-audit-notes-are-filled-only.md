---
schema_version: 1
id: "iss-2610262040355731"
slug: "itd-1-says-audit-notes-are-filled-only"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 941 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-1-acceptance-gates.md:45"
remedy: "none (filed automatically)"
related_intents: [itd-1]
finding_ref: cfk-251604118d89b5dd
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262037138873. Same stale sentence in itd-1 line 45, cited against 02-verification-matrix.md."
duplicates: [iss-2610262037138873]
---

itd-1 says Audit Notes are filled only by a hand-run audit; the verification matrix says the shipping close itself requests the audit and can ingest the verdict.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-1-acceptance-gates.md:45` — "Until that lands, a shipped intent's `## Audit Notes` is populated only when `/abcd:intent audit` is run by hand."
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:56` — "When `oracle.roles.intent-auditor` reaches a provider connection a hand-run close sends that request there and ingests the verdict"

itd-1 defers automatic invocation on planned to shipped, so Audit Notes are filled only when /abcd:intent audit is run by hand. The matrix's lifecycle-hook row says the close that ships an intent makes its audit owed. With a provider route, a hand-run close sends the request and ingests the verdict, and a build run ingests it at the close. Both cannot describe the same close.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 941.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262037138873. Same stale sentence in itd-1 line 45, cited against 02-verification-matrix.md.
