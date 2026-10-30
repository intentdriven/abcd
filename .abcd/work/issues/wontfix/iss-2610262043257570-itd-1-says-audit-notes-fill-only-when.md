---
schema_version: 1
id: "iss-2610262043257570"
slug: "itd-1-says-audit-notes-fill-only-when"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1136 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-1-acceptance-gates.md:45"
remedy: "none (filed automatically)"
related_intents: [itd-1, itd-2609081951381895]
finding_ref: cfk-84799b8e95ee72b4
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262037138873. Same stale sentence in itd-1 line 45, cited against the shipped API-adapter intent."
duplicates: [iss-2610262037138873]
---

itd-1 says Audit Notes fill only when the audit is run by hand; itd-2609081951381895 has spec close send the review and ingest the answer

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-1-acceptance-gates.md:45` — "is populated only when `/abcd:intent audit` is run by hand"
- End B: `.abcd/development/intents/shipped/itd-2609081951381895-abcd-ships-an-openai-compatible-api.md:77` — "sends the review it emits when the auditor is routed to a provider"

itd-1 states that automatic invocation on the planned-to-shipped transition is deferred, so a shipped intent's Audit Notes is populated only when the audit is run by hand. The shipped API-adapter intent rules that spec close sends the review it emits when the auditor is routed to a provider and runs the answer through its own ingest, which writes the verdict at the transition with no hand step. Both cannot describe the current lifecycle.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1136.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262037138873. Same stale sentence in itd-1 line 45, cited against the shipped API-adapter intent.
