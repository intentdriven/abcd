---
schema_version: 1
id: "iss-2610262045054264"
slug: "the-agent-that-writes-per-criterion"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1250 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-50-loop-toward-acceptance.md:25"
remedy: "none (filed automatically)"
related_intents: [itd-50, itd-24]
finding_ref: cfk-dbb8d61f43dc9f3a
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031563379. Same End A; itd-24 (shipped) already names the intent auditor."
duplicates: [iss-2610262031563379]
---

The agent that writes per-criterion audit verdicts is the intent-fidelity-reviewer in itd-50 and the intent auditor in itd-24

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-50-loop-toward-acceptance.md:25` — "the `intent-fidelity-reviewer` records a `NOT_MET` verdict"
- End B: `.abcd/development/intents/shipped/itd-24-reflect-command.md:61` — "judging delivered reality against each intent's criteria is the intent auditor's job"

itd-50, still planned, builds its acceptance loop on the intent-fidelity-reviewer recording NOT_MET verdicts. itd-24 gives the same job, judging delivered reality against each intent's criteria and writing the audit notes, to the intent auditor. Two names for one agent, one of them in a record still to be built, leaves a builder unsure which agent definition the loop drives.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1250.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031563379. Same End A; itd-24 (shipped) already names the intent auditor.
