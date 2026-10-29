---
schema_version: 1
id: "iss-2610262043089096"
slug: "the-delegated-payload-draft-refuses-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1117 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-2609180517121254-every-payload-a-host-hands-back-from-a.md:15"
remedy: "none (filed automatically)"
related_intents: [itd-2609180517121254, itd-166]
finding_ref: cfk-463bc206bdf2ee90
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. internal/core/oracle/receipt.go records ModelReported as empty when a payload names none and says requiring it is itd-2609180517121254's, so the code already treats the refusal as the draft's future change; the draft is a one-line seed with no criteria yet."
---

The delegated-payload draft refuses a payload that names no model; itd-166 records a missing served model as a reason, never leaving it out.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-2609180517121254-every-payload-a-host-hands-back-from-a.md:15` — "and the ingesting verb refuses one that does not"
- End B: `.abcd/development/intents/shipped/itd-166-each-run-should-record-the-model.md:18` — "Whatever the route cannot show is written down as unpinned or self-reported, never left out and never guessed."

itd-2609180517121254 requires every payload a host hands back from a delegated step to name its model, and 'the ingesting verb refuses one that does not'. itd-166, which covers a role run through the host as well as runners and providers, says 'whatever the route cannot show is written down as unpinned or self-reported, never left out', and its decisions record an absent served model as 'not reported' rather than failing the call. For the same case, a host-delegated answer that names no model, one record refuses the ingest and the other records the gap and goes on.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1117.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. internal/core/oracle/receipt.go records ModelReported as empty when a payload names none and says requiring it is itd-2609180517121254's, so the code already treats the refusal as the draft's future change; the draft is a one-line seed with no criteria yet.
