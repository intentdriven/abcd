---
schema_version: 1
id: "iss-2610262037413056"
slug: "35-drain-counts-spend-only-in-agent-runs"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 741 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/35-drain.md:606"
remedy: "In 35-drain.md, replace 'No runner reports tokens or money, so neither is counted' with the true reason: the spend is counted in agent runs because that is the unit common to host-run and runner-run lanes, while a runner-routed run carries its tokens and cost in its own run record (itd-166)."
related_intents: [itd-166]
finding_ref: cfk-b271cc6b5dcea03a
---

35-drain counts spend only in agent runs because no runner reports tokens or money; itd-166 has every route record token counts and cost

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/35-drain.md:606` — "reports tokens or money, so neither is counted."
- End B: `.abcd/development/intents/shipped/itd-166-each-run-should-record-the-model.md:18` — "then the model, provider, token counts and cost the answer reported"

35-drain counts the drain's spend in agent runs and says no runner reports tokens or money, so neither is counted. Shipped itd-166 says every role run through the host, a command-line runner or a provider connection records the model, provider, token counts and cost. For the claude and opencode runners the cost is computed by the harness. The drain summary's stated reason for leaving tokens and cost out is therefore false of the delivered runner record.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 741.
