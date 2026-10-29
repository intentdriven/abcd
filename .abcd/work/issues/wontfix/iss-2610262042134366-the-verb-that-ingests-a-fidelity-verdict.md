---
schema_version: 1
id: "iss-2610262042134366"
slug: "the-verb-that-ingests-a-fidelity-verdict"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1054 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-80-intent-lifecycle-automation.md:32"
remedy: "none (filed automatically)"
related_intents: [itd-80]
finding_ref: cfk-b109411096afc769
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Both the verb and the agent name were changed by one shipped rename record."
---

The verb that ingests a fidelity verdict is `abcd intent review ingest` in itd-80 and `abcd intent audit ingest` in the verification matrix.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-80-intent-lifecycle-automation.md:32` — "`abcd intent review ingest` ingests a verdict"
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:56` — "`abcd intent audit ingest` writes the returned verdict into the intent's own Audit Notes"

In itd-80, `abcd intent review ingest` ingests the Role 1 verdict that an `intent-fidelity-reviewer` agent returns into the intent's Audit Notes. The matrix's Intent lifecycle hook row writes the same returned verdict into the intent's Audit Notes with `abcd intent audit ingest`, routed through the `oracle.roles.intent-auditor` role. One act and one agent each carry two names.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1054.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Both the verb and the agent name were changed by one shipped rename record.
