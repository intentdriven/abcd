---
schema_version: 1
id: "iss-2610262033219255"
slug: "itd-6-has-ahoy-discover-rp-and-lock"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 393 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:43"
remedy: "none (filed automatically)"
related_intents: [itd-6]
finding_ref: cfk-a9406ff69dce6b76
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The quoted bullet is the setup discovery the re-file note names as dropped; code accepts only host-delegated, native, cli, api and mcp as oracle.backend and ahoy records host-delegated without asking."
---

itd-6 has ahoy discover RP and lock `oracle.backend = "rp"`; 01-ahoy records host-delegated without asking, as the only answer with an adapter

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:43` — "and tests reachability. If reachable: lock `oracle.backend = "rp"`."
- End B: `.abcd/development/brief/04-surfaces/01-ahoy.md:893` — "The oracle backend is not asked while host-delegated is the only answer with an adapter"

itd-6 scopes a one-time RP setup discovery in ahoy that tests reachability and then locks `oracle.backend` to `rp` (or `codex` or `in-session`). 01-ahoy says the oracle backend is not asked while host-delegated is the only answer with an adapter, and the install records host-delegated. Yet itd-6's status says the RP adapter has been built. ahoy cannot both lock the backend to rp when it finds RP and treat host-delegated as the only backend with an adapter.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 393.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The quoted bullet is the setup discovery the re-file note names as dropped; code accepts only host-delegated, native, cli, api and mcp as oracle.backend and ahoy records host-delegated without asking.
