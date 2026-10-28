---
schema_version: 1
id: "iss-2610262037211310"
slug: "itd-19-s-per-stage-defaults-set-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 716 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-19-stage-aware-behaviour.md:31"
remedy: "none (filed automatically)"
related_intents: [itd-19]
finding_ref: cfk-93ca02f0c8aa300c
---

itd-19's per-stage defaults set the oracle to in-session, rp or rp+codex; the oracle.review key accepts only host or rp.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-19-stage-aware-behaviour.md:31` — "oracle=in-session (fast iteration)"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:544` — "`oracle.review` in the layered configuration names `host`, the bundled default, or `rp`"

34-build gives the review route's values in .abcd/config.json as host (the bundled default) or rp, and refuses any fault in oracle.review before anything moves. itd-19 calls the same host route in-session and adds rp+codex. A stage default written from the draft would use a second name for the default route and a value the reader refuses.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 716.
