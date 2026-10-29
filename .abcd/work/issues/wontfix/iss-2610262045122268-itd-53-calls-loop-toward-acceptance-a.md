---
schema_version: 1
id: "iss-2610262045122268"
slug: "itd-53-calls-loop-toward-acceptance-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1259 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-53-review-queue-auto-drain-fidelity-gate.md:44"
remedy: "none (filed automatically)"
related_intents: [itd-53, itd-50]
finding_ref: cfk-13d88d26eaa06335
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The shipped record's banner names itd-50 as build's, so its out-of-scope line about an elected policy riding on a drainer is read through it; itd-50 is still planned."
---

itd-53 calls loop-toward-acceptance a facilitator-elected policy riding on its drainer; itd-50 ruled the loop is build's last stage, not a per-intent mode

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-53-review-queue-auto-drain-fidelity-gate.md:44` — "The facilitator-elected loop-toward-acceptance policy"
- End B: `.abcd/development/intents/planned/itd-50-loop-toward-acceptance.md:98` — "not a per-intent mode; every intent the run ships goes through it."

itd-53 defers the facilitator-elected loop-toward-acceptance policy to itd-50 and says it rides on itd-53's drainer. itd-50's decision 1 makes the loop the last stage of `build`, not a per-intent mode, with every intent the run ships going through it. A loop that every shipped intent goes through is not one the facilitator elects per intent, and it does not depend on itd-53's drainer.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1259.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The shipped record's banner names itd-50 as build's, so its out-of-scope line about an elected policy riding on a drainer is read through it; itd-50 is still planned.
