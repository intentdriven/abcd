---
schema_version: 1
id: "iss-2610262035104777"
slug: "itd-53-s-press-release-has-an-opt-in"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 552 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-53-review-queue-auto-drain-fidelity-gate.md:22"
remedy: "none (filed automatically)"
related_intents: [itd-53]
finding_ref: cfk-521fc743d3d347c3
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The record's banner says the autodrain was dropped and its press release is read through the re-scope; no hook or schedule drains the owed queue in the code."
---

itd-53's press release has an opt-in drainer run owed reviews at loop ends, session edges and pre-commit or CI steps; the intent chapter says nothing starts the drain on its own

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-53-review-queue-auto-drain-fidelity-gate.md:22` — "This intent adds an opt-in drainer that runs queued reviews at a safe boundary (after a loop, at a session edge, in a pre-commit or CI step — never inside the pure close hook)"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:328` — "Nothing starts the drain on its own: no hook, gate or schedule"

itd-53 announces a drainer that runs queued reviews automatically at a safe boundary: after a loop, at a session edge, or in a pre-commit or CI step. The intent chapter says nothing starts the drain on its own, with no hook, gate or schedule. Automatic boundary-triggered draining and a drain that only runs when asked are incompatible descriptions of one surface.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 552.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The record's banner says the autodrain was dropped and its press release is read through the re-scope; no hook or schedule drains the owed queue in the code.
