---
schema_version: 1
id: "iss-2610262037410328"
slug: "27-implement-says-the-model-service-can"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 742 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/27-implement.md:508"
remedy: "none (filed automatically)"
related_intents: [itd-166]
finding_ref: cfk-3879fcdbbccd694c
---

27-implement says the model service can only be probed through a canary agent because abcd calls no model; itd-166 records every model call abcd makes

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/27-implement.md:508` — "only by the verdict of a canary agent the lead runs, since abcd calls no model."
- End B: `.abcd/development/intents/shipped/itd-166-each-run-should-record-the-model.md:18` — "Every model call abcd makes now records two sides"

27-implement limits the outage probe of the model service to a canary agent the lead runs, on the ground that abcd calls no model. The same page says abcd runs no agent. Shipped itd-166 says abcd makes model calls through provider connections and runners, and records two sides for each. Its adapter streams usage from the provider, and the runners are launched by abcd. The premise the probe design rests on no longer holds for the delivered surface.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 742.
